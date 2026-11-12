// Package leadfeed is the read-only projection of the lead's task journal.
// It is fed by full snapshots and never goes through the Chat aggregate or tag processing.
package leadfeed

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"time"
)

// SchemaVersion is the only snapshot schema this build accepts.
const SchemaVersion = 1

const (
	lineHeader = "header"
	lineTask   = "task"
	lineRecord = "record"
	lineFooter = "footer"

	maxLineBytes    = 4 << 20
	readBufferBytes = 64 << 10
	revisionHexLen  = 16
)

// Snapshot rejection reasons; a rejected snapshot must leave the published feed untouched.
var (
	ErrSnapshotMalformed  = errors.New("snapshot is malformed")
	ErrSnapshotSchema     = errors.New("snapshot schema is not supported")
	ErrSnapshotIncomplete = errors.New("snapshot is marked incomplete by its source")
	ErrSnapshotTruncated  = errors.New("snapshot has no footer")
	ErrSnapshotRevision   = errors.New("snapshot content does not match its revision")
)

// Header identifies a snapshot: source, epoch and revision name one immutable content set.
type Header struct {
	Schema     int       `json:"schema"`
	Source     string    `json:"source"`
	Epoch      string    `json:"epoch"`
	Revision   string    `json:"revision"`
	AsOf       time.Time `json:"as_of"`
	Derivation int       `json:"derivation"`
	Complete   bool      `json:"complete"`
	Errors     []string  `json:"errors"`
}

// Spent is time per waiting band in whole seconds, computed by the source as of Header.AsOf.
type Spent struct {
	Queued int64 `json:"queued"`
	Work   int64 `json:"work"`
	Lead   int64 `json:"lead"`
	User   int64 `json:"user"`
}

// Transition is one status change; Derived marks a change the source inferred rather than recorded.
type Transition struct {
	At      time.Time `json:"at"`
	Status  string    `json:"status"`
	Cause   *string   `json:"cause"`
	Derived bool      `json:"derived"`
}

// Task is a lead task with its source status kept verbatim.
type Task struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Project     string       `json:"project"`
	Agent       string       `json:"agent"`
	CreatedAt   time.Time    `json:"created_at"`
	Status      string       `json:"status"`
	WaitingOn   *string      `json:"waiting_on"`
	StatusSince time.Time    `json:"status_since"`
	Note        string       `json:"note"`
	Returns     int          `json:"returns"`
	Spent       Spent        `json:"spent"`
	Transitions []Transition `json:"transitions"`
}

// Source points at where the lead tool found a record.
type Source struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// Record is one thread entry. ID is stable across snapshots; Order is not.
type Record struct {
	ID           string    `json:"id"`
	Task         string    `json:"task"`
	Kind         string    `json:"kind"`
	At           time.Time `json:"at"`
	Order        int       `json:"order"`
	Actor        string    `json:"actor"`
	ActorDerived bool      `json:"actor_derived"`
	Source       Source    `json:"source"`
	Body         *string   `json:"body"`
	BodyDerived  bool      `json:"body_derived"`
	BodyMissing  *string   `json:"body_missing"`
	Format       string    `json:"format"`
}

// Snapshot is a fully validated export.
type Snapshot struct {
	Header  Header
	Tasks   []Task
	Records []Record
}

type footer struct {
	Revision string `json:"revision"`
	Tasks    int    `json:"tasks"`
	Records  int    `json:"records"`
}

// ParseSnapshot reads a JSONL export and returns it only if it is complete and self-consistent.
func ParseSnapshot(r io.Reader) (*Snapshot, error) {
	p := &parser{in: bufio.NewReaderSize(r, readBufferBytes), sum: sha256.New()}

	if err := p.readHeader(); err != nil {
		return nil, err
	}
	if err := p.readBody(); err != nil {
		return nil, err
	}
	if err := p.check(); err != nil {
		return nil, err
	}

	return &p.snap, nil
}

type parser struct {
	in     *bufio.Reader
	sum    hash.Hash
	hashed int
	lineNo int
	snap   Snapshot
	foot   *footer
}

// next returns the next non-empty line without its newline, or io.EOF.
func (p *parser) next() ([]byte, error) {
	for {
		line, err := p.in.ReadBytes('\n')
		if len(line) > maxLineBytes {
			return nil, fmt.Errorf("%w: line %d is longer than %d bytes", ErrSnapshotMalformed, p.lineNo+1, maxLineBytes)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read snapshot: %w", err)
		}
		line = bytes.TrimRight(line, "\r\n")
		if len(line) > 0 {
			p.lineNo++
			return line, nil
		}
		if err != nil {
			return nil, io.EOF
		}
	}
}

func (p *parser) lineType(line []byte) (string, error) {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		return "", fmt.Errorf("%w: line %d: %w", ErrSnapshotMalformed, p.lineNo, err)
	}
	return probe.Type, nil
}

func (p *parser) decode(line []byte, dst any) error {
	if err := json.Unmarshal(line, dst); err != nil {
		return fmt.Errorf("%w: line %d: %w", ErrSnapshotMalformed, p.lineNo, err)
	}
	return nil
}

func (p *parser) readHeader() error {
	line, err := p.next()
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: empty input", ErrSnapshotTruncated)
	}
	if err != nil {
		return err
	}

	kind, err := p.lineType(line)
	if err != nil {
		return err
	}
	if kind != lineHeader {
		return fmt.Errorf("%w: first line is %q, want header", ErrSnapshotMalformed, kind)
	}

	h := &p.snap.Header
	if err = p.decode(line, h); err != nil {
		return err
	}
	if h.Schema != SchemaVersion {
		return fmt.Errorf("%w: schema %d, want %d", ErrSnapshotSchema, h.Schema, SchemaVersion)
	}
	if h.Source == "" || h.Epoch == "" || h.Revision == "" || h.AsOf.IsZero() {
		return fmt.Errorf("%w: header needs source, epoch, revision and as_of", ErrSnapshotMalformed)
	}
	if !h.Complete || len(h.Errors) > 0 {
		return fmt.Errorf("%w: %v", ErrSnapshotIncomplete, h.Errors)
	}
	return nil
}

func (p *parser) readBody() error {
	for {
		line, err := p.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if p.foot != nil {
			return fmt.Errorf("%w: line %d follows the footer", ErrSnapshotMalformed, p.lineNo)
		}

		kind, err := p.lineType(line)
		if err != nil {
			return err
		}

		switch kind {
		case lineTask:
			var t Task
			if err = p.decode(line, &t); err != nil {
				return err
			}
			p.snap.Tasks = append(p.snap.Tasks, t)
			p.addToHash(line)
		case lineRecord:
			var rec Record
			if err = p.decode(line, &rec); err != nil {
				return err
			}
			p.snap.Records = append(p.snap.Records, rec)
			p.addToHash(line)
		case lineFooter:
			p.foot = &footer{}
			if err = p.decode(line, p.foot); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: line %d has unknown type %q", ErrSnapshotMalformed, p.lineNo, kind)
		}
	}

	if p.foot == nil {
		return ErrSnapshotTruncated
	}
	return nil
}

// addToHash mirrors the source: revision is sha256 over task and record lines joined by "\n".
func (p *parser) addToHash(line []byte) {
	if p.hashed > 0 {
		_, _ = p.sum.Write([]byte{'\n'})
	}
	_, _ = p.sum.Write(line)
	p.hashed++
}

func (p *parser) check() error {
	h, f := p.snap.Header, p.foot

	if f.Revision != h.Revision {
		return fmt.Errorf("%w: footer revision %q, header %q", ErrSnapshotMalformed, f.Revision, h.Revision)
	}
	if f.Tasks != len(p.snap.Tasks) || f.Records != len(p.snap.Records) {
		return fmt.Errorf("%w: footer counts %d/%d, read %d/%d",
			ErrSnapshotMalformed, f.Tasks, f.Records, len(p.snap.Tasks), len(p.snap.Records))
	}
	if got := hex.EncodeToString(p.sum.Sum(nil))[:revisionHexLen]; got != h.Revision {
		return fmt.Errorf("%w: computed %s, header %s", ErrSnapshotRevision, got, h.Revision)
	}

	return p.checkReferences()
}

func (p *parser) checkReferences() error {
	tasks := make(map[string]struct{}, len(p.snap.Tasks))
	for _, t := range p.snap.Tasks {
		if t.ID == "" || t.Status == "" {
			return fmt.Errorf("%w: task needs id and status", ErrSnapshotMalformed)
		}
		if _, dup := tasks[t.ID]; dup {
			return fmt.Errorf("%w: duplicate task %q", ErrSnapshotMalformed, t.ID)
		}
		tasks[t.ID] = struct{}{}
	}

	records := make(map[string]struct{}, len(p.snap.Records))
	for _, rec := range p.snap.Records {
		if rec.ID == "" || rec.Kind == "" {
			return fmt.Errorf("%w: record needs id and kind", ErrSnapshotMalformed)
		}
		if _, dup := records[rec.ID]; dup {
			return fmt.Errorf("%w: duplicate record %q", ErrSnapshotMalformed, rec.ID)
		}
		if _, ok := tasks[rec.Task]; !ok {
			return fmt.Errorf("%w: record %q refers to unknown task %q", ErrSnapshotMalformed, rec.ID, rec.Task)
		}
		if (rec.Body == nil) == (rec.BodyMissing == nil) {
			return fmt.Errorf("%w: record %q needs exactly one of body and body_missing", ErrSnapshotMalformed, rec.ID)
		}
		records[rec.ID] = struct{}{}
	}

	for _, t := range p.snap.Tasks {
		for _, tr := range t.Transitions {
			if tr.Cause == nil {
				continue
			}
			if _, ok := records[*tr.Cause]; !ok {
				return fmt.Errorf("%w: task %q transition refers to unknown record %q",
					ErrSnapshotMalformed, t.ID, *tr.Cause)
			}
		}
	}
	return nil
}
