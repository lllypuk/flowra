package fixtures

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

const leadFeedRevisionHexLen = 16

// LeadFeedBase is the time the synthetic lead feed snapshots are built around.
var LeadFeedBase = time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)

// LeadFeedTask builds a synthetic task; waitingOn "" means the task waits on nobody.
func LeadFeedTask(id, status, waitingOn string, createdMinutes int) leadfeed.Task {
	created := LeadFeedBase.Add(time.Duration(createdMinutes) * time.Minute)
	task := leadfeed.Task{
		ID:          id,
		Title:       "Synthetic task " + id,
		Project:     "alpha",
		Agent:       "claude",
		CreatedAt:   created,
		Status:      status,
		StatusSince: created,
		Spent:       leadfeed.Spent{Work: 60}, //nolint:mnd // arbitrary seconds
		Transitions: []leadfeed.Transition{{At: created, Status: status}},
	}
	if waitingOn != "" {
		task.WaitingOn = &waitingOn
	}
	return task
}

// LeadFeedLetter builds a synthetic letter record with a body.
func LeadFeedLetter(id, taskID, body string, order int) leadfeed.Record {
	return leadfeed.Record{
		ID:          id,
		Task:        taskID,
		Kind:        "letter",
		At:          LeadFeedBase.Add(time.Duration(order) * time.Second),
		Order:       order,
		Actor:       "lead",
		Source:      leadfeed.Source{Kind: "journal", Ref: id},
		Body:        &body,
		BodyDerived: true,
		Format:      "text",
	}
}

// LeadFeedSnapshotJSONL encodes a snapshot the way the lead tool exports it, with a matching revision and footer.
func LeadFeedSnapshotJSONL(asOf time.Time, tasks []leadfeed.Task, records []leadfeed.Record) ([]byte, error) {
	lines := make([][]byte, 0, len(tasks)+len(records))
	for _, task := range tasks {
		line, err := json.Marshal(struct {
			leadfeed.Task

			Type string `json:"type"`
		}{Type: "task", Task: task})
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	for _, record := range records {
		line, err := json.Marshal(struct {
			leadfeed.Record

			Type string `json:"type"`
		}{Type: "record", Record: record})
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	sum := sha256.Sum256(bytes.Join(lines, []byte("\n")))
	revision := hex.EncodeToString(sum[:])[:leadFeedRevisionHexLen]

	header, err := json.Marshal(map[string]any{
		"type": "header", "schema": leadfeed.SchemaVersion, "source": "lead-task", "epoch": "feedfacecafe",
		"revision": revision, "as_of": asOf.UTC().Format(time.RFC3339Nano), "derivation": 1,
		"complete": true, "errors": []string{},
	})
	if err != nil {
		return nil, err
	}
	footer, err := json.Marshal(map[string]any{
		"type": "footer", "revision": revision, "tasks": len(tasks), "records": len(records),
	})
	if err != nil {
		return nil, err
	}

	out := append([][]byte{header}, lines...)
	out = append(out, footer)
	return append(bytes.Join(out, []byte("\n")), '\n'), nil
}
