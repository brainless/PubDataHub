package query

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/brainless/PubDataHub/internal/log"
)

// ExportJobImpl implements the Job interface for query export operations
type ExportJobImpl struct {
	BaseJob

	// Export-specific fields
	dataSource string
	query      string
	format     OutputFormat
	outputFile string
	engine     *TUIQueryEngine

	// Progress tracking
	rowsExported     int64
	totalRows        int64
	bytesWritten     int64
	compressionRatio float64

	// State
	isPaused bool
}

// BaseJob provides common job functionality
type BaseJob struct {
	JobID          string
	JobType        jobs.JobType
	JobPriority    jobs.JobPriority
	JobDescription string
	JobMetadata    jobs.JobMetadata
	JobProgress    jobs.JobProgress
}

// ID returns the job identifier
func (b *BaseJob) ID() string {
	return b.JobID
}

// Type returns the job type
func (b *BaseJob) Type() jobs.JobType {
	return b.JobType
}

// Priority returns the job priority
func (b *BaseJob) Priority() jobs.JobPriority {
	return b.JobPriority
}

// SetPriority sets the job priority
func (b *BaseJob) SetPriority(priority jobs.JobPriority) {
	b.JobPriority = priority
}

// Description returns the job description
func (b *BaseJob) Description() string {
	return b.JobDescription
}

// Metadata returns the job metadata
func (b *BaseJob) Metadata() jobs.JobMetadata {
	return b.JobMetadata
}

// Progress returns the current job progress
func (b *BaseJob) Progress() jobs.JobProgress {
	return b.JobProgress
}

// Execute runs the export job
func (e *ExportJobImpl) Execute(ctx context.Context, progressCallback jobs.ProgressCallback) error {
	log.Logger.Infof("Starting export job: %s", e.ID())

	// Validate the export parameters
	if err := e.Validate(); err != nil {
		return fmt.Errorf("export job validation failed: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := e.ensureOutputDirectory(); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Execute the query to get data
	result, err := e.engine.ExecuteConcurrent(e.dataSource, e.query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	e.totalRows = int64(result.Count)

	// Report initial progress
	e.updateProgress(0, "Starting export", progressCallback)

	// Export the data based on format
	switch e.format {
	case OutputFormatCSV:
		err = e.exportToCSV(ctx, result, progressCallback)
	case OutputFormatJSON:
		err = e.exportToJSON(ctx, result, progressCallback)
	case OutputFormatTSV:
		err = e.exportToTSV(ctx, result, progressCallback)
	default:
		return fmt.Errorf("unsupported export format: %s", e.format)
	}

	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Report completion
	e.updateProgress(e.totalRows, "Export completed", progressCallback)

	log.Logger.Infof("Export job completed: %s (%d rows, %.2f MB)",
		e.ID(), e.rowsExported, float64(e.bytesWritten)/(1024*1024))

	return nil
}

// CanPause returns whether this job can be paused
func (e *ExportJobImpl) CanPause() bool {
	return true
}

// Pause pauses the export job
func (e *ExportJobImpl) Pause() error {
	e.isPaused = true
	log.Logger.Infof("Export job paused: %s", e.ID())
	return nil
}

// Resume resumes the export job
func (e *ExportJobImpl) Resume(ctx context.Context) error {
	e.isPaused = false
	log.Logger.Infof("Export job resumed: %s", e.ID())
	return nil
}

// Validate validates the export job parameters
func (e *ExportJobImpl) Validate() error {
	if e.dataSource == "" {
		return fmt.Errorf("data source is required")
	}

	if e.query == "" {
		return fmt.Errorf("query is required")
	}

	if e.outputFile == "" {
		return fmt.Errorf("output file is required")
	}

	// Validate format
	validFormats := map[OutputFormat]bool{
		OutputFormatCSV:  true,
		OutputFormatJSON: true,
		OutputFormatTSV:  true,
	}

	if !validFormats[e.format] {
		return fmt.Errorf("unsupported format: %s", e.format)
	}

	// Check if data source exists
	if _, exists := e.engine.dataSources[e.dataSource]; !exists {
		return fmt.Errorf("unknown data source: %s", e.dataSource)
	}

	return nil
}

// ensureOutputDirectory creates the output directory if it doesn't exist
func (e *ExportJobImpl) ensureOutputDirectory() error {
	dir := filepath.Dir(e.outputFile)
	if dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// exportToCSV exports query results to CSV format
func (e *ExportJobImpl) exportToCSV(ctx context.Context, result QueryResult, progressCallback jobs.ProgressCallback) error {
	file, err := os.Create(e.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	if err := writer.Write(result.Columns); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write data rows
	for i, row := range result.Rows {
		// Check if paused or cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if e.isPaused {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Convert row to strings
		stringRow := make([]string, len(row))
		for j, cell := range row {
			if cell != nil {
				stringRow[j] = fmt.Sprintf("%v", cell)
			}
		}

		if err := writer.Write(stringRow); err != nil {
			return fmt.Errorf("failed to write CSV row %d: %w", i, err)
		}

		e.rowsExported++

		// Report progress every 1000 rows
		if i%1000 == 0 {
			e.updateProgress(int64(i), fmt.Sprintf("Exported %d rows", i), progressCallback)
		}
	}

	// Get file size
	if stat, err := file.Stat(); err == nil {
		e.bytesWritten = stat.Size()
	}

	return nil
}

// exportToJSON exports query results to JSON format
func (e *ExportJobImpl) exportToJSON(ctx context.Context, result QueryResult, progressCallback jobs.ProgressCallback) error {
	file, err := os.Create(e.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Create structured output
	output := map[string]interface{}{
		"metadata": map[string]interface{}{
			"query":       result.Query,
			"data_source": result.DataSource,
			"timestamp":   result.Timestamp,
			"duration":    result.Duration.String(),
			"row_count":   result.Count,
		},
		"columns": result.Columns,
		"data":    []map[string]interface{}{},
	}

	// Convert rows to structured format
	data := make([]map[string]interface{}, 0, len(result.Rows))
	for i, row := range result.Rows {
		// Check if paused or cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if e.isPaused {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		rowData := make(map[string]interface{})
		for j, cell := range row {
			if j < len(result.Columns) {
				rowData[result.Columns[j]] = cell
			}
		}
		data = append(data, rowData)

		e.rowsExported++

		// Report progress every 1000 rows
		if i%1000 == 0 {
			e.updateProgress(int64(i), fmt.Sprintf("Processed %d rows", i), progressCallback)
		}
	}

	output["data"] = data

	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Get file size
	if stat, err := file.Stat(); err == nil {
		e.bytesWritten = stat.Size()
	}

	return nil
}

// exportToTSV exports query results to TSV format
func (e *ExportJobImpl) exportToTSV(ctx context.Context, result QueryResult, progressCallback jobs.ProgressCallback) error {
	file, err := os.Create(e.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create TSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '\t' // Use tab as separator
	defer writer.Flush()

	// Write headers
	if err := writer.Write(result.Columns); err != nil {
		return fmt.Errorf("failed to write TSV headers: %w", err)
	}

	// Write data rows
	for i, row := range result.Rows {
		// Check if paused or cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if e.isPaused {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Convert row to strings
		stringRow := make([]string, len(row))
		for j, cell := range row {
			if cell != nil {
				stringRow[j] = fmt.Sprintf("%v", cell)
			}
		}

		if err := writer.Write(stringRow); err != nil {
			return fmt.Errorf("failed to write TSV row %d: %w", i, err)
		}

		e.rowsExported++

		// Report progress every 1000 rows
		if i%1000 == 0 {
			e.updateProgress(int64(i), fmt.Sprintf("Exported %d rows", i), progressCallback)
		}
	}

	// Get file size
	if stat, err := file.Stat(); err == nil {
		e.bytesWritten = stat.Size()
	}

	return nil
}

// updateProgress updates the job progress and calls the callback
func (e *ExportJobImpl) updateProgress(current int64, message string, callback jobs.ProgressCallback) {
	e.BaseJob.JobProgress = jobs.JobProgress{
		Current: current,
		Total:   e.totalRows,
		Message: message,
	}

	if callback != nil {
		callback(e.BaseJob.JobProgress)
	}
}
