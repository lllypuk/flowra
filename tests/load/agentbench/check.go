package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type contendCheck struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	// Final answer still unknown (503 or no response): either outcome is allowed.
	Unknown int `json:"unknown"`
	Events  int `json:"events"`
	// Accepted requests whose value has no event: the lost updates.
	Lost int `json:"lost"`
	// Rejected requests whose value has an event anyway.
	RejectedApplied int `json:"rejected_applied"`
	// Values with more than one event: a request applied twice.
	Duplicated int `json:"duplicated"`
	// Events whose value no request asked for.
	Foreign      int    `json:"foreign"`
	LastEvent    string `json:"last_event"`
	ChatsModel   string `json:"chats_read_model"`
	TasksModel   string `json:"tasks_read_model"`
	ModelsAgree  bool   `json:"models_agree"`
	ContractHeld bool   `json:"contract_held"`
}

// checkContend compares what the clients were told with the chat.due_date_set events and both read models.
func checkContend(ctx context.Context, cfg config, chatID string, outcomes map[string]int) (*contendCheck, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.mongoURI))
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Disconnect(ctx) }()
	db := client.Database(cfg.mongoDB)

	cur, err := db.Collection("events").Find(ctx,
		bson.M{"aggregate_id": chatID, "event_type": "chat.due_date_set"},
		options.Find().SetSort(bson.D{{Key: "version", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var events []struct {
		Data struct {
			NewDueDate string `bson:"new_due_date"`
		} `bson:"data"`
	}
	if err = cur.All(ctx, &events); err != nil {
		return nil, err
	}

	res := &contendCheck{Events: len(events)}
	seen := map[string]int{}
	for _, e := range events {
		day := e.Data.NewDueDate[:min(len(time.DateOnly), len(e.Data.NewDueDate))]
		seen[day]++
		res.LastEvent = day
	}
	for day, n := range seen {
		if n > 1 {
			res.Duplicated++
		}
		if _, asked := outcomes[day]; !asked {
			res.Foreign++
		}
	}
	for day, status := range outcomes {
		applied := seen[day] > 0
		switch {
		case status == http.StatusOK:
			res.Accepted++
			if !applied {
				res.Lost++
			}
		case status == 0 || status == http.StatusServiceUnavailable:
			res.Unknown++
		default:
			res.Rejected++
			if applied {
				res.RejectedApplied++
			}
		}
	}

	res.ChatsModel, err = dueDateOf(ctx, db.Collection("chats_read_model"), bson.M{"chat_id": chatID})
	if err != nil {
		return nil, err
	}
	res.TasksModel, err = dueDateOf(ctx, db.Collection("tasks_read_model"), bson.M{"task_id": chatID})
	if err != nil {
		return nil, err
	}
	res.ModelsAgree = res.ChatsModel == res.LastEvent && res.TasksModel == res.LastEvent
	res.ContractHeld = res.Lost == 0 && res.RejectedApplied == 0 && res.Duplicated == 0 && res.Foreign == 0 &&
		res.ModelsAgree
	return res, nil
}

func dueDateOf(ctx context.Context, coll *mongo.Collection, filter bson.M) (string, error) {
	var doc struct {
		DueDate *time.Time `bson:"due_date"`
	}
	if err := coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		return "", fmt.Errorf("%s: %w", coll.Name(), err)
	}
	if doc.DueDate == nil {
		return "", nil
	}
	return doc.DueDate.UTC().Format(time.DateOnly), nil
}
