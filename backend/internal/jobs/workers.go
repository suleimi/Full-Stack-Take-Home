package jobs

import (
	"context"
	"log"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/riverqueue/river"
)

type CalculateEmissionWorker struct {
	// An embedded WorkerDefaults sets up default methods to fulfill the rest of
	// the Worker interface:
	river.WorkerDefaults[CalculateEmissionArgs]
	s store.IStore
	l *log.Logger
}

func (w *CalculateEmissionWorker) Work(ctx context.Context, job *river.Job[CalculateEmissionArgs]) error {
	logger := w.l
	if logger == nil {
		logger = log.Default()
	}

	startedAt := time.Now()
	logger.Printf("jobs: refresh_emissions started window=%q", job.Args.Window)

	_, err := w.s.DB().ExecContext(ctx,
		"CALL refresh_emissions($1::interval)", job.Args.Window)
	if err != nil {
		logger.Printf("jobs: refresh_emissions failed window=%q duration=%s err=%v", job.Args.Window, time.Since(startedAt), err)
		return err
	}

	logger.Printf("jobs: refresh_emissions finished window=%q duration=%s", job.Args.Window, time.Since(startedAt))
	return err
}

func RegisterWorkers(store store.IStore, logger *log.Logger) *river.Workers {
	workers := river.NewWorkers()
	// AddWorker panics if the worker is already registered or invalid:
	river.AddWorker(workers, &CalculateEmissionWorker{s: store, l: logger})

	return workers
}
