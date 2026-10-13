package onboarding

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"meta-automation/internal/domain/job"
)

type JobRunner interface {
	Run(ctx context.Context, task *job.Job) error
}

type Queue struct {
	runner JobRunner
	tasks  chan *job.Job

	mu      sync.Mutex
	running bool
	closed  bool
}

var (
	ErrQueueClosed = errors.New("antrean sudah ditutup")
	ErrQueueFull   = errors.New("antrean penuh")
)

func NewQueue(runner JobRunner) *Queue {
	return &Queue{
		runner: runner,
		tasks:  make(chan *job.Job, 10),
	}
}

func (q *Queue) Enqueue(ctx context.Context, task *job.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("masukkan job ke antrean: %w", err)
	}

	if q.closed {
		return ErrQueueClosed
	}

	select {
	case q.tasks <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (q *Queue) processNext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("ambil job dari antrean: %w", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("ambil job dari antrean: %w", ctx.Err())
	case task := <-q.tasks:
		return q.runner.Run(ctx, task)
	}
}

func (q *Queue) Run(
	ctx context.Context,
	onJobError func(error),
) error {
	if onJobError == nil {
		return errors.New("handler error job wajib diisi")
	}

	q.mu.Lock()

	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}

	if q.running {
		q.mu.Unlock()
		return errors.New("worker antrean sudah berjalan")
	}
	q.running = true
	q.mu.Unlock()

	defer func() {
		q.mu.Lock()
		q.running = false
		q.mu.Unlock()
	}()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := q.processNext(ctx)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			onJobError(err)
		}
	}
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.closed = true
}

func (s *Service) CancelQueued(task *job.Job) error {
	if task.Status() != job.StatusQueued {
		return errors.New("hanya job queued yang dapat dibatalkan lewat antrean")
	}

	if err := task.Cancel(); err != nil {
		return fmt.Errorf("batalkan job antre: %w", err)
	}

	s.store.Save(task.Snapshot())
	return nil
}

func (q *Queue) Drain() ([]*job.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		return nil, errors.New("antrean harus ditutup sebelum dikosongkan")
	}

	if q.running {
		return nil, errors.New("worker harus berhenti sebelum antrean dikosongkan")
	}

	var pending []*job.Job

	for {
		select {
		case task := <-q.tasks:
			pending = append(pending, task)
		default:
			return pending, nil
		}
	}
}
