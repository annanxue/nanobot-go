package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/nanobotgo/utils"
)

type command struct {
	op             string
	jobID          string
	job            *CronJob
	name           string
	schedule       CronSchedule
	message        string
	deliver        bool
	channel        string
	to             string
	deleteAfterRun bool
	enabled        bool
	force          bool
	resultCh       chan interface{}
}

type CronService struct {
	storePath  string
	onJob      func(*CronJob) (string, error)
	store      *CronStore
	cron       *cron.Cron
	running    bool
	cmdCh      chan command
	stopCh     chan struct{}
	jobEntries map[string]cron.EntryID
	entriesMu  sync.Mutex
}

func NewCronService(storePath string) *CronService {
	cs := &CronService{
		storePath:  storePath,
		cron:       cron.New(cron.WithSeconds()),
		cmdCh:      make(chan command, 100),
		stopCh:     make(chan struct{}),
		jobEntries: make(map[string]cron.EntryID),
	}
	cs.doLoadStore()
	go cs.run()
	return cs
}

func (cs *CronService) run() {
	for {
		select {
		case <-cs.stopCh:
			return
		case cmd := <-cs.cmdCh:
			cs.handleCommand(cmd)
		}
	}
}

func (cs *CronService) handleCommand(cmd command) {
	switch cmd.op {
	case "load":
		cs.doLoadStore()
		cmd.resultCh <- cs.store

	case "save":
		err := cs.doSaveStore()
		cmd.resultCh <- err

	case "add":
		cs.doAddJob(cmd.name, cmd.schedule, cmd.message, cmd.deliver, cmd.channel, cmd.to, cmd.deleteAfterRun, cmd.resultCh)

	case "remove":
		result := cs.doRemoveJob(cmd.jobID)
		cmd.resultCh <- result

	case "enable":
		cs.doEnableJob(cmd.jobID, cmd.enabled, cmd.resultCh)

	case "list":
		result := cs.doListJobs(cmd.enabled)
		cmd.resultCh <- result

	case "run":
		result := cs.doRunJob(cmd.jobID, cmd.force)
		cmd.resultCh <- result

	case "status":
		result := cs.doStatus()
		cmd.resultCh <- result

	case "recompute":
		cs.doRecomputeNextRuns()
		cmd.resultCh <- nil
	}
}

func (cs *CronService) loadStore() *CronStore {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "load", resultCh: resultCh}
	result := <-resultCh
	return result.(*CronStore)
}

func (cs *CronService) saveStore() error {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "save", resultCh: resultCh}
	err := <-resultCh
	if err != nil {
		return err.(error)
	}
	return nil
}

func (cs *CronService) doLoadStore() {
	if cs.store != nil {
		return
	}

	if _, err := os.Stat(cs.storePath); err == nil {
		data, err := os.ReadFile(cs.storePath)
		if err == nil {
			var store CronStore
			if err := json.Unmarshal(data, &store); err == nil {
				cs.store = &store
				return
			}
		}
		utils.Log.WithError(err).Warn("Failed to load cron store")
	}

	cs.store = &CronStore{Version: 1, Jobs: []CronJob{}}
}

func (cs *CronService) doSaveStore() error {
	if cs.store == nil {
		return nil
	}

	data, err := json.MarshalIndent(cs.store, "", "  ")
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
	if cs.running {
		return nil
	}

	cs.doLoadStore()
	cs.doRecomputeNextRuns()
	cs.scheduleAllJobs()

	err := cs.doSaveStore()

	cs.cron.Start()
	utils.Log.Infof("Cron service started with %d jobs", len(cs.store.Jobs))
	cs.running = true

	return err
}

func (cs *CronService) Stop() {
	if !cs.running {
		return
	}

	ctx := cs.cron.Stop()
	<-ctx.Done()
	cs.running = false

	close(cs.stopCh)
}

func (cs *CronService) scheduleAllJobs() {
	cs.entriesMu.Lock()
	defer cs.entriesMu.Unlock()

	for _, job := range cs.store.Jobs {
		if job.Enabled {
			cs.scheduleJob(&job)
		}
	}
}

func (cs *CronService) scheduleJob(job *CronJob) {
	var spec string
	switch job.Schedule.Kind {
	case ScheduleKindEvery:
		if job.Schedule.EveryMs != nil {
			seconds := *job.Schedule.EveryMs / 1000
			spec = fmt.Sprintf("@every %ds", seconds)
		}
	case ScheduleKindCron:
		if job.Schedule.Expr != "" {
			spec = job.Schedule.Expr
		}
	case ScheduleKindAt:
		if job.Schedule.AtMs != nil {
			atTime := time.UnixMilli(int64(*job.Schedule.AtMs))
			delay := time.Until(atTime)
			if delay > 0 {
				spec = fmt.Sprintf("@every %s", delay)
			}
		}
	}

	if spec != "" {
		entryID, err := cs.cron.AddFunc(spec, func() {
			cs.executeJob(job)
		})
		if err != nil {
			utils.Log.Errorf("Failed to schedule job %s: %v", job.ID, err)
			return
		}
		cs.jobEntries[job.ID] = entryID
		utils.Log.Infof("Scheduled job %s with spec %s", job.ID, spec)
	}
}

func (cs *CronService) unscheduleJob(jobID string) {
	cs.entriesMu.Lock()
	defer cs.entriesMu.Unlock()

	if entryID, ok := cs.jobEntries[jobID]; ok {
		cs.cron.Remove(entryID)
		delete(cs.jobEntries, jobID)
	}
}

func (cs *CronService) recomputeNextRuns() {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "recompute", resultCh: resultCh}
	<-resultCh
}

func (cs *CronService) doRecomputeNextRuns() {
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
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "list", enabled: includeDisabled, resultCh: resultCh}
	result := <-resultCh
	return result.([]CronJob)
}

func (cs *CronService) doListJobs(includeDisabled bool) []CronJob {
	if includeDisabled {
		return cs.store.Jobs
	}

	var jobs []CronJob
	for _, job := range cs.store.Jobs {
		if job.Enabled {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (cs *CronService) AddJob(name string, schedule CronSchedule, message string, deliver bool, channel, to string, deleteAfterRun bool) (*CronJob, error) {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{
		op:             "add",
		name:           name,
		schedule:       schedule,
		message:        message,
		deliver:        deliver,
		channel:        channel,
		to:             to,
		deleteAfterRun: deleteAfterRun,
		resultCh:       resultCh,
	}
	job := <-resultCh
	errVal := <-resultCh
	if errVal != nil {
		return job.(*CronJob), errVal.(error)
	}
	return job.(*CronJob), nil
}

func (cs *CronService) doAddJob(name string, schedule CronSchedule, message string, deliver bool, channel, to string, deleteAfterRun bool, resultCh chan interface{}) {
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

	cs.store.Jobs = append(cs.store.Jobs, job)

	if cs.running {
		cs.scheduleJob(&job)
	}

	utils.Log.Infof("Cron: added job '%s' (%s)", name, job.ID)

	err := cs.doSaveStore()

	resultCh <- &job
	resultCh <- err
}

func (cs *CronService) RemoveJob(jobID string) bool {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "remove", jobID: jobID, resultCh: resultCh}
	result := <-resultCh
	return result.(bool)
}

func (cs *CronService) doRemoveJob(jobID string) bool {
	cs.unscheduleJob(jobID)

	before := len(cs.store.Jobs)
	var jobs []CronJob
	for _, job := range cs.store.Jobs {
		if job.ID != jobID {
			jobs = append(jobs, job)
		}
	}
	cs.store.Jobs = jobs
	removed := len(cs.store.Jobs) < before

	if removed {
		cs.doSaveStore()
		utils.Log.Infof("Cron: removed job %s", jobID)
	}

	return removed
}

func (cs *CronService) EnableJob(jobID string, enabled bool) *CronJob {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "enable", jobID: jobID, enabled: enabled, resultCh: resultCh}
	result := <-resultCh
	if result == nil {
		return nil
	}
	return result.(*CronJob)
}

func (cs *CronService) doEnableJob(jobID string, enabled bool, resultCh chan interface{}) {
	for i := range cs.store.Jobs {
		if cs.store.Jobs[i].ID == jobID {
			cs.store.Jobs[i].Enabled = enabled
			cs.store.Jobs[i].UpdatedAtMs = int(time.Now().UnixMilli())

			if enabled {
				now := time.Now().UnixMilli()
				cs.store.Jobs[i].State.NextRunAtMs = cs.computeNextRun(&cs.store.Jobs[i].Schedule, now)
				if cs.running {
					cs.scheduleJob(&cs.store.Jobs[i])
				}
			} else {
				cs.store.Jobs[i].State.NextRunAtMs = nil
				cs.unscheduleJob(jobID)
			}

			cs.doSaveStore()
			job := cs.store.Jobs[i]
			resultCh <- &job
			return
		}
	}

	resultCh <- nil
}

func (cs *CronService) SetOnJob(callback func(*CronJob) (string, error)) {
	cs.onJob = callback
}

func (cs *CronService) RunJob(jobID string, force bool) bool {
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "run", jobID: jobID, force: force, resultCh: resultCh}
	result := <-resultCh
	return result.(bool)
}

func (cs *CronService) doRunJob(jobID string, force bool) bool {
	var job *CronJob
	for i := range cs.store.Jobs {
		if cs.store.Jobs[i].ID == jobID {
			job = &cs.store.Jobs[i]
			break
		}
	}

	if job == nil {
		return false
	}

	if !force && !job.Enabled {
		return false
	}

	cs.executeJob(job)

	cs.doSaveStore()

	return true
}

func (cs *CronService) executeJob(job *CronJob) {
	startMs := time.Now().UnixMilli()
	utils.Log.Infof("Cron: executing job '%s' (%s)", job.Name, job.ID)

	var err error
	if cs.onJob != nil {
		_, err = cs.onJob(job)
	}

	if err != nil {
		job.State.LastStatus = JobStatusError
		job.State.LastError = err.Error()
		utils.Log.Errorf("Cron: job '%s' failed: %v", job.Name, err)
	} else {
		job.State.LastStatus = JobStatusOK
		job.State.LastError = ""
		utils.Log.Infof("Cron: job '%s' completed", job.Name)
	}

	job.State.LastRunAtMs = &startMs
	job.UpdatedAtMs = int(time.Now().UnixMilli())

	if job.Schedule.Kind == ScheduleKindAt {
		if job.DeleteAfterRun {
			var jobs []CronJob
			for _, j := range cs.store.Jobs {
				if j.ID != job.ID {
					jobs = append(jobs, j)
				}
			}
			cs.store.Jobs = jobs
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
	resultCh := make(chan interface{}, 1)
	cs.cmdCh <- command{op: "status", resultCh: resultCh}
	result := <-resultCh
	return result.(map[string]interface{})
}

func (cs *CronService) doStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":         cs.running,
		"jobs":            len(cs.store.Jobs),
		"next_wake_at_ms": cs.getNextWakeMs(),
	}
}

func (cs *CronService) getNextWakeMs() *int64 {
	var minNext *int64
	for _, job := range cs.store.Jobs {
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
