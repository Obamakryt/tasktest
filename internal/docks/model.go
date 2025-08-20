package docks

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"gomodlag/internal/logger"
	"gomodlag/internal/storage"
	"mime/multipart"
)

type DocMeta struct {
	Id       uuid.UUID `json:"-"`
	Name     string    `json:"name" form:"name"`
	File     bool      `json:"file" form:"file"`
	Public   bool      `json:"public" form:"public"`
	Token    string    `json:"token" form:"token"`
	Mime     string    `json:"mime" form:"mime"`
	Grant    []string  `json:"grant" form:"grant"`
	OwnerId  int       `json:"-"`
	FilePath *string   `json:"-"`
}

type UploadRequest struct {
	Meta DocMeta
	File *multipart.FileHeader `form:"file"`
	Json json.RawMessage
}

type ServiceDocks struct {
	storage.DockModel
	logger.Logger
}

type DockById struct {
	IdUser int
	IdDock uuid.UUID
}

type DockLogic interface {
	AddNewLogic(ctx context.Context, data UploadRequest) error
	FindDocksLogic(ctx context.Context, data storage.GetDock) ([]storage.DocumentWithGrants, error)
	GetDockByIdLogic(ctx context.Context, data DockById) (storage.DocumentWithGrants, error)
	DeleteDockLogic(ctx context.Context, data DockById) error
}
