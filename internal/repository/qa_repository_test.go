package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/witfoo/due-diligence-portal/internal/domain"
)

func createTestUser(t *testing.T, db *DB, id, email string) {
	t.Helper()
	repo := NewUserRepository(db)
	user := &domain.User{
		ID: id, Email: email, Name: "Test User",
		PasswordHash: "h", Role: domain.RoleAdmin, IsActive: true,
	}
	require.NoError(t, repo.Create(context.Background(), user))
}

func createTestCategory(t *testing.T, db *DB, id, slug string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO categories (id, name, slug) VALUES (?, ?, ?)`,
		id, "Test Category", slug)
	require.NoError(t, err)
}

func createTestDocument(t *testing.T, db *DB, id, categoryID, userID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO documents (id, name, category_id, uploaded_by, mime_type, file_size) VALUES (?, ?, ?, ?, ?, ?)`,
		id, "Test Doc", categoryID, userID, "application/pdf", 1024)
	require.NoError(t, err)
}

func TestQARepository_CreateThread(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "qa@test.com")

	thread := &domain.QAThread{
		ID:      "t1",
		Subject: "Question about financials",
		Status:  domain.QAStatusOpen,
		AskedBy: "u1",
	}

	err := repo.CreateThread(ctx, thread)
	require.NoError(t, err)
	assert.False(t, thread.CreatedAt.IsZero())
}

func TestQARepository_GetThread(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "qa@test.com")

	thread := &domain.QAThread{
		ID:      "t1",
		Subject: "Question about financials",
		Status:  domain.QAStatusOpen,
		AskedBy: "u1",
	}
	require.NoError(t, repo.CreateThread(ctx, thread))

	found, err := repo.GetThread(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "Question about financials", found.Subject)
	assert.Equal(t, domain.QAStatusOpen, found.Status)
	assert.Equal(t, "u1", found.AskedBy)
}

func TestQARepository_GetThread_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	_, err := repo.GetThread(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrThreadNotFound)
}

func TestQARepository_ListThreads(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "qa@test.com")

	for i := 0; i < 5; i++ {
		status := domain.QAStatusOpen
		if i >= 3 {
			status = domain.QAStatusAnswered
		}
		thread := &domain.QAThread{
			ID: fmt.Sprintf("t%d", i), Subject: fmt.Sprintf("Q %d", i),
			Status: status, AskedBy: "u1",
		}
		require.NoError(t, repo.CreateThread(ctx, thread))
	}

	// List all.
	threads, total, err := repo.ListThreads(ctx, "", "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, threads, 5)

	// Filter by status.
	threads, total, err = repo.ListThreads(ctx, domain.QAStatusOpen, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, threads, 3)

	// Pagination.
	threads, total, err = repo.ListThreads(ctx, "", "", 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, threads, 2)

	// Scope by asker (all threads were asked by "u1").
	threads, total, err = repo.ListThreads(ctx, "", "u1", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, threads, 5)

	threads, total, err = repo.ListThreads(ctx, "", "someone-else", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, threads, 0)
}

func TestQARepository_UpdateThreadStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "qa@test.com")

	thread := &domain.QAThread{
		ID: "t1", Subject: "Q", Status: domain.QAStatusOpen, AskedBy: "u1",
	}
	require.NoError(t, repo.CreateThread(ctx, thread))

	err := repo.UpdateThreadStatus(ctx, "t1", domain.QAStatusAnswered)
	require.NoError(t, err)

	found, err := repo.GetThread(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, domain.QAStatusAnswered, found.Status)
}

func TestQARepository_UpdateThreadStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	err := repo.UpdateThreadStatus(ctx, "nonexistent", domain.QAStatusClosed)
	assert.ErrorIs(t, err, domain.ErrThreadNotFound)
}

func TestQARepository_CreateMessage(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "qa@test.com")

	thread := &domain.QAThread{
		ID: "t1", Subject: "Q", Status: domain.QAStatusOpen, AskedBy: "u1",
	}
	require.NoError(t, repo.CreateThread(ctx, thread))

	msg := &domain.QAMessage{
		ID:       "m1",
		ThreadID: "t1",
		AuthorID: "u1",
		Body:     "This is a question.",
	}
	err := repo.CreateMessage(ctx, msg)
	require.NoError(t, err)
	assert.False(t, msg.CreatedAt.IsZero())
}

func TestQARepository_ListMessages(t *testing.T) {
	db := setupTestDB(t)
	repo := NewQARepository(db)
	ctx := context.Background()

	createTestUser(t, db, "u1", "qa@test.com")

	thread := &domain.QAThread{
		ID: "t1", Subject: "Q", Status: domain.QAStatusOpen, AskedBy: "u1",
	}
	require.NoError(t, repo.CreateThread(ctx, thread))

	for i := 0; i < 3; i++ {
		msg := &domain.QAMessage{
			ID: fmt.Sprintf("m%d", i), ThreadID: "t1", AuthorID: "u1",
			Body: fmt.Sprintf("Message %d", i),
		}
		require.NoError(t, repo.CreateMessage(ctx, msg))
	}
	// One internal (company-only) message.
	require.NoError(t, repo.CreateMessage(ctx, &domain.QAMessage{
		ID: "m-internal", ThreadID: "t1", AuthorID: "u1", Body: "internal note", IsInternal: true,
	}))

	// includeInternal=true returns all four.
	messages, err := repo.ListMessages(ctx, "t1", true)
	require.NoError(t, err)
	assert.Len(t, messages, 4)
	assert.Equal(t, "Message 0", messages[0].Body)

	// includeInternal=false hides the internal message.
	messages, err = repo.ListMessages(ctx, "t1", false)
	require.NoError(t, err)
	assert.Len(t, messages, 3)
	for _, m := range messages {
		assert.False(t, m.IsInternal)
	}
}
