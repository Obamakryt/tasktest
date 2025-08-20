package tests

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"gomodlag/internal/storage"
	"testing"
)

// setupTestDB подключается к тестовой БД
func setupTestDB(t *testing.T) *storage.StructPool {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://boss:bosspass@localhost:5443/test")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Проверяем соединение
	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return &storage.StructPool{Pool: pool}
}

// cleanupTestDB очищает тестовые данные
func cleanupTestDB(t *testing.T, s *storage.StructPool) {
	ctx := context.Background()

	_, err := s.Pool.Exec(ctx, `
		TRUNCATE TABLE sessions, document_grants, documents, users RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("Failed to clean tables: %v", err)
	}
}
