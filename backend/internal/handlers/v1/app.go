package v1

import (
	"context"
	"log"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/cache"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/config"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/jobs"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/logger"
	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
)

type AppV1 struct {
	Store     store.IStore
	Cache     cache.Cache
	JobClient jobs.JoClient
	Logger    *log.Logger
	Config    *config.Config
}

func NewApp(c *config.Config) (*AppV1, error) {
	appLogger := logger.NewStruturedLogger(c.LogLevel)
	appStore, err := store.NewStore(c.PostgresHost, c.PostgresPort, c.PostgresDB, c.PostgresUser, c.PostgresPassword)
	if err != nil {
		return nil, err
	}
	appCache, err := cache.NewCache(c.RedissHost, c.RedisPort)
	if err != nil {
		return nil, err
	}

	appJobClient, err := jobs.NewJobClient(appStore, appLogger)
	if err != nil {
		return nil, err
	}

	return &AppV1{
		Store:     appStore,
		Cache:     appCache,
		Logger:    appLogger,
		JobClient: appJobClient,
		Config:    c,
	}, nil
}

func (a *AppV1) StartBackgroundJobWorker(ctx context.Context) error {
	return a.JobClient.Start(ctx)
}

func (a *AppV1) Shutdown(ctx context.Context) error {
	err := a.JobClient.Stop(ctx)
	if err != nil {
		return err
	}

	err = a.Store.DB().Close()
	if err != nil {
		return err
	}

	err = a.Cache.Close()
	if err != nil {
		return err
	}

	return nil
}
