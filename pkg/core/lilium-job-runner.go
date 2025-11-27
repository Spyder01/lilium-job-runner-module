package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	lilium "github.com/spyder01/lilium-go"
	"github.com/spyder01/lilium-job/pkg/config"
)

type JobRunnerModule struct {
	cfg     config.LiliumJobs
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	stopped bool
}

func NewJobRunnerModule(jobs config.LiliumJobs) *JobRunnerModule {
	return &JobRunnerModule{
		cfg: jobs,
	}
}

func (m *JobRunnerModule) Name() string {
	return "Lilium Job Runner Module"
}

func (m *JobRunnerModule) Priority() uint {
	return 10
}

func (m *JobRunnerModule) Init(app *lilium.AppContext) error {
	app.Logger.Info("Initializing background jobs module...")

	for _, jc := range m.cfg.Jobs {
		if !jc.Enabled {
			app.Logger.Warnf("Job disabled: %s", jc.Name)
			continue
		}
		if jc.Task == nil {
			return fmt.Errorf("job '%s' has no task assigned", jc.Name)
		}
		if jc.MaxConcurrency <= 0 {
			jc.MaxConcurrency = 1
		}
		app.Logger.Infof(
			"Job registered: name=%s interval=%s repeat=%v",
			jc.Name, jc.Interval, jc.Repeat,
		)
	}
	return nil
}

func (m *JobRunnerModule) Start(app *lilium.AppContext) error {
	app.Logger.Info("Starting background jobs...")

	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return fmt.Errorf("job runner already shut down")
	}
	ctx, cancel := context.WithCancel(app.Ctx)
	m.cancel = cancel
	m.mu.Unlock()

	for _, jc := range m.cfg.Jobs {
		if !jc.Enabled {
			continue
		}
		m.startJob(ctx, app, jc)
	}

	return nil
}

func (m *JobRunnerModule) Shutdown(app *lilium.AppContext) error {
	app.Logger.Info("Shutting down background jobs...")

	m.mu.Lock()
	m.stopped = true
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()

	m.wg.Wait()

	app.Logger.Info("All background jobs stopped gracefully")
	return nil
}

// =====================================================================
// INTERNAL: Scheduler and execution
// =====================================================================

func (m *JobRunnerModule) startJob(ctx context.Context, app *lilium.AppContext, jc *config.LiliumJobRunnerConfig) {
	m.wg.Add(1)

	go func() {
		defer m.wg.Done()

		sem := make(chan struct{}, jc.MaxConcurrency)

		runOnce := func() {
			sem <- struct{}{}
			m.wg.Add(1)
			go func() {
				defer m.wg.Done()
				defer func() { <-sem }()
				m.runJob(ctx, app, jc)
			}()
		}

		runOnce()
		if !jc.Repeat {
			return
		}

		ticker := time.NewTicker(jc.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				app.Logger.Infof("Job stopped: %s", jc.Name)
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()
}

func (m *JobRunnerModule) runJob(parentCtx context.Context, app *lilium.AppContext, jc *config.LiliumJobRunnerConfig) {
	backoff := defaultDuration(jc.InitialBackoff, 1*time.Second)
	maxBackoff := defaultDuration(jc.MaxBackoff, 30*time.Second)
	tries := 0

	for {
		select {
		case <-parentCtx.Done():
			return
		default:
		}

		originalCtx := app.Ctx
		if jc.Timeout > 0 {
			tCtx, cancel := context.WithTimeout(parentCtx, jc.Timeout)
			app.Ctx = tCtx
			defer cancel()
		}

		tries++
		app.Logger.Infof("Job run begin: %s (attempt %d)", jc.Name, tries)

		start := time.Now()
		err := safeRun(app, jc.Task)
		elapsed := time.Since(start)

		app.Ctx = originalCtx

		if err == nil {
			app.Logger.Infof("Job success: %s (%s)", jc.Name, elapsed)
			return
		}

		app.Logger.Errorf("Job failed: %s (attempt=%d) err=%v", jc.Name, tries, err)

		if tries > jc.Retries {
			app.Logger.Errorf("Job giving up: %s", jc.Name)
			return
		}

		sleep := backoff
		if sleep > maxBackoff {
			sleep = maxBackoff
		} else {
			backoff *= 2
		}

		select {
		case <-parentCtx.Done():
			return
		case <-time.After(sleep):
		}
	}
}

func safeRun(app *lilium.AppContext, task lilium.LiliumTask) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return task(app)
}

func defaultDuration(v, def time.Duration) time.Duration {
	if v <= 0 {
		return def
	}
	return v
}

func (m *JobRunnerModule) RegisterTask(name string, task lilium.LiliumTask) error {
	if task == nil {
		return fmt.Errorf("task for job '%s' cannot be nil", name)
	}

	// Lock while updating shared state
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, jc := range m.cfg.Jobs {
		if jc.Name == name {
			if jc.Task != nil {
				// Warn but override
				fmt.Printf("[JobRunner] Warning: task already assigned for job '%s', overriding\n", name)
			}

			jc.Task = task
			return nil
		}
	}

	return fmt.Errorf("no such job exists in config: '%s'", name)
}
