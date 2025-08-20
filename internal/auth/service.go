package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"gomodlag/internal/storage"
	"gomodlag/pkg"
	"log/slog"
	"time"
)

func (s *ServiceDB) LogicRegister(ctx context.Context, data Register) (string, error) {
	hashpass := pkg.CreateHash(data.Password)
	login, err := s.Register(ctx, hashpass, data.Login)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return "", fmt.Errorf("username exists")
			default:
				s.Logger.Error("Postgres error",
					slog.String("code", pgErr.Code),
					slog.String("message", pgErr.Message))
			}
		}
		s.Logger.Info("LogicRegister err: ", err, slog.String("username", data.Login))
		return "", storage.SomeWrong
	}
	return login, nil
}

func (s *ServiceDB) LogicLogin(ctx context.Context, data Login, ttl time.Duration) (string, error) {
	hashpass := pkg.CreateHash(data.Password)
	newtoken := pkg.GenerateToken()
	err := s.Login(ctx, hashpass, data.Login, newtoken, ttl)
	if err != nil {
		return "", err
	}
	return newtoken, nil
}

func (s *ServiceDB) DeleteSession(ctx context.Context, token string) error {
	id, err := s.TokenValidator.ValidateToken(ctx, token)
	if err != nil {
		return err
	}
	return s.DeleteToken(ctx, id)
}
