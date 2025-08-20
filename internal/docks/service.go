package docks

import (
	"context"
	"fmt"
	"gomodlag/internal/storage"
	"gomodlag/pkg"
	"sort"
)

func (s *ServiceDocks) AddNewLogic(ctx context.Context, data UploadRequest) error {
	if data.File != nil {
		filepath, err := pkg.SaveFile(data.File, "/app/uploads")
		if err != nil {
			return fmt.Errorf("failed save file")
		}
		data.Meta.FilePath = &filepath
		if !data.Meta.File {
			data.Meta.File = true
		}
	}
	data.Meta.Id = pkg.GenerateDockId()

	tx, err := s.Begin(ctx)
	if err != nil {
		return storage.Internal
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	d := storage.Dock{
		Id:       data.Meta.Id,
		IsFile:   data.Meta.File,
		Public:   data.Meta.Public,
		Name:     data.Meta.Name,
		Mime:     data.Meta.Mime,
		Json:     data.Json,
		Filepath: *data.Meta.FilePath,
		OwnerId:  data.Meta.OwnerId,
	}
	_, err = s.NewDocs(ctx, d, tx)
	if err != nil {
		return fmt.Errorf("failed create new doc")
	}

	_, err = s.AddGrant(ctx, data.Meta.Grant, data.Meta.Id, tx)
	if err != nil {
		return fmt.Errorf("failed to add grant")
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}
	committed = true
	return nil
}

func (s *ServiceDocks) FindDocksLogic(ctx context.Context, data storage.GetDock) ([]storage.DocumentWithGrants, error) {
	results, err := s.GetDock(ctx, data)
	if err != nil {
		return nil, err ////// 400 dat i vse
	} else {
		sort.Slice(results, func(i, j int) bool {
			if results[i].Name == results[j].Name {
				return results[i].CreatedAt.After(results[j].CreatedAt)
			}
			return results[i].Name < results[j].Name
		})
	}

	return results, nil
}

func (s *ServiceDocks) GetDockByIdLogic(ctx context.Context, data DockById) (storage.DocumentWithGrants, error) {
	dock, err := s.GetDockById(ctx, data.IdUser, data.IdDock)
	if err != nil {
		return storage.DocumentWithGrants{}, err
	}
	if dock.IsFile {
		file, err := pkg.GetFile(dock.Filepath)
		if err != nil {
			return storage.DocumentWithGrants{}, storage.Internal
		}
		dock.File = file
	}
	return dock, nil
}

func (s *ServiceDocks) DeleteDockLogic(ctx context.Context, data DockById) error {
	err := s.DeleteDock(ctx, data.IdUser, data.IdDock)
	if err != nil {
		return err
	}
	return nil
}
