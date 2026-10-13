package onboarding

import "meta-automation/internal/domain/job"

type JobStore interface {
	Insert(snapshot job.Snapshot) bool
	Save(snapshot job.Snapshot)
	Get(id string) (job.Snapshot, bool)
}
