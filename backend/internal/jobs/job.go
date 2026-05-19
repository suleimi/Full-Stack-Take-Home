package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/store"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
)

type JoClient interface {
	EnqueJob(ctx context.Context, job JobArgs) error
	EnqeueTxJob(ctx context.Context, tx *sql.Tx, job JobArgs) error
	Stop(ctx context.Context) error
	Start(ctx context.Context) error
}

type RiverJobClient struct {
	r *river.Client[*sql.Tx]
}

func (jc *RiverJobClient) EnqueJob(ctx context.Context, job JobArgs) error {
	_, err := jc.r.Insert(ctx, job, nil)
	return err
}

func (jc *RiverJobClient) EnqeueTxJob(ctx context.Context, tx *sql.Tx, job JobArgs) error {
	_, err := jc.r.InsertTx(ctx, tx, job, nil)
	return err
}

func (jc *RiverJobClient) Stop(ctx context.Context) error {
	return jc.r.Stop(ctx)
}

func (jc *RiverJobClient) Start(ctx context.Context) error {
	return jc.r.Start(ctx)
}

func NewJobClient(store store.IStore, logger *log.Logger) (JoClient, error) {
	if logger == nil {
		logger = log.Default()
	}

	logger.Print("jobs: configuring River client and periodic refresh workers")

	workers := RegisterWorkers(store, logger)
	c, err := river.NewClient(riverdatabasesql.New(store.DB()), &river.Config{
		Workers:  workers,
		Queues:   map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 4}},
		PollOnly: true,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(15*time.Second),
				func() (river.JobArgs, *river.InsertOpts) {
					return CalculateEmissionArgs{"3 hours", 15 * time.Second}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(
				river.PeriodicInterval(time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return CalculateEmissionArgs{"48 hours", time.Hour}, nil
				}, &river.PeriodicJobOpts{RunOnStart: true}),
		},
	})
	if err != nil {
		return nil, err
	}

	jc := &RiverJobClient{r: c}
	return jc, nil
}
