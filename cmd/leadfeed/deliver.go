package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	mongorepo "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
)

const (
	// callTimeout bounds one lead-task call; the call itself waits up to ten seconds for the mail.
	callTimeout = 30 * time.Second
	// killGrace lets a killed call's pipes close, so a hung grandchild cannot hold the run.
	killGrace = 2 * time.Second
)

func runDeliverCommand(
	ctx context.Context,
	args []string,
	mongoURI, mongoDB string,
	stdout, stderr io.Writer,
	usage func(),
) int {
	flags := flag.NewFlagSet("leadfeed deliver", flag.ContinueOnError)
	flags.SetOutput(stderr)
	leadTask := flags.String("lead-task", "", "path of the lead-task executable")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *leadTask == "" {
		usage()
		return exitUsage
	}
	return withRepo(
		ctx,
		mongoURI,
		mongoDB,
		stderr,
		func(repo *mongorepo.MongoLeadFeedRepository, db *mongo.Database) int {
			sender := execSender{path: *leadTask}
			deliverer := leadfeed.NewDeliverer(repo, mongorepo.NewMongoOwnerAnswerRepository(db), sender, repo, nil)
			return runDeliver(ctx, deliverer, stdout, stderr)
		},
	)
}

func runDeliver(ctx context.Context, deliverer *leadfeed.Deliverer, stdout, stderr io.Writer) int {
	result, err := deliverer.Run(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "leadfeed: deliver: %v\n", err)
		return exitFailed
	}
	fmt.Fprintf(stdout, "sent=%d failed=%d unconfirmed=%d other_epoch=%d\n",
		result.Sent, result.Failed, result.Unconfirmed, result.OtherEpoch)
	if result.Failed > 0 {
		fmt.Fprintf(stderr, "leadfeed: %d answers not delivered: %s\n", result.Failed, result.LastError)
		return exitFailed
	}
	return exitOK
}

type execSender struct {
	path string
}

func (s execSender) SendAnswer(ctx context.Context, qid string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, s.path, "answer", qid) //nolint:gosec // the path is the operator's own flag
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stderr = &stderr
	cmd.WaitDelay = killGrace

	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		return fmt.Errorf("lead-task answer: %w", ctx.Err())
	case errors.As(err, &exitErr):
		return fmt.Errorf("lead-task answer exited with %d: %s", exitErr.ExitCode(), strings.TrimSpace(stderr.String()))
	default:
		return fmt.Errorf("lead-task answer: %w", err)
	}
}
