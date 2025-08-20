package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gomodlag/internal/api"
	"gomodlag/internal/cache"
	"gomodlag/internal/docks"
	"gomodlag/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockAuthService для тестов
type MockDockService struct {
	mock.Mock
}

func (m *MockDockService) AddNewLogic(ctx context.Context, data docks.UploadRequest) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockDockService) FindDocksLogic(ctx context.Context, data storage.GetDock) ([]storage.DocumentWithGrants, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]storage.DocumentWithGrants), args.Error(1)
}

func (m *MockDockService) GetDockByIdLogic(ctx context.Context, data docks.DockById) (storage.DocumentWithGrants, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(storage.DocumentWithGrants), args.Error(1)
}

func (m *MockDockService) DeleteDockLogic(ctx context.Context, data docks.DockById) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// Добавляем методы интерфейса если нужно
func (m *MockDockService) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockDockService) NewDocs(ctx context.Context, dock storage.Dock, tx pgx.Tx) (bool, error) {
	args := m.Called(ctx, dock, tx)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockService) AddGrant(ctx context.Context, grants []string, docid uuid.UUID, tx pgx.Tx) (bool, error) {
	args := m.Called(ctx, grants, docid, tx)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockService) GetDock(ctx context.Context, filter storage.GetDock) ([]storage.DocumentWithGrants, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]storage.DocumentWithGrants), args.Error(1)
}

func (m *MockDockService) GetDockById(ctx context.Context, idUser int, idDock uuid.UUID) (storage.DocumentWithGrants, error) {
	args := m.Called(ctx, idUser, idDock)
	return args.Get(0).(storage.DocumentWithGrants), args.Error(1)
}

func (m *MockDockService) DeleteDock(ctx context.Context, idUser int, idDock uuid.UUID) error {
	args := m.Called(ctx, idUser, idDock)
	return args.Error(0)
}

func TestDockHandler_Smoke(t *testing.T) {
	e := echo.New()

	mockDock := new(MockDockService)
	mockCache := cache.NewMemoryCache(time.Minute)
	handler := &api.DockHandler{
		DockLogic: mockDock,
		Cache:     mockCache,
	}

	// Test ListDocs - должен работать с контекстом пользователя
	t.Run("ListDocs works with user context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Устанавливаем userid в контекст (как это делает middleware)
		c.Set("userid", 1)

		expectedFilter := storage.GetDock{
			Id:    1,
			Limit: 50, // default limit
		}
		mockDock.On("FindDocksLogic", mock.Anything, expectedFilter).
			Return([]storage.DocumentWithGrants{}, nil)

		err := handler.ListDocsHandler(c)

		// Теперь не должно быть ошибки
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockDock.AssertExpectations(t)
	})

	// Test GetDoc returns 400 on invalid UUID
	t.Run("GetDoc validates UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")
		c.Set("userid", 1)

		err := handler.GetDocHandler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	// Test DeleteDoc returns 400 on invalid UUID
	t.Run("DeleteDoc validates UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/docs/invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid")
		c.Set("userid", 1)

		err := handler.DeleteDocHandler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	// Test без user context
	t.Run("ListDocs fails without user context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// НЕ устанавливаем userid

		err := handler.ListDocsHandler(c)

		// Должен вернуть 400 потому что нет user context
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	// Test с параметрами фильтра
	t.Run("ListDocs with filter parameters", func(t *testing.T) {
		body := []byte(`{"key": "name", "value": "test", "limit": 10}`)
		req := httptest.NewRequest(http.MethodGet, "/api/docs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userid", 1)

		// Настраиваем mock
		expectedFilter := storage.GetDock{
			Id:    1,
			Key:   "name",
			Value: "test",
			Limit: 10,
		}
		mockDock.On("FindDocksLogic", mock.Anything, expectedFilter).
			Return([]storage.DocumentWithGrants{}, nil)

		err := handler.ListDocsHandler(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockDock.AssertExpectations(t)
	})
}

// Тест на структуру ответа
func TestAPIResponseStructure(t *testing.T) {
	e := echo.New()

	// Test успешного ответа
	t.Run("Success response has correct structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		testData := map[string]string{"test": "data"}
		err := api.Ok(c, testData, nil)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response api.ApiResp
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Nil(t, response.Error)
		assert.NotNil(t, response.Response)
	})

	// Test ошибки
	t.Run("Error response has correct structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := api.BadReq(c, "test error")

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response api.ApiResp
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.Error)
		assert.Equal(t, 400, response.Error.Code)
		assert.Equal(t, "test error", response.Error.Text)
	})
}

// Быстрый тест на middleware
func TestAuthMiddleware(t *testing.T) {
	e := echo.New()

	mockValidator := new(MockTokenValidator)
	middleware := api.AuthTokenRequired(mockValidator)

	t.Run("Middleware rejects request without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		}

		err := middleware(handler)(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// Mock для TokenValidator
type MockTokenValidator struct {
	mock.Mock
}

func (m *MockTokenValidator) ValidateToken(ctx context.Context, token string) (int, error) {
	args := m.Called(ctx, token)
	return args.Int(0), args.Error(1)
}
func TestAuthMiddleware_Smoke(t *testing.T) {
	e := echo.New()

	mockValidator := new(MockTokenValidator)
	middleware := api.AuthTokenRequired(mockValidator)

	t.Run("Middleware rejects request without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Не добавляем token в body
		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		}

		err := middleware(handler)(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Middleware accepts request with valid token", func(t *testing.T) {
		body := []byte(`{"token": "valid_token"}`)
		req := httptest.NewRequest(http.MethodGet, "/api/docs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockValidator.On("ValidateToken", mock.Anything, "valid_token").Return(1, nil)

		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		}

		err := middleware(handler)(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockValidator.AssertExpectations(t)
	})
}
