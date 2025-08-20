package support

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"gomodlag/internal/docks"
	"net/http"
)

func ParseUploadRequest(c echo.Context) (docks.UploadRequest, error) {
	var req docks.UploadRequest

	//////load file
	_, f, err := c.Request().FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			// net faila
			f = nil
		} else {
			return req, fmt.Errorf("failed to read file: %w", err)
		}
	}
	req.File = f

	metaStr := c.Request().FormValue("meta")
	if metaStr == "" {
		return req, fmt.Errorf("meta is required")
	}
	if err := json.Unmarshal([]byte(metaStr), &req.Meta); err != nil {
		return req, fmt.Errorf("invalid meta")
	}

	if jsonStr := c.Request().FormValue("json"); jsonStr != "" {
		if err := json.Unmarshal([]byte(jsonStr), &req.Json); err != nil {
			return req, fmt.Errorf("invalid json")
		}
	} else {
		req.Json = nil
	}
	return req, nil
}
