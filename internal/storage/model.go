package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"time"
)

var Invalidtoken = errors.New("invalid token")
var Invaliddata = errors.New("invalid data")
var SomeWrong = errors.New("something went wrong")
var Internal = errors.New("internal server error")
var Forbidden = errors.New("you cannot delete this document")

type Token struct {
	Token       string
	TimeCreated time.Time
	TimeExpired time.Time
}
type Dock struct {
	Id       uuid.UUID
	IsFile   bool
	Public   bool
	Name     string
	Mime     string
	Json     json.RawMessage
	Filepath string
	OwnerId  int
}

type GetDock struct {
	Id    int    `json:"-"`
	Token string `json:"token"`
	Login string `json:"login,omitempty"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Limit int    `json:"limit"`
}
type DocumentWithGrants struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Mime         string          `json:"mime"`
	IsFile       bool            `json:"is_file"`
	Public       bool            `json:"public"`
	CreatedAt    time.Time       `json:"created_at"`
	GrantedUsers []string        `json:"granted_users"`
	Json         json.RawMessage `json:"json_data"`
	File         []byte          `json:"file"`
	Filepath     string          `json:"-"`
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (int, error)
}

type AuthRegDelModel interface {
	Register(ctx context.Context, password, username string) (string, error)
	Login(ctx context.Context, password, username string, token string, ttl time.Duration) error
	DeleteToken(ctx context.Context, idUser int) error
}

type DockModel interface {
	GetDockById(ctx context.Context, idUser int, idDock uuid.UUID) (DocumentWithGrants, error)
	DeleteDock(ctx context.Context, idUser int, idDock uuid.UUID) error
	GetDock(ctx context.Context, filter GetDock) ([]DocumentWithGrants, error)
	AddGrant(ctx context.Context, grants []string, docid uuid.UUID, tx pgx.Tx) (bool, error)
	NewDocs(ctx context.Context, dock Dock, tx pgx.Tx) (bool, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}
