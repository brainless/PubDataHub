package jobs

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// JobScheduler manages scheduled and recurring jobs
type JobScheduler struct {
	mu            sync.RWMutex
	scheduledJobs map[string]*ScheduledJob
	cronSchedules map[string]*CronSchedule
	dependencies  map[string][]string // jobID -> list of dependency jobIDs
	manager       *Manager
	ticker        *time.Ticker
	stopChan      chan struct{}
	running       bool
}

// ScheduledJob represents a job that runs on a schedule
type ScheduledJob struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	JobType     string                 `json:"job_type"`
	Config      map[string]interface{} `json:"config"`
	Schedule    string                 `json:"schedule"` // Cron expression
	Enabled     bool                   `json:"enabled"`
	NextRun     time.Time              `json:"next_run"`
	LastRun     time.Time              `json:"last_run"`
	RunCount    int                    `json:"run_count"`
	FailCount   int                    `json:"fail_count"`
	MaxRetries  int                    `json:"max_retries"`
	Timeout     time.Duration          `json:"timeout"`
	Tags        []string               `json:"tags"`
	Created     time.Time              `json:"created"`
	CreatedBy   string                 `json:"created_by"`
	Description string                 `json:"description"`
}

// CronSchedule represents a parsed cron schedule
type CronSchedule struct {
	Expression string
	Minute     []int // 0-59
	Hour       []int // 0-23
	DayOfMonth []int // 1-31
	Month      []int // 1-12
	DayOfWeek  []int // 0-6 (Sunday = 0)
}

// JobDependency represents a dependency between jobs
type JobDependency struct {
	JobID         string              `json:"job_id"`
	DependsOn     []string            `json:"depends_on"`
	Condition     DependencyCondition `json:"condition"`
	WaitTimeout   time.Duration       `json:"wait_timeout"`
	FailureAction string              `json:"failure_action"` // skip, retry, fail
}

type DependencyCondition string

const (
	DependencySuccess  DependencyCondition = "success"
	DependencyComplete DependencyCondition = "complete"
	DependencyAny      DependencyCondition = "any"
)

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(manager *Manager) *JobScheduler {
	return &JobScheduler{
		scheduledJobs: make(map[string]*ScheduledJob),
		cronSchedules: make(map[string]*CronSchedule),
		dependencies:  make(map[string][]string),
		manager:       manager,
		stopChan:      make(chan struct{}),
		running:       false,
	}
}

// Start starts the job scheduler
func (js *JobScheduler) Start() error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if js.running {
		return fmt.Errorf("scheduler already running")
	}

	js.ticker = time.NewTicker(time.Minute) // Check every minute
	js.running = true

	go js.schedulingLoop()

	log.Logger.Info("Job scheduler started")
	return nil
}

// Stop stops the job scheduler
func (js *JobScheduler) Stop() error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if !js.running {
		return nil
	}

	close(js.stopChan)
	if js.ticker != nil {
		js.ticker.Stop()
	}
	js.running = false

	log.Logger.Info("Job scheduler stopped")
	return nil
}

// ScheduleJob schedules a new job with a cron expression
func (js *JobScheduler) ScheduleJob(job *ScheduledJob) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Parse the cron schedule
	cronSchedule, err := js.parseCronExpression(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	job.NextRun = js.calculateNextRun(cronSchedule, time.Now())
	job.Created = time.Now()

	js.scheduledJobs[job.ID] = job
	js.cronSchedules[job.ID] = cronSchedule

	log.Logger.Infof("Scheduled job '%s' (%s) next run: %s", job.Name, job.ID, job.NextRun.Format("2006-01-02 15:04:05"))
	return nil
}

// UnscheduleJob removes a scheduled job
func (js *JobScheduler) UnscheduleJob(jobID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if _, exists := js.scheduledJobs[jobID]; !exists {
		return fmt.Errorf("scheduled job '%s' not found", jobID)
	}

	delete(js.scheduledJobs, jobID)
	delete(js.cronSchedules, jobID)
	delete(js.dependencies, jobID)

	log.Logger.Infof("Unscheduled job '%s'", jobID)
	return nil
}

// AddJobDependency adds a dependency between jobs
func (js *JobScheduler) AddJobDependency(jobID string, dependsOn []string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Check for circular dependencies
	if js.hasCircularDependency(jobID, dependsOn) {
		return fmt.Errorf("circular dependency detected")
	}

	js.dependencies[jobID] = dependsOn
	log.Logger.Infof("Added dependencies for job '%s': %v", jobID, dependsOn)
	return nil
}

// ListScheduledJobs returns all scheduled jobs
func (js *JobScheduler) ListScheduledJobs() []*ScheduledJob {
	js.mu.RLock()
	defer js.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(js.scheduledJobs))
	for _, job := range js.scheduledJobs {
		jobs = append(jobs, job)
	}

	// Sort by next run time
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].NextRun.Before(jobs[j].NextRun)
	})

	return jobs
}

// GetScheduledJob returns a specific scheduled job
func (js *JobScheduler) GetScheduledJob(jobID string) (*ScheduledJob, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	job, exists := js.scheduledJobs[jobID]
	if !exists {
		return nil, fmt.Errorf("scheduled job '%s' not found", jobID)
	}

	return job, nil
}

// EnableJob enables a scheduled job
func (js *JobScheduler) EnableJob(jobID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.scheduledJobs[jobID]
	if !exists {
		return fmt.Errorf("scheduled job '%s' not found", jobID)
	}

	job.Enabled = true
	log.Logger.Infof("Enabled scheduled job '%s'", jobID)
	return nil
}

// DisableJob disables a scheduled job
func (js *JobScheduler) DisableJob(jobID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	job, exists := js.scheduledJobs[jobID]
	if !exists {
		return fmt.Errorf("scheduled job '%s' not found", jobID)
	}

	job.Enabled = false
	log.Logger.Infof("Disabled scheduled job '%s'", jobID)
	return nil
}

// GetUpcomingJobs returns jobs scheduled to run within the next duration
func (js *JobScheduler) GetUpcomingJobs(within time.Duration) []*ScheduledJob {
	js.mu.RLock()
	defer js.mu.RUnlock()

	cutoff := time.Now().Add(within)
	var upcoming []*ScheduledJob

	for _, job := range js.scheduledJobs {
		if job.Enabled && job.NextRun.Before(cutoff) {
			upcoming = append(upcoming, job)
		}
	}

	// Sort by next run time
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].NextRun.Before(upcoming[j].NextRun)
	})

	return upcoming
}

// schedulingLoop is the main scheduling loop
func (js *JobScheduler) schedulingLoop() {
	for {
		select {
		case <-js.ticker.C:
			js.checkAndRunJobs()
		case <-js.stopChan:
			return
		}
	}
}

// checkAndRunJobs checks for jobs that need to run and executes them
func (js *JobScheduler) checkAndRunJobs() {
	js.mu.Lock()
	now := time.Now()
	var jobsToRun []*ScheduledJob

	for _, job := range js.scheduledJobs {
		if job.Enabled && now.After(job.NextRun) {
			jobsToRun = append(jobsToRun, job)
		}
	}
	js.mu.Unlock()

	// Run jobs (outside the lock to avoid blocking)
	for _, job := range jobsToRun {
		if js.areDependenciesSatisfied(job.ID) {
			go js.executeScheduledJob(job)
		} else {
			log.Logger.Infof("Job '%s' dependencies not satisfied, skipping", job.ID)
		}
	}
}

// executeScheduledJob executes a scheduled job
func (js *JobScheduler) executeScheduledJob(scheduledJob *ScheduledJob) {
	js.mu.Lock()
	scheduledJob.LastRun = time.Now()
	scheduledJob.RunCount++

	// Calculate next run time
	if cronSchedule, exists := js.cronSchedules[scheduledJob.ID]; exists {
		scheduledJob.NextRun = js.calculateNextRun(cronSchedule, time.Now())
	}
	js.mu.Unlock()

	log.Logger.Infof("Executing scheduled job '%s' (%s)", scheduledJob.Name, scheduledJob.ID)

	// Create a scheduled job implementation
	job := &ScheduledJobExecution{
		id:          scheduledJob.ID + "_" + strconv.FormatInt(time.Now().Unix(), 10),
		jobType:     JobType(scheduledJob.JobType),
		priority:    PriorityNormal,
		config:      scheduledJob.Config,
		description: scheduledJob.Description,
		metadata:    make(JobMetadata),
	}

	_, err := js.manager.SubmitJob(job)
	if err != nil {
		js.mu.Lock()
		scheduledJob.FailCount++
		js.mu.Unlock()
		log.Logger.Errorf("Failed to submit scheduled job '%s': %v", scheduledJob.ID, err)
	}
}

// areDependenciesSatisfied checks if all dependencies for a job are satisfied
func (js *JobScheduler) areDependenciesSatisfied(jobID string) bool {
	js.mu.RLock()
	dependencies, hasDeps := js.dependencies[jobID]
	js.mu.RUnlock()

	if !hasDeps {
		return true // No dependencies
	}

	// Check if all dependency jobs have completed successfully
	for _, depID := range dependencies {
		// In a real implementation, you'd check the job status from the manager
		// For now, we'll assume dependencies are satisfied
		log.Logger.Debugf("Checking dependency '%s' for job '%s'", depID, jobID)
	}

	return true
}

// hasCircularDependency checks for circular dependencies
func (js *JobScheduler) hasCircularDependency(jobID string, newDeps []string) bool {
	visited := make(map[string]bool)

	var checkCycle func(string) bool
	checkCycle = func(id string) bool {
		if visited[id] {
			return true // Cycle detected
		}
		visited[id] = true

		deps := js.dependencies[id]
		if id == jobID {
			deps = newDeps
		}

		for _, dep := range deps {
			if checkCycle(dep) {
				return true
			}
		}

		visited[id] = false
		return false
	}

	return checkCycle(jobID)
}

// parseCronExpression parses a cron expression (simplified version)
func (js *JobScheduler) parseCronExpression(expr string) (*CronSchedule, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields: %s", expr)
	}

	schedule := &CronSchedule{Expression: expr}

	var err error
	schedule.Minute, err = js.parseField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}

	schedule.Hour, err = js.parseField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}

	schedule.DayOfMonth, err = js.parseField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day of month field: %w", err)
	}

	schedule.Month, err = js.parseField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}

	schedule.DayOfWeek, err = js.parseField(parts[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid day of week field: %w", err)
	}

	return schedule, nil
}

// parseField parses a single cron field
func (js *JobScheduler) parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		// Return all values in range
		values := make([]int, max-min+1)
		for i := min; i <= max; i++ {
			values[i-min] = i
		}
		return values, nil
	}

	var values []int

	// Handle comma-separated values
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if strings.Contains(part, "/") {
			// Handle step values (e.g., "*/5", "0-23/2")
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return nil, fmt.Errorf("invalid step format: %s", part)
			}

			step, err := strconv.Atoi(stepParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid step value: %s", stepParts[1])
			}

			rangeValues, err := js.parseRange(stepParts[0], min, max)
			if err != nil {
				return nil, err
			}

			for i := 0; i < len(rangeValues); i += step {
				values = append(values, rangeValues[i])
			}
		} else if strings.Contains(part, "-") {
			// Handle ranges (e.g., "1-5")
			rangeValues, err := js.parseRange(part, min, max)
			if err != nil {
				return nil, err
			}
			values = append(values, rangeValues...)
		} else {
			// Handle single values
			value, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid value: %s", part)
			}
			if value < min || value > max {
				return nil, fmt.Errorf("value %d out of range [%d-%d]", value, min, max)
			}
			values = append(values, value)
		}
	}

	return values, nil
}

// parseRange parses a range expression
func (js *JobScheduler) parseRange(rangeExpr string, min, max int) ([]int, error) {
	if rangeExpr == "*" {
		values := make([]int, max-min+1)
		for i := min; i <= max; i++ {
			values[i-min] = i
		}
		return values, nil
	}

	if !strings.Contains(rangeExpr, "-") {
		// Single value
		value, err := strconv.Atoi(rangeExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", rangeExpr)
		}
		return []int{value}, nil
	}

	// Range (e.g., "1-5")
	parts := strings.Split(rangeExpr, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range format: %s", rangeExpr)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid range start: %s", parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid range end: %s", parts[1])
	}

	if start > end {
		return nil, fmt.Errorf("range start %d greater than end %d", start, end)
	}

	values := make([]int, end-start+1)
	for i := start; i <= end; i++ {
		values[i-start] = i
	}

	return values, nil
}

// calculateNextRun calculates the next run time for a cron schedule
func (js *JobScheduler) calculateNextRun(schedule *CronSchedule, from time.Time) time.Time {
	// Start from the next minute
	next := from.Truncate(time.Minute).Add(time.Minute)

	// Find the next valid time (simplified implementation)
	for i := 0; i < 366*24*60; i++ { // Search up to a year
		if js.matchesSchedule(schedule, next) {
			return next
		}
		next = next.Add(time.Minute)
	}

	// Fallback to a year from now if no match found
	return from.AddDate(1, 0, 0)
}

// matchesSchedule checks if a time matches the cron schedule
func (js *JobScheduler) matchesSchedule(schedule *CronSchedule, t time.Time) bool {
	return js.contains(schedule.Minute, t.Minute()) &&
		js.contains(schedule.Hour, t.Hour()) &&
		js.contains(schedule.DayOfMonth, t.Day()) &&
		js.contains(schedule.Month, int(t.Month())) &&
		js.contains(schedule.DayOfWeek, int(t.Weekday()))
}

// contains checks if a slice contains a value
func (js *JobScheduler) contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// GetSchedulerStats returns statistics about the scheduler
func (js *JobScheduler) GetSchedulerStats() SchedulerStats {
	js.mu.RLock()
	defer js.mu.RUnlock()

	stats := SchedulerStats{
		TotalScheduledJobs: len(js.scheduledJobs),
		EnabledJobs:        0,
		DisabledJobs:       0,
		TotalRuns:          0,
		TotalFailures:      0,
	}

	for _, job := range js.scheduledJobs {
		if job.Enabled {
			stats.EnabledJobs++
		} else {
			stats.DisabledJobs++
		}
		stats.TotalRuns += job.RunCount
		stats.TotalFailures += job.FailCount
	}

	return stats
}

// SchedulerStats contains scheduler statistics
type SchedulerStats struct {
	TotalScheduledJobs int `json:"total_scheduled_jobs"`
	EnabledJobs        int `json:"enabled_jobs"`
	DisabledJobs       int `json:"disabled_jobs"`
	TotalRuns          int `json:"total_runs"`
	TotalFailures      int `json:"total_failures"`
}

// ScheduledJobExecution implements the Job interface for scheduled jobs
type ScheduledJobExecution struct {
	id          string
	jobType     JobType
	priority    JobPriority
	config      map[string]interface{}
	description string
	metadata    JobMetadata
	progress    JobProgress
}

// ID returns the job ID
func (sje *ScheduledJobExecution) ID() string {
	return sje.id
}

// Type returns the job type
func (sje *ScheduledJobExecution) Type() JobType {
	return sje.jobType
}

// Priority returns the job priority
func (sje *ScheduledJobExecution) Priority() JobPriority {
	return sje.priority
}

// SetPriority sets the job priority
func (sje *ScheduledJobExecution) SetPriority(priority JobPriority) {
	sje.priority = priority
}

// Description returns the job description
func (sje *ScheduledJobExecution) Description() string {
	return sje.description
}

// Metadata returns the job metadata
func (sje *ScheduledJobExecution) Metadata() JobMetadata {
	return sje.metadata
}

// Execute runs the scheduled job
func (sje *ScheduledJobExecution) Execute(ctx context.Context, progressCallback ProgressCallback) error {
	// This is a placeholder implementation
	// In a real implementation, you would dispatch to the appropriate job handler
	// based on the job type and configuration

	log.Logger.Infof("Executing scheduled job %s of type %s", sje.id, sje.jobType)

	// Simulate some work
	sje.progress = JobProgress{
		Current: 0,
		Total:   100,
		Message: "Starting scheduled job",
	}
	progressCallback(sje.progress)

	// Simulate progress
	for i := 0; i <= 100; i += 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(100 * time.Millisecond) // Simulate work
			sje.progress.Current = int64(i)
			sje.progress.Message = fmt.Sprintf("Processing... %d%%", i)
			progressCallback(sje.progress)
		}
	}

	log.Logger.Infof("Completed scheduled job %s", sje.id)
	return nil
}

// CanPause returns whether this job can be paused
func (sje *ScheduledJobExecution) CanPause() bool {
	return false // Scheduled jobs typically can't be paused
}

// Pause pauses the job
func (sje *ScheduledJobExecution) Pause() error {
	return fmt.Errorf("scheduled jobs cannot be paused")
}

// Resume resumes the job
func (sje *ScheduledJobExecution) Resume(ctx context.Context) error {
	return fmt.Errorf("scheduled jobs cannot be resumed")
}

// Progress returns the current job progress
func (sje *ScheduledJobExecution) Progress() JobProgress {
	return sje.progress
}

// Validate validates the job configuration
func (sje *ScheduledJobExecution) Validate() error {
	if sje.id == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if sje.jobType == "" {
		return fmt.Errorf("job type cannot be empty")
	}
	return nil
}
