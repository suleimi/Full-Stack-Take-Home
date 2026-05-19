package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

type JobArgs interface {
	Kind() string
}

type CalculateEmissionArgs struct {
	Window  string        `json:"window"` // e.g. "3 hours" / "48 hours"
	RunFreq time.Duration `json:"frequency"`
}

func (CalculateEmissionArgs) Kind() string { return "calc:emmission" }

func (c CalculateEmissionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByPeriod: c.RunFreq},
	}
}
