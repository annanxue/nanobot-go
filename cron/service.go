package cron

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type CronService struct {
	storePath string
	onJob     func(*CronJob) (string, error)
	store     *CronStore
	cron      *cron.Cron
	running   bool
	mu        sync.RWMutex
}

func NewCronService(storePath string) *CronService {
	return &CronService{
		storePath: storePath,
		cron:      cron.New(cron.WithSeconds()),
		running:   false,
	}
}

func (cs *CronService) loadStore() *CronStore {
	cs.mu.RLock()
	if cs.store != nil {
		cs.mu.RUnlock()
		return cs.store
	}
	cs.mu.RUnlock()

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.store != nil {
		return cs.store
	}

	if _, err := os.Stat(cs.storePath); err == nil {
		data, err := os.ReadFile(cs.storePath)
		if err == nil {
			var store CronStore
			if err := json.Unmarshal(data, &store); err == nil {
				cs.store = &store
				return cs.store
			}
		}
		logrus.WithError(err).Warn("Failed to load cron store")
	}

	cs.store = &CronStore{Version: 1, Jobs: []CronJob{}}
	return cs.store
}

func (cs *CronService) saveStore() error {
	cs.mu.RLock()
	store := cs.store
	cs.mu.RUnlock()

	if store == nil {
		return nil
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(cs.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(cs.storePath, data, 0644)
}

func (cs *CronService) Start() error {
	cs.mu.Lock()
	if cs.running {
		cs.mu.Unlock()
		return nil
	}
	cs.mu.Unlock()

	// Load store without holding the lock
	cs.loadStore()

	// Recompute next runs
	cs.mu.Lock()
	cs.recomputeNextRuns()
	err := cs.saveStore()
	cs.running = true
	cs.cron.Start()
	logrus.Infof("Cron service started with %d jobs", len(cs.store.Jobs))
	cs.mu.Unlock()

	return err
}

func (cs *CronService) Stop() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if !cs.running {
		return
	}

	ctx := cs.cron.Stop()
	<-ctx.Done()
	cs.running = false
}

func (cs *CronService) recomputeNextRuns() {
	now := time.Now().UnixMilli()
	for i := range cs.store.Jobs {
		if cs.store.Jobs[i].Enabled {
			nextRun := cs.computeNextRun(&cs.store.Jobs[i].Schedule, now)
			cs.store.Jobs[i].State.NextRunAtMs = nextRun
		}
	}
}

func (cs *CronService) computeNextRun(schedule *CronSchedule, nowMs int64) *int64 {
	if schedule.Kind == ScheduleKindAt {
		if schedule.AtMs != nil && int64(*schedule.AtMs) > nowMs {
			atMs := int64(*schedule.AtMs)
			return &atMs
		}
		return nil
	}

	if schedule.Kind == ScheduleKindEvery {
		if schedule.EveryMs != nil && *schedule.EveryMs > 0 {
			next := nowMs + int64(*schedule.EveryMs)
			return &next
		}
		return nil
	}

	if schedule.Kind == ScheduleKindCron && schedule.Expr != "" {
		scheduleParser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if schedule, err := scheduleParser.Parse(schedule.Expr); err == nil {
			nextTime := schedule.Next(time.Unix(nowMs/1000, 0))
			nextMs := nextTime.UnixMilli()
			return &nextMs
		}
	}

	return nil
}

func (cs *CronService) ListJobs(includeDisabled bool) []CronJob {
	// Load store without holding the lock
	store := cs.loadStore()

	// Now get the lock to access the store
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if includeDisabled {
		return store.Jobs
	}

	var jobs []CronJob
	for _, job := range store.Jobs {
		if job.Enabled {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (cs *CronService) AddJob(name string, schedule CronSchedule, message string, deliver bool, channel, to string, deleteAfterRun bool) (*CronJob, error) {
	cs.mu.Lock()
	if cs.store == nil {
		cs.mu.Unlock()
		// Load store without holding the lock
		cs.loadStore()
		cs.mu.Lock()
	}

	store := cs.store
	now := time.Now().UnixMilli()

	job := CronJob{
		ID:       generateID(),
		Name:     name,
		Enabled:  true,
		Schedule: schedule,
		Payload: CronPayload{
			Kind:    PayloadKindAgentTurn,
			Message: message,
			Deliver: deliver,
			Channel: channel,
			To:      to,
		},
		State: CronJobState{
			NextRunAtMs: cs.computeNextRun(&schedule, now),
		},
		CreatedAtMs:    int(now),
		UpdatedAtMs:    int(now),
		DeleteAfterRun: deleteAfterRun,
	}

	store.Jobs = append(store.Jobs, job)
	err := cs.saveStore()

	logrus.Infof("Cron: added job '%s' (%s)", name, job.ID)
	cs.mu.Unlock()

	return &job, err
}

func (cs *CronService) RemoveJob(jobID string) bool {
	cs.mu.Lock()
	if cs.store == nil {
		cs.mu.Unlock()
		// Load store without holding the lock
		cs.loadStore()
		cs.mu.Lock()
	}

	store := cs.store
	before := len(store.Jobs)
	var jobs []CronJob
	for _, job := range store.Jobs {
		if job.ID != jobID {
			jobs = append(jobs, job)
		}
	}
	store.Jobs = jobs
	removed := len(store.Jobs) < before

	if removed {
		cs.saveStore()
		logrus.Infof("Cron: removed job %s", jobID)
	}

	cs.mu.Unlock()

	return removed
}

func (cs *CronService) EnableJob(jobID string, enabled bool) *CronJob {
	cs.mu.Lock()
	if cs.store == nil {
		cs.mu.Unlock()
		// Load store without holding the lock
		cs.loadStore()
		cs.mu.Lock()
	}

	store := cs.store
	for i := range store.Jobs {
		if store.Jobs[i].ID == jobID {
			store.Jobs[i].Enabled = enabled
			store.Jobs[i].UpdatedAtMs = int(time.Now().UnixMilli())
			if enabled {
				now := time.Now().UnixMilli()
				store.Jobs[i].State.NextRunAtMs = cs.computeNextRun(&store.Jobs[i].Schedule, now)
			} else {
				store.Jobs[i].State.NextRunAtMs = nil
			}
			cs.saveStore()
			job := store.Jobs[i]
			cs.mu.Unlock()
			return &job
		}
	}

	cs.mu.Unlock()

	return nil
}

func (cs *CronService) SetOnJob(callback func(*CronJob) (string, error)) {
	cs.onJob = callback
}

func (cs *CronService) RunJob(jobID string, force bool) bool {
	cs.mu.Lock()
	if cs.store == nil {
		cs.mu.Unlock()
		// Load store without holding the lock
		cs.loadStore()
		cs.mu.Lock()
	}

	store := cs.store
	var job *CronJob
	for i := range store.Jobs {
		if store.Jobs[i].ID == jobID {
			job = &store.Jobs[i]
			break
		}
	}
	cs.mu.Unlock()

	if job == nil {
		return false
	}

	if !force && !job.Enabled {
		return false
	}

	cs.executeJob(job)

	// Save store with lock
	cs.mu.Lock()
	cs.saveStore()
	cs.mu.Unlock()

	return true
}

func (cs *CronService) executeJob(job *CronJob) {
	startMs := time.Now().UnixMilli()
	logrus.Infof("Cron: executing job '%s' (%s)", job.Name, job.ID)

	var err error
	if cs.onJob != nil {
		_, err = cs.onJob(job)
	}

	if err != nil {
		job.State.LastStatus = JobStatusError
		job.State.LastError = err.Error()
		logrus.Errorf("Cron: job '%s' failed: %v", job.Name, err)
	} else {
		job.State.LastStatus = JobStatusOK
		job.State.LastError = ""
		logrus.Infof("Cron: job '%s' completed", job.Name)
	}

	job.State.LastRunAtMs = &startMs
	job.UpdatedAtMs = int(time.Now().UnixMilli())

	if job.Schedule.Kind == ScheduleKindAt {
		if job.DeleteAfterRun {
			// Remove job with lock
			cs.mu.Lock()
			store := cs.store
			if store != nil {
				var jobs []CronJob
				for _, j := range store.Jobs {
					if j.ID != job.ID {
						jobs = append(jobs, j)
					}
				}
				store.Jobs = jobs
			}
			cs.mu.Unlock()
		} else {
			job.Enabled = false
			job.State.NextRunAtMs = nil
		}
	} else {
		now := time.Now().UnixMilli()
		job.State.NextRunAtMs = cs.computeNextRun(&job.Schedule, now)
	}
}

func (cs *CronService) Status() map[string]interface{} {
	// Load store without holding the lock
	cs.loadStore()

	// Now get the lock to access the store
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return map[string]interface{}{
		"enabled":         cs.running,
		"jobs":            len(cs.store.Jobs),
		"next_wake_at_ms": cs.getNextWakeMs(),
	}
}

func (cs *CronService) getNextWakeMs() *int64 {
	store := cs.store
	var minNext *int64
	for _, job := range store.Jobs {
		if job.Enabled && job.State.NextRunAtMs != nil {
			if minNext == nil || *job.State.NextRunAtMs < *minNext {
				minNext = job.State.NextRunAtMs
			}
		}
	}
	return minNext
}

func generateID() string {
	return time.Now().Format("20060102150405")
}
