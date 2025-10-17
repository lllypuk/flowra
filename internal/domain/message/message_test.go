package message_test

import (
	"testing"

	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/message"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

//nolint:gocognit,errorlint // Test complexity is acceptable
func TestNewMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		content := "Hello, world!"

		msg, err := message.NewMessage(chatID, authorID, content, uuid.UUID(""))

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if msg.ChatID() != chatID {
			t.Errorf("expected chatID %v, got %v", chatID, msg.ChatID())
		}
		if msg.AuthorID() != authorID {
			t.Errorf("expected authorID %v, got %v", authorID, msg.AuthorID())
		}
		if msg.Content() != content {
			t.Errorf("expected content %q, got %q", content, msg.Content())
		}
		if msg.IsDeleted() {
			t.Error("expected message not to be deleted")
		}
	})

	t.Run("empty chatID", func(t *testing.T) {
		authorID := uuid.NewUUID()
		content := "Test"

		_, err := message.NewMessage(uuid.UUID(""), authorID, content, uuid.UUID(""))

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("empty authorID", func(t *testing.T) {
		chatID := uuid.NewUUID()
		content := "Test"

		_, err := message.NewMessage(chatID, uuid.UUID(""), content, uuid.UUID(""))

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()

		_, err := message.NewMessage(chatID, authorID, "", uuid.UUID(""))

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("with parent message (thread)", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		parentID := uuid.NewUUID()

		msg, err := message.NewMessage(chatID, authorID, "Reply", parentID)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !msg.IsReply() {
			t.Error("expected message to be a reply")
		}
		if msg.ParentMessageID() != parentID {
			t.Errorf("expected parentMessageID %v, got %v", parentID, msg.ParentMessageID())
		}
	})
}

//nolint:errorlint // Direct error comparison is acceptable in tests
func TestMessage_EditContent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Original", uuid.UUID(""))

		err := msg.EditContent("Updated", authorID)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if msg.Content() != "Updated" {
			t.Errorf("expected content 'Updated', got %q", msg.Content())
		}
		if !msg.IsEdited() {
			t.Error("expected message to be marked as edited")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Original", uuid.UUID(""))

		err := msg.EditContent("", authorID)

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Original", uuid.UUID(""))

		err := msg.EditContent("Hacked", otherUserID)

		if err != errs.ErrForbidden {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
		if msg.Content() != "Original" {
			t.Error("content should not be changed")
		}
	})

	t.Run("cannot edit deleted message", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Original", uuid.UUID(""))
		_ = msg.Delete(authorID)

		err := msg.EditContent("Updated", authorID)

		if err != errs.ErrInvalidState {
			t.Errorf("expected ErrInvalidState, got %v", err)
		}
	})
}

//nolint:errorlint // Direct error comparison is acceptable in tests
func TestMessage_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.Delete(authorID)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !msg.IsDeleted() {
			t.Error("expected message to be deleted")
		}
		if msg.DeletedAt() == nil {
			t.Error("expected DeletedAt to be set")
		}
	})

	t.Run("forbidden - not author", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.Delete(otherUserID)

		if err != errs.ErrForbidden {
			t.Errorf("expected ErrForbidden, got %v", err)
		}
		if msg.IsDeleted() {
			t.Error("message should not be deleted")
		}
	})

	t.Run("cannot delete already deleted message", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		_ = msg.Delete(authorID)

		err := msg.Delete(authorID)

		if err != errs.ErrInvalidState {
			t.Errorf("expected ErrInvalidState, got %v", err)
		}
	})
}

//nolint:gocognit,errorlint // Test complexity is acceptable
func TestMessage_AddReaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddReaction(userID, "👍")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !msg.HasReaction(userID, "👍") {
			t.Error("expected message to have reaction")
		}
		if msg.GetReactionCount("👍") != 1 {
			t.Errorf("expected reaction count 1, got %d", msg.GetReactionCount("👍"))
		}
	})

	t.Run("duplicate reaction", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		_ = msg.AddReaction(userID, "👍")

		err := msg.AddReaction(userID, "👍")

		if err != errs.ErrAlreadyExists {
			t.Errorf("expected ErrAlreadyExists, got %v", err)
		}
	})

	t.Run("multiple users same emoji", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		user1 := uuid.NewUUID()
		user2 := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		_ = msg.AddReaction(user1, "👍")
		_ = msg.AddReaction(user2, "👍")

		if msg.GetReactionCount("👍") != 2 {
			t.Errorf("expected reaction count 2, got %d", msg.GetReactionCount("👍"))
		}
	})

	t.Run("same user different emojis", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		_ = msg.AddReaction(userID, "👍")
		err := msg.AddReaction(userID, "❤️")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !msg.HasReaction(userID, "👍") || !msg.HasReaction(userID, "❤️") {
			t.Error("expected message to have both reactions")
		}
	})

	t.Run("cannot add reaction to deleted message", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		_ = msg.Delete(authorID)

		err := msg.AddReaction(userID, "👍")

		if err != errs.ErrInvalidState {
			t.Errorf("expected ErrInvalidState, got %v", err)
		}
	})

	t.Run("invalid reaction - empty userID", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddReaction(uuid.UUID(""), "👍")

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid reaction - empty emoji", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddReaction(userID, "")

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

//nolint:errorlint // Direct error comparison is acceptable in tests
func TestMessage_RemoveReaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		_ = msg.AddReaction(userID, "👍")

		err := msg.RemoveReaction(userID, "👍")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if msg.HasReaction(userID, "👍") {
			t.Error("expected reaction to be removed")
		}
	})

	t.Run("reaction not found", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.RemoveReaction(userID, "👍")

		if err != errs.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("remove only specific user reaction", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		user1 := uuid.NewUUID()
		user2 := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		_ = msg.AddReaction(user1, "👍")
		_ = msg.AddReaction(user2, "👍")

		_ = msg.RemoveReaction(user1, "👍")

		if msg.HasReaction(user1, "👍") {
			t.Error("expected user1 reaction to be removed")
		}
		if !msg.HasReaction(user2, "👍") {
			t.Error("expected user2 reaction to remain")
		}
		if msg.GetReactionCount("👍") != 1 {
			t.Errorf("expected reaction count 1, got %d", msg.GetReactionCount("👍"))
		}
	})
}

//nolint:errorlint // Direct error comparison is acceptable in tests
func TestMessage_AddAttachment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		fileID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddAttachment(fileID, "test.pdf", 1024, "application/pdf")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		attachments := msg.Attachments()
		if len(attachments) != 1 {
			t.Errorf("expected 1 attachment, got %d", len(attachments))
		}
		if attachments[0].FileID() != fileID {
			t.Error("attachment fileID mismatch")
		}
	})

	t.Run("multiple attachments", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		_ = msg.AddAttachment(uuid.NewUUID(), "file1.pdf", 1024, "application/pdf")
		_ = msg.AddAttachment(uuid.NewUUID(), "file2.jpg", 2048, "image/jpeg")

		attachments := msg.Attachments()
		if len(attachments) != 2 {
			t.Errorf("expected 2 attachments, got %d", len(attachments))
		}
	})

	t.Run("cannot add attachment to deleted message", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		_ = msg.Delete(authorID)

		err := msg.AddAttachment(uuid.NewUUID(), "test.pdf", 1024, "application/pdf")

		if err != errs.ErrInvalidState {
			t.Errorf("expected ErrInvalidState, got %v", err)
		}
	})

	t.Run("invalid attachment - empty fileID", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddAttachment(uuid.UUID(""), "test.pdf", 1024, "application/pdf")

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid attachment - empty filename", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddAttachment(uuid.NewUUID(), "", 1024, "application/pdf")

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid attachment - zero size", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddAttachment(uuid.NewUUID(), "test.pdf", 0, "application/pdf")

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("invalid attachment - empty mimetype", func(t *testing.T) {
		chatID := uuid.NewUUID()
		authorID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

		err := msg.AddAttachment(uuid.NewUUID(), "test.pdf", 1024, "")

		if err != errs.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestMessage_Getters(t *testing.T) {
	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	parentID := uuid.NewUUID()
	content := "Test message"

	msg, _ := message.NewMessage(chatID, authorID, content, parentID)

	if msg.ID().IsZero() {
		t.Error("expected ID to be generated")
	}
	if msg.ChatID() != chatID {
		t.Error("ChatID getter failed")
	}
	if msg.AuthorID() != authorID {
		t.Error("AuthorID getter failed")
	}
	if msg.Content() != content {
		t.Error("Content getter failed")
	}
	if msg.ParentMessageID() != parentID {
		t.Error("ParentMessageID getter failed")
	}
	if msg.CreatedAt().IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if msg.EditedAt() != nil {
		t.Error("expected EditedAt to be nil initially")
	}
	if msg.DeletedAt() != nil {
		t.Error("expected DeletedAt to be nil initially")
	}
}

func TestMessage_CanBeEditedBy(t *testing.T) {
	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	otherUserID := uuid.NewUUID()
	msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

	if !msg.CanBeEditedBy(authorID) {
		t.Error("expected author to be able to edit")
	}
	if msg.CanBeEditedBy(otherUserID) {
		t.Error("expected other user not to be able to edit")
	}
}

func TestMessage_IsReply(t *testing.T) {
	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()

	t.Run("regular message", func(t *testing.T) {
		msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))
		if msg.IsReply() {
			t.Error("expected message not to be a reply")
		}
	})

	t.Run("reply message", func(t *testing.T) {
		parentID := uuid.NewUUID()
		msg, _ := message.NewMessage(chatID, authorID, "Reply", parentID)
		if !msg.IsReply() {
			t.Error("expected message to be a reply")
		}
	})
}

func TestMessage_GetReactionCount(t *testing.T) {
	chatID := uuid.NewUUID()
	authorID := uuid.NewUUID()
	msg, _ := message.NewMessage(chatID, authorID, "Test", uuid.UUID(""))

	user1 := uuid.NewUUID()
	user2 := uuid.NewUUID()
	user3 := uuid.NewUUID()

	_ = msg.AddReaction(user1, "👍")
	_ = msg.AddReaction(user2, "👍")
	_ = msg.AddReaction(user3, "❤️")

	if count := msg.GetReactionCount("👍"); count != 2 {
		t.Errorf("expected 👍 count 2, got %d", count)
	}
	if count := msg.GetReactionCount("❤️"); count != 1 {
		t.Errorf("expected ❤️ count 1, got %d", count)
	}
	if count := msg.GetReactionCount("🔥"); count != 0 {
		t.Errorf("expected 🔥 count 0, got %d", count)
	}
}
