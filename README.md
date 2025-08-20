##Env file is visiable!

internal
api           HTTP handlers + middleware
auth          Аутентификация
docks         Логика документов  
storage       PostgreSQL layer
cache         кеш
config        Конфигурация
cfg

POST /api/register - Регистрация

POST /api/auth - Логин

POST /api/auth/:token - Логаут

Документы
POST /api/docs - Загрузить документ
HEAD  /api/docs - http.statusok
GET /api/docs - Список документов
HEAD /api/docs/:id - http.statusok
GET /api/docs/:id - Получить документ

DELETE /api/docs/:id - Удалить документ

База данных
PostgreSQL с таблицами:

users - пользователи

sessions - активные сессии

documents - документы

document_grants - права доступа


In-memory кеш с автоматической очисткой:

Файлы: 1 час

JSON данные: 1 час

Инвалидация при:

Удалении документа


Запуск 
go run cmd/main.go

Тесты

go test ./pkg/tests/api_test/... -v
go test ./pkg/tests/mock_test/... -v
