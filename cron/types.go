package cron

type ScheduleKind string

const (
	ScheduleKindAt    ScheduleKind = "at"
	ScheduleKindEvery ScheduleKind = "every"
	ScheduleKindCron  ScheduleKind = "cron"
)

type CronSchedule struct {
	Kind    ScheduleKind `json:"kind"`
	AtMs    *int64       `json:"at_ms,omitempty"`
	EveryMs *int64       `json:"every_ms,omitempty"`
	Expr    string       `json:"expr,omitempty"`
	TZ      string       `json:"tz,omitempty"`
}

type PayloadKind string

const (
	PayloadKindSystemEvent PayloadKind = "system_event"
	PayloadKindAgentTurn   PayloadKind = "agent_turn"
)

type CronPayload struct {
	Kind    PayloadKind `json:"kind"`
	Message string      `json:"message"`
	Deliver bool        `json:"deliver"`
	Channel string      `json:"channel,omitempty"`
	To      string      `json:"to,omitempty"`
}

type JobStatus string

const (
	JobStatusOK      JobStatus = "ok"
	JobStatusError   JobStatus = "error"
	JobStatusSkipped JobStatus = "skipped"
)

type CronJobState struct {
	NextRunAtMs *int64    `json:"next_run_at_ms,omitempty"`
	LastRunAtMs *int64    `json:"last_run_at_ms,omitempty"`
	LastStatus  JobStatus `json:"last_status,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
}

type CronJob struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Enabled        bool         `json:"enabled"`
	Schedule       CronSchedule `json:"schedule"`
	Payload        CronPayload  `json:"payload"`
	State          CronJobState `json:"state"`
	CreatedAtMs    int          `json:"created_at_ms"`
	UpdatedAtMs    int          `json:"updated_at_ms"`
	DeleteAfterRun bool         `json:"delete_after_run"`
}

type CronStore struct {
	Version int       `json:"version"`
	Jobs    []CronJob `json:"jobs"`
}
