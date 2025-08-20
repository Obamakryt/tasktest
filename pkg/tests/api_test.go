package tests

import (
	"context"
	"encoding/json"
	"gomodlag/internal/storage"
	"gomodlag/pkg"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegister_Success тест успешной регистрации
func TestRegister_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Успешная регистрация
	login, err := s.Register(ctx, "hashed_password_123", "testuser")

	assert.NoError(t, err)
	assert.Equal(t, "testuser", login)
}

// TestRegister_Duplicate тест дубликата пользователя
func TestRegister_Duplicate(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Первая регистрация - успех
	_, err := s.Register(ctx, "pass1", "duplicate_user")
	assert.NoError(t, err)

	// Вторая регистрация - ошибка
	_, err = s.Register(ctx, "pass2", "duplicate_user")
	assert.Error(t, err)
}

func TestLogin_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Сначала регистрируем с хешированным паролем
	password := "correct_pass"
	hashedPassword := pkg.CreateHash(password) // Хешируем пароль
	username := "login_user"
	_, err := s.Register(ctx, hashedPassword, username)
	require.NoError(t, err)

	// Пытаемся залогиниться с хешированным паролем
	token := "test_token_123"
	err = s.Login(ctx, hashedPassword, username, token, time.Hour)

	assert.NoError(t, err)
}

// TestLogin_WrongCredentials тест неправильных credentials
func TestLogin_WrongCredentials(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя с хешированным паролем
	hashedPassword := pkg.CreateHash("right_pass")
	username := "test_user"
	_, err := s.Register(ctx, hashedPassword, username)
	require.NoError(t, err)

	// Пытаемся залогиниться с неправильным хешированным паролем
	wrongHashedPassword := pkg.CreateHash("wrong_pass")
	err = s.Login(ctx, wrongHashedPassword, username, "token", time.Hour)

	assert.Error(t, err)
	assert.Equal(t, storage.Invaliddata, err)
}

// TestValidateToken_Success тест валидации токена
func TestValidateToken_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем и логиним
	password := "hashed_pass"
	username := "token_user"
	_, err := s.Register(ctx, password, username)
	require.NoError(t, err)

	token := "valid_token_123"
	err = s.Login(ctx, password, username, token, time.Hour)
	require.NoError(t, err)

	// Валидируем токен
	userID, err := s.ValidateToken(ctx, token)

	assert.NoError(t, err)
	assert.Greater(t, userID, 0) // ID должен быть положительным
}
func TestValidateToken_Expired(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя
	password := "hashed_pass"
	username := "expired_user"
	_, err := s.Register(ctx, password, username)
	require.NoError(t, err)

	// Получаем ID пользователя
	var userID int
	err = s.Pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", username).Scan(&userID)
	require.NoError(t, err)

	// Создаем просроченный токен вручную - В UTC!
	expiredToken := "expired_token"

	// Используем UTC время для consistency
	nowUTC := time.Now().UTC()
	expireTime := nowUTC.Add(-time.Hour) // истек час назад в UTC
	createTime := nowUTC.Add(-2 * time.Hour)

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO sessions (token, created_at, expire_at, user_id)
		VALUES ($1, $2, $3, $4)
	`, expiredToken, createTime, expireTime, userID)
	require.NoError(t, err)

	// Пытаемся валидировать
	userIDResult, err := s.ValidateToken(ctx, expiredToken)

	assert.Error(t, err)
	assert.Equal(t, storage.Invalidtoken, err)
	assert.Equal(t, 0, userIDResult)
}

// TestValidateToken_NotFound тест несуществующего токена
func TestValidateToken_NotFound(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Пытаемся валидировать несуществующий токен
	_, err := s.ValidateToken(ctx, "nonexistent_token")

	assert.Error(t, err)
	assert.Equal(t, storage.Invalidtoken, err)
}

// TestNewDocs_Success тест создания документа
func TestNewDocs_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя
	_, err := s.Register(ctx, "pass", "doc_owner")
	require.NoError(t, err)

	var userID int
	err = s.Pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", "doc_owner").Scan(&userID)
	require.NoError(t, err)

	// Начинаем транзакцию
	tx, err := s.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Создаем документ
	doc := storage.Dock{
		Id:       uuid.New(),
		Name:     "Test Document",
		Public:   true,
		IsFile:   false,
		Mime:     "application/json",
		Json:     json.RawMessage(`{"test": "data"}`),
		Filepath: "",
		OwnerId:  userID,
	}

	success, err := s.NewDocs(ctx, doc, tx)

	assert.NoError(t, err)
	assert.True(t, success)
}

// TestGetDock_Success тест получения документов
func TestGetDock_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя
	_, err := s.Register(ctx, "pass", "filter_user")
	require.NoError(t, err)

	var userID int
	err = s.Pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", "filter_user").Scan(&userID)
	require.NoError(t, err)

	// Создаем тестовый документ
	docID := uuid.New()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO documents (id, name, public, is_file, mime, json_data, own_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, docID, "Test Doc", true, false, "text/plain", `{}`, userID)
	require.NoError(t, err)

	// Пытаемся получить документы
	filter := storage.GetDock{
		Id:    userID,
		Key:   "name",
		Value: "Test Doc",
		Limit: 10,
	}

	docs, err := s.GetDock(ctx, filter)

	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Test Doc", docs[0].Name)
}

// TestGetDockById_Success тест получения документа по ID
func TestGetDockById_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя
	_, err := s.Register(ctx, "pass", "getbyid_user")
	require.NoError(t, err)

	var userID int
	err = s.Pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", "getbyid_user").Scan(&userID)
	require.NoError(t, err)

	// Создаем документ
	docID := uuid.New()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO documents (id, name, public, is_file, mime, json_data, own_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, docID, "Specific Doc", true, false, "text/plain", `{"key": "value"}`, userID)
	require.NoError(t, err)

	// Получаем документ по ID
	doc, err := s.GetDockById(ctx, userID, docID)

	assert.NoError(t, err)
	assert.Equal(t, "Specific Doc", doc.Name)
	assert.Equal(t, docID, doc.ID)
}

// TestDeleteDock_Success тест удаления документа
func TestDeleteDock_Success(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя
	_, err := s.Register(ctx, "pass", "delete_user")
	require.NoError(t, err)

	var userID int
	err = s.Pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", "delete_user").Scan(&userID)
	require.NoError(t, err)

	// Создаем документ
	docID := uuid.New()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO documents (id, name, public, is_file, mime, json_data, own_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, docID, "To Delete", true, false, "text/plain", `{}`, userID)
	require.NoError(t, err)

	// Удаляем документ
	err = s.DeleteDock(ctx, userID, docID)

	assert.NoError(t, err)

	// Проверяем, что документ удален
	var count int
	err = s.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM documents WHERE id = $1", docID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestDeleteDock_NotFound тест удаления несуществующего документа
func TestDeleteDock_NotFound(t *testing.T) {
	s := setupTestDB(t)
	defer cleanupTestDB(t, s)

	ctx := context.Background()

	// Регистрируем пользователя
	_, err := s.Register(ctx, "pass", "notfound_user")
	require.NoError(t, err)

	var userID int
	err = s.Pool.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", "notfound_user").Scan(&userID)
	require.NoError(t, err)

	// Пытаемся удалить несуществующий документ
	nonExistentID := uuid.New()
	err = s.DeleteDock(ctx, userID, nonExistentID)

	assert.Error(t, err)
	assert.Equal(t, storage.Forbidden, err)
}
