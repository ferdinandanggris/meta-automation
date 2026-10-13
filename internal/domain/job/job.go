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
		return nil, &ValidationError{
			Field:   "id",
			Message: "wajib diisi",
		}
	}

	if businessID == "" {
		return nil, &ValidationError{
			Field:   "business_id",
			Message: "wajib diisi",
		}
	}

	if name == "" {
		return nil, &ValidationError{
			Field:   "name",
			Message: "wajib diisi",
		}
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

func (j *Job) Cancel() error {
	if j.status != StatusQueued && j.status != StatusRunning {
		return errors.New("job hanya bisa dibatalkan dari queued atau running")
	}

	j.status = StatusCancelled
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

type Snapshot struct {
	ID         string
	BusinessID string
	Name       string
	Status     Status
	CreatedAt  time.Time
}

func (j *Job) Snapshot() Snapshot {
	return Snapshot{
		ID:         j.ID,
		BusinessID: j.BusinessID,
		Name:       j.Name,
		Status:     j.status,
		CreatedAt:  j.CreatedAt,
	}
}
