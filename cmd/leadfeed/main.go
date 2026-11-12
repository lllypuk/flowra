// Command leadfeed imports the lead's task snapshot into the read-only feed and reports its state.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	mongorepo "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
)

// Exit codes; a timer or a script tells "nothing changed" from "result unknown" by them.
const (
	exitOK            = 0
	exitFailed        = 1
	exitUsage         = 2
	exitCommitUnknown = 3
)

const (
	defaultMongoURI = "mongodb://localhost:27017/?replicaSet=rs0"
	defaultMongoDB  = "flowra"

	connectTimeout    = 10 * time.Second
	disconnectTimeout = 5 * time.Second
	stdinPath         = "-"

	// timeLayout matches the fixed milliseconds of the source and of the feed API.
	timeLayout = "2006-01-02T15:04:05.000Z"
)

const usage = `usage:
  leadfeed [flags] import [file|-]   publish a snapshot (stdin by default)
  leadfeed [flags] status            show the published snapshot and the last attempt
  leadfeed [flags] fail <reason>     record that the source could not produce a snapshot

flags:
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("leadfeed", flag.ContinueOnError)
	flags.SetOutput(stderr)
	mongoURI := flags.String("mongo-uri", envOr("MONGODB_URI", defaultMongoURI), "MongoDB URI (replica set required)")
	mongoDB := flags.String("mongo-db", envOr("MONGODB_DATABASE", defaultMongoDB), "MongoDB database")
	maxBytes := flags.Int64("max-bytes", leadfeed.DefaultMaxSnapshotBytes, "largest snapshot accepted, in bytes")
	flags.Usage = func() {
		fmt.Fprint(stderr, usage)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	rest := flags.Args()
	if len(rest) == 0 {
		flags.Usage()
		return exitUsage
	}

	switch rest[0] {
	case "import":
		if len(rest) > 2 { //nolint:mnd // command plus one optional path
			flags.Usage()
			return exitUsage
		}
		path := stdinPath
		if len(rest) == 2 { //nolint:mnd // command plus path
			path = rest[1]
		}
		return withRepo(ctx, *mongoURI, *mongoDB, stderr, func(repo *mongorepo.MongoLeadFeedRepository) int {
			return runImport(ctx, repo, path, *maxBytes, stdin, stdout, stderr)
		})
	case "status":
		if len(rest) != 1 {
			flags.Usage()
			return exitUsage
		}
		return withRepo(ctx, *mongoURI, *mongoDB, stderr, func(repo *mongorepo.MongoLeadFeedRepository) int {
			return runStatus(ctx, repo, stdout, stderr)
		})
	case "fail":
		if len(rest) != 2 { //nolint:mnd // command plus reason
			flags.Usage()
			return exitUsage
		}
		return withRepo(ctx, *mongoURI, *mongoDB, stderr, func(repo *mongorepo.MongoLeadFeedRepository) int {
			return runFail(ctx, repo, rest[1], stderr)
		})
	default:
		fmt.Fprintf(stderr, "leadfeed: unknown command %q\n", rest[0])
		flags.Usage()
		return exitUsage
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func withRepo(
	ctx context.Context,
	uri, dbName string,
	stderr io.Writer,
	fn func(repo *mongorepo.MongoLeadFeedRepository) int,
) int {
	client, err := mongo.Connect(options.Client().SetServerSelectionTimeout(connectTimeout).ApplyURI(uri))
	if err != nil {
		fmt.Fprintf(stderr, "leadfeed: connect to MongoDB: %v\n", err)
		return exitFailed
	}
	defer func() {
		// Detached from ctx: after an interrupt the connection still has to be closed.
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), disconnectTimeout)
		defer cancel()
		_ = client.Disconnect(closeCtx)
	}()

	db := client.Database(dbName)
	// The API creates indexes only at its own startup, and the importer may run before it ever starts.
	for _, coll := range []string{mongodbinfra.CollectionLeadFeedTasks, mongodbinfra.CollectionLeadFeedRecords} {
		if err = mongodbinfra.CreateCollectionIndexes(ctx, db, coll); err != nil {
			fmt.Fprintf(stderr, "leadfeed: ensure indexes: %v\n", err)
			return exitFailed
		}
	}

	return fn(mongorepo.NewMongoLeadFeedRepository(client, db))
}

func runImport(
	ctx context.Context,
	repo *mongorepo.MongoLeadFeedRepository,
	path string,
	maxBytes int64,
	stdin io.Reader,
	stdout, stderr io.Writer,
) int {
	input := stdin
	if path != stdinPath {
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(stderr, "leadfeed: %v\n", err)
			return exitFailed
		}
		defer file.Close()
		input = file
	}

	importer := leadfeed.NewImporter(repo, leadfeed.WithMaxSnapshotBytes(maxBytes))
	result, err := importer.Import(ctx, input)
	if errors.Is(err, leadfeed.ErrCommitUnknown) {
		fmt.Fprintf(stderr, "leadfeed: result unknown, the feed holds the old or the new snapshot in full: %v\n", err)
		return exitCommitUnknown
	}
	if err != nil {
		fmt.Fprintf(stderr, "leadfeed: snapshot not published: %v\n", err)
		return exitFailed
	}

	fmt.Fprintf(stdout, "%s revision=%s as_of=%s tasks=%d records=%d\n",
		result.Outcome, result.Revision, result.AsOf.UTC().Format(timeLayout), result.Tasks, result.Records)
	if result.PreviousEpoch != "" {
		fmt.Fprintf(stdout, "source journal changed: previous epoch %s\n", result.PreviousEpoch)
	}
	return exitOK
}

// runFail makes a source-side failure visible in the feed status; the published snapshot is not touched.
func runFail(ctx context.Context, repo *mongorepo.MongoLeadFeedRepository, reason string, stderr io.Writer) int {
	err := repo.RecordAttempt(ctx, leadfeed.Attempt{At: time.Now(), Outcome: leadfeed.AttemptFailed, Error: reason})
	if err != nil {
		fmt.Fprintf(stderr, "leadfeed: %v\n", err)
		return exitFailed
	}
	return exitOK
}

func runStatus(ctx context.Context, repo *mongorepo.MongoLeadFeedRepository, stdout, stderr io.Writer) int {
	var (
		state   *leadfeed.State
		attempt *leadfeed.Attempt
	)
	err := repo.Read(ctx, func(readCtx context.Context, view leadfeed.View) error {
		var readErr error
		if state, readErr = view.State(readCtx); readErr != nil {
			return readErr
		}
		attempt, readErr = view.LastAttempt(readCtx)
		return readErr
	})
	if err != nil {
		fmt.Fprintf(stderr, "leadfeed: read feed state: %v\n", err)
		return exitFailed
	}

	if state == nil {
		fmt.Fprintln(stdout, "published: nothing yet")
	} else {
		fmt.Fprintf(stdout, "published: revision=%s as_of=%s tasks=%d records=%d source=%s epoch=%s at=%s\n",
			state.Revision, state.AsOf.Format(timeLayout), state.Tasks, state.Records,
			state.Source, state.Epoch, state.PublishedAt.Format(timeLayout))
	}

	if attempt == nil {
		fmt.Fprintln(stdout, "last attempt: none")
		return exitOK
	}
	fmt.Fprintf(stdout, "last attempt: %s at=%s revision=%s\n",
		attempt.Outcome, attempt.At.Format(timeLayout), attempt.Revision)
	if attempt.Error != "" {
		fmt.Fprintf(stdout, "last error: %s\n", attempt.Error)
	}
	return exitOK
}
