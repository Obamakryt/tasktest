package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gomodlag/internal/auth"
	"gomodlag/internal/cache"
	"gomodlag/internal/docks"
	"gomodlag/internal/logger"
	"gomodlag/internal/storage"
	"gomodlag/pkg"
	"gomodlag/support"
	"net/http"
	"time"
)

type AuthRegDelHandler struct {
	auth.AuthRegDelLogic
	logger.Logger
}

func (a *AuthRegDelHandler) RegisterHandler(c echo.Context, adminToken string) error {
	var Data auth.Register
	err := c.Bind(&Data)
	if err != nil {
		return BadReq(c, Invalid)
	}
	if Data.AdminToken != adminToken {
		return BadReq(c, invalidToken)
	}
	if ok := pkg.Validator(Data.Login, Data.Password); !ok {
		return BadReq(c, Invalid)
	}
	login, err := a.LogicRegister(c.Request().Context(), Data)
	if err != nil {
		if errors.Is(err, storage.SomeWrong) {
			return somewrong(c)
		}
		return BadReq(c, Invalid)
	}
	type resp struct {
		Login string `json:"login"`
	}
	return Ok(c, resp{Login: login}, nil)
}

func (a *AuthRegDelHandler) AuthHandler(c echo.Context, Ttl time.Duration) error {
	var Data auth.Login
	err := c.Bind(&Data)
	if err != nil {
		return BadReq(c, Invalid)
	}

	token, err := a.LogicLogin(c.Request().Context(), Data, Ttl)

	if err != nil {
		if errors.Is(err, storage.Invaliddata) {
			return BadReq(c, Invalid) /// 400
		} else {
			return somewrong(c) //// 500
		}
	}
	type resp struct {
		Token string `json:"token"`
	}
	return Ok(c, resp{Token: token}, nil)
}

func (a *AuthRegDelHandler) LogOutHandler(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return BadReq(c, Invalid)
	}
	err := a.DeleteSession(c.Request().Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, storage.Invalidtoken):
			return BadReq(c, Invalid)
		default:
			return somewrong(c)
		}
	}
	type resp map[string]bool
	return Ok(c, resp{token: true}, nil)
}

type DockHandler struct {
	docks.DockLogic
	Cache *cache.MemoryCache
	logger.Logger
}

func (d *DockHandler) UploadDocHandler(c echo.Context, db storage.TokenValidator) error {
	data, err := support.ParseUploadRequest(c)
	if err != nil {
		return BadReq(c, Invalid)
	}

	id, err := db.ValidateToken(c.Request().Context(), data.Meta.Token)
	if err != nil {
		return BadReq(c, invalidToken)
	}

	data.Meta.OwnerId = id

	if err := d.AddNewLogic(c.Request().Context(), data); err != nil {
		return somewrong(c)
	}
	resp := map[string]any{
		"data": map[string]any{
			"json": data.Json,
			"file": data.Meta.Name,
		},
	}
	return Ok(c, nil, resp)
}

// /polychit
func (d *DockHandler) ListDocsHandler(c echo.Context) error {
	if c.Request().Method == http.MethodHead {
		return c.NoContent(http.StatusOK)
	}
	var FilterData storage.GetDock

	err := c.Bind(&FilterData)
	if err != nil {
		return BadReq(c, Invalid)
	}
	userID, o := c.Get("userid").(int)
	if !o {
		return BadReq(c, "invalid user context")
	}
	FilterData.Id = userID
	if FilterData.Limit == 0 {
		FilterData.Limit = 50
	}
	docs, err := d.FindDocksLogic(c.Request().Context(), FilterData)
	if err != nil {
		return somewrong(c)
	}
	respDocs := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		respDocs = append(respDocs, map[string]any{
			"id":      doc.ID.String(),
			"name":    doc.Name,
			"mime":    doc.Mime,
			"file":    doc.IsFile,
			"public":  doc.Public,
			"created": doc.CreatedAt.Format("2006-01-02 15:04:05"),
			"grant":   doc.GrantedUsers,
		})
	}

	return Ok(c, nil, map[string]any{
		"docs": respDocs,
	})
}

func (d *DockHandler) GetDocHandler(c echo.Context) error {
	if c.Request().Method == http.MethodHead {
		return c.NoContent(http.StatusOK)
	}
	id := c.Param("id")
	dockId, err := uuid.Parse(id)
	if err != nil {
		return BadReq(c, Invalid)
	}
	userID, o := c.Get("userid").(int)
	if !o {
		return BadReq(c, "invalid user context")
	}
	cacheKey := fmt.Sprintf(cache.Key, dockId.String(), userID)
	///file
	if docData, mimeType, found := d.Cache.GetFile(cacheKey); found {
		return c.Blob(http.StatusOK, mimeType, docData)
	}
	///json
	var cachedJson json.RawMessage
	if found, err := d.Cache.GetJSON(cacheKey, &cachedJson); found && err == nil {
		resp := map[string]any{"data": cachedJson}
		return Ok(c, nil, resp)
	}

	data := docks.DockById{
		IdUser: userID, IdDock: dockId,
	}

	doc, err := d.GetDockByIdLogic(c.Request().Context(), data)
	if err != nil {
		if errors.Is(err, storage.Invaliddata) {
			return BadReq(c, "document not found")
		}
		return somewrong(c)
	}
	if doc.IsFile {
		if doc.File == nil {
			return somewrong(c)
		}
		d.Cache.SetFile(cacheKey, doc.File, doc.Mime)
		return c.Blob(http.StatusOK, doc.Mime, doc.File)
	}
	err = d.Cache.SetJSON(cacheKey, doc.Json)
	if err != nil {
		d.Logger.Info("failed cache set", err)
	}
	resp := map[string]any{
		"data": doc.Json,
	}
	return Ok(c, nil, resp)
}

func (d *DockHandler) DeleteDocHandler(c echo.Context) error {
	id := c.Param("id")
	dockId, err := uuid.Parse(id)
	if err != nil {
		return BadReq(c, Invalid)
	}
	userID := c.Get("userid").(int)
	data := docks.DockById{IdUser: userID, IdDock: dockId}

	err = d.DeleteDockLogic(c.Request().Context(), data)
	if err != nil {
		if errors.Is(err, storage.Forbidden) {
			return norute(c, "you cannot delete this document")
		}
		return somewrong(c)
	}
	cacheKey := fmt.Sprintf(cache.Key, dockId.String(), userID)
	d.Cache.Delete(cacheKey)

	return Ok(c, map[string]bool{id: true}, nil)
}
