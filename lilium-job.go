package liliumjob

import (
	"fmt"

	"github.com/spyder01/lilium-job/pkg/config"
	"github.com/spyder01/lilium-job/pkg/core"
)

type (
	LiliumJobs = config.LiliumJobs
	LiliumJob  = config.LiliumJobRunnerConfig

	LiliumJobRunner = core.JobRunnerModule
)

func LoadLiliumJobsConfig(path string) (*LiliumJobs, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load job config: %w", err)
	}
	return cfg, nil
}

func New(jobCfg *LiliumJobs) *LiliumJobRunner {
	return core.NewJobRunnerModule(*jobCfg)
}
