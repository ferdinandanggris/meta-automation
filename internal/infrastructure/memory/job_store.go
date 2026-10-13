package memory

import (
	"sync"

	"meta-automation/internal/domain/job"
)

type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]job.Snapshot
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]job.Snapshot),
	}
}

func (s *JobStore) Save(snapshot job.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[snapshot.ID] = snapshot
}

func (s *JobStore) Get(id string) (job.Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, found := s.jobs[id]
	return snapshot, found
}

func (s *JobStore) Insert(snapshot job.Snapshot) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.jobs[snapshot.ID]; found {
		return false
	}

	s.jobs[snapshot.ID] = snapshot
	return true
}
