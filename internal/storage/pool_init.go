package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gomodlag/internal/logger"
	"log/slog"
	"time"
)

type StructPool struct {
	Pool *pgxpool.Pool
}

func NewPool(DBURL string, logger logger.Logger) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(DBURL)
	if err != nil {
		logger.Error("Failed to parse DB config")
		return nil, fmt.Errorf("invalid DB config: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		logger.Warn("Connect failed", slog.String("error", err.Error()))
		return nil, err
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err = pool.Ping(ctx)
		if err != nil {
			pool.Close()
			var pgErr *pgconn.PgError
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				return nil, fmt.Errorf("timeout exceeded db connection")
			case errors.As(err, &pgErr):
				return nil, fmt.Errorf("pgx error: %s, %s", pgErr.Code, pgErr.Message)
			default:
				return nil, fmt.Errorf("connection db error: %w", err)
			}
		}
		logger.Info("Successfully connected to pgx")
		return pool, nil
	}
}
