package job

import (
	"errors"
	"strings"
	"time"
)

type Status string

const (
	StatusQueued               Status = "queued"
	StatusRunning              Status = "running"
	StatusWaitingForLogin      Status = "waiting_for_login"
	StatusAwaitingManualAction Status = "awaiting_manual_action"
	StatusFailed               Status = "failed"
	StatusCancelled            Status = "cancelled"
)

type Job struct {
	ID         string
	BusinessID string
	Name       string
	status     Status
	CreatedAt  time.Time
}

func New(id, businessID, name string) (*Job, error) {

	id = strings.TrimSpace(id)
	businessID = strings.TrimSpace(businessID)
	name = strings.TrimSpace(name)

	if id == "" {
		return nil, errors.New("job ID wajib diisi")
	}
	if businessID == "" {
		return nil, errors.New("business ID wajib diisi")
	}
	if name == "" {
		return nil, errors.New("nama WABA wajib diisi")
	}

	return &Job{
		ID:         id,
		BusinessID: businessID,
		Name:       name,
		status:     StatusQueued,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func (j *Job) Fail() error {
	if j.status != StatusRunning {
		return errors.New("job hanya dapat gagal dari status running")
	}

	j.status = StatusFailed
	return nil
}

func (j *Job) Status() Status {
	return j.status
}

func (j *Job) AwaitManualAction() error {
	if j.status != StatusRunning {
		return errors.New("job harus running sebelum menunggu tindakan manual")
	}

	j.status = StatusAwaitingManualAction
	return nil
}

func (j *Job) Start() error {
	if j.status != StatusQueued {
		return errors.New("job dapat dimulai dari opened")
	}

	j.status = StatusRunning
	return nil
}
