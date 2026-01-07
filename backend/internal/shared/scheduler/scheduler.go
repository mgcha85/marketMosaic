package scheduler

import (
	"log"

	"github.com/robfig/cron/v3"
)

// Job represents a schedulable job
type Job struct {
	Name     string
	Schedule string
	Func     func()
}

// UnifiedScheduler manages all batch jobs
type UnifiedScheduler struct {
	cron *cron.Cron
	jobs []Job
}

// New creates a new UnifiedScheduler
func New() *UnifiedScheduler {
	return &UnifiedScheduler{
		cron: cron.New(),
		jobs: make([]Job, 0),
	}
}

// AddJob registers a new job
func (s *UnifiedScheduler) AddJob(name, schedule string, fn func()) error {
	_, err := s.cron.AddFunc(schedule, func() {
		log.Printf("[SCHEDULER] Running job: %s", name)
		fn()
		log.Printf("[SCHEDULER] Completed job: %s", name)
	})
	if err != nil {
		return err
	}

	s.jobs = append(s.jobs, Job{
		Name:     name,
		Schedule: schedule,
		Func:     fn,
	})

	log.Printf("[SCHEDULER] Registered job: %s (%s)", name, schedule)
	return nil
}

// Start begins the scheduler
func (s *UnifiedScheduler) Start() {
	s.cron.Start()
	log.Println("[SCHEDULER] Started with", len(s.jobs), "jobs")
}

// Stop gracefully stops the scheduler
func (s *UnifiedScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("[SCHEDULER] Stopped")
}

// GetJobs returns all registered jobs
func (s *UnifiedScheduler) GetJobs() []Job {
	return s.jobs
}
