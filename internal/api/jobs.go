package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/google/uuid"
)

// JobInfo represents information about a job for API responses
type JobInfo struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	State        string           `json:"state"`
	Priority     int              `json:"priority"`
	Progress     jobs.JobProgress `json:"progress"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      *time.Time       `json:"end_time,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	RetryCount   int              `json:"retry_count"`
	MaxRetries   int              `json:"max_retries"`
	CreatedBy    string           `json:"created_by"`
	Description  string           `json:"description"`
	Metadata     jobs.JobMetadata `json:"metadata"`
}

// convertJobStatusToJobInfo converts a jobs.JobStatus to a JobInfo
func convertJobStatusToJobInfo(status *jobs.JobStatus) JobInfo {
	return JobInfo{
		ID:           status.ID,
		Type:         string(status.Type),
		State:        string(status.State),
		Priority:     int(status.Priority),
		Progress:     status.Progress,
		StartTime:    status.StartTime,
		EndTime:      status.EndTime,
		ErrorMessage: status.ErrorMessage,
		RetryCount:   status.RetryCount,
		MaxRetries:   status.MaxRetries,
		CreatedBy:    status.CreatedBy,
		Description:  status.Description,
		Metadata:     status.Metadata,
	}
}

// getJobsHandler handles requests to list jobs
func (s *Server) getJobsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual job manager integration
	// For now, return an empty list
	jobsList := []JobInfo{}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(jobsList); err != nil {
		http.Error(w, "Failed to encode jobs", http.StatusInternalServerError)
		return
	}
}

// startDownloadJobHandler handles requests to start a new download job
func (s *Server) startDownloadJobHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req struct {
		Source string `json:"source"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate source
	if req.Source == "" {
		http.Error(w, "Source is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual job creation with job manager
	// For now, create a mock job
	jobID := uuid.New().String()
	jobInfo := JobInfo{
		ID:          jobID,
		Type:        "download",
		State:       "queued",
		Priority:    5,
		Progress:    jobs.JobProgress{Current: 0, Total: 0, Message: "Job queued"},
		StartTime:   time.Now(),
		CreatedBy:   "api",
		Description: fmt.Sprintf("Download job for %s", req.Source),
		Metadata:    jobs.JobMetadata{"source": req.Source},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(jobInfo); err != nil {
		http.Error(w, "Failed to encode job info", http.StatusInternalServerError)
		return
	}
}

// pauseJobHandler handles requests to pause a job
func (s *Server) pauseJobHandler(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 5 || pathParts[4] != "pause" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	jobID := pathParts[3]
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual job pausing with job manager
	// For now, return a success response
	response := map[string]interface{}{
		"message": fmt.Sprintf("Job %s paused successfully", jobID),
		"job_id":  jobID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// resumeJobHandler handles requests to resume a job
func (s *Server) resumeJobHandler(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 5 || pathParts[4] != "resume" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	jobID := pathParts[3]
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual job resuming with job manager
	// For now, return a success response
	response := map[string]interface{}{
		"message": fmt.Sprintf("Job %s resumed successfully", jobID),
		"job_id":  jobID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// registerJobsRoutes registers the jobs-related routes
func (s *Server) registerJobsRoutes() {
	mux := s.httpServer.Handler.(*http.ServeMux)
	mux.HandleFunc("GET /api/jobs", s.getJobsHandler)
	mux.HandleFunc("POST /api/jobs/download", s.startDownloadJobHandler)
	mux.HandleFunc("POST /api/jobs/{job_id}/pause", s.pauseJobHandler)
	mux.HandleFunc("POST /api/jobs/{job_id}/resume", s.resumeJobHandler)
}
