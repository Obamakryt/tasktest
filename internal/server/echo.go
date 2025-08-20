package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gomodlag/internal/api"
	"gomodlag/internal/auth"
	"gomodlag/internal/cache"
	"gomodlag/internal/config"
	"gomodlag/internal/docks"
	"gomodlag/internal/logger"
	"gomodlag/internal/storage"
)

func Start(config config.Config) {

	logg := logger.SetupLogger()
	dbPool := storage.StructPool{} // твоя реализация
	pool, err := storage.NewPool(config.DBURL, *logg)
	if err != nil {
		logg.Error("NewPool-ERR", err)
	}
	dbPool.Pool = pool
	MemCache := cache.NewMemoryCache(config.CacheTTL)
	authService := &auth.ServiceDB{&dbPool, *logg, &dbPool}
	dockService := &docks.ServiceDocks{&dbPool, *logg}

	authHandler := &api.AuthRegDelHandler{authService, *logg}
	dockHandler := &api.DockHandler{dockService, MemCache, *logg}

	e := echo.New()

	e.Use(middleware.Logger(), middleware.Recover())

	API := e.Group("/api")

	// маршруты auth
	API.POST("/register", func(c echo.Context) error {
		return authHandler.RegisterHandler(c, config.AdminToken)
	})
	API.POST("/auth", func(c echo.Context) error {
		return authHandler.AuthHandler(c, config.DockTTL)
	})
	API.POST("/auth/:token", authHandler.LogOutHandler)

	// маршруты для docs
	docs := API.Group("/docs")

	docs.POST("", func(c echo.Context) error {
		return dockHandler.UploadDocHandler(c, &dbPool)
	})

	docs.GET("", dockHandler.ListDocsHandler, api.AuthTokenRequired(&dbPool))
	docs.HEAD("", dockHandler.ListDocsHandler, api.AuthTokenRequired(&dbPool))
	docs.GET("/:id", dockHandler.GetDocHandler, api.AuthTokenRequired(&dbPool))
	docs.HEAD("/:id", dockHandler.GetDocHandler, api.AuthTokenRequired(&dbPool))
	docs.DELETE("/:id", dockHandler.DeleteDocHandler, api.AuthTokenRequired(&dbPool))

	e.Logger.Fatal(e.Start(config.ServerPort))
}
