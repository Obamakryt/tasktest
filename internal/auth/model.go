package auth

import (
	"context"
	"gomodlag/internal/logger"
	"gomodlag/internal/storage"
	"time"
)

type Register struct {
	Login      string `json:"login"`
	Password   string `json:"password"`
	AdminToken string `json:"token"`
}
type Login struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ServiceDB struct {
	storage.AuthRegDelModel
	logger.Logger
	storage.TokenValidator
}

type AuthRegDelLogic interface {
	DeleteSession(ctx context.Context, token string) error
	LogicLogin(ctx context.Context, data Login, ttl time.Duration) (string, error)
	LogicRegister(ctx context.Context, data Register) (string, error)
}
