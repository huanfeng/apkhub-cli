package utils

import (
	"fmt"
	"strings"
	"time"
)

// ProgressBar represents a progress bar
type ProgressBar struct {
	total       int64
	current     int64
	description string
	startTime   time.Time
	width       int
	showETA     bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int64, description string) *ProgressBar {
	return &ProgressBar{
		total:       total,
		current:     0,
		description: description,
		startTime:   time.Now(),
		width:       50,
		showETA:     true,
	}
}

// Update updates the progress bar
func (pb *ProgressBar) Update(current int64) {
	pb.current = current
	pb.render()
}

// Increment increments the progress by 1
func (pb *ProgressBar) Increment() {
	pb.current++
	pb.render()
}

// SetDescription updates the description
func (pb *ProgressBar) SetDescription(desc string) {
	pb.description = desc
	pb.render()
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish() {
	pb.current = pb.total
	pb.render()
	fmt.Println() // New line after completion
}

// render renders the progress bar
func (pb *ProgressBar) render() {
	if pb.total <= 0 {
		return
	}

	percentage := float64(pb.current) / float64(pb.total) * 100
	filled := int(float64(pb.width) * float64(pb.current) / float64(pb.total))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.width-filled)
	
	// Calculate ETA
	elapsed := time.Since(pb.startTime)
	var eta string
	if pb.showETA && pb.current > 0 {
		totalTime := time.Duration(float64(elapsed) * float64(pb.total) / float64(pb.current))
		remaining := totalTime - elapsed
		if remaining > 0 {
			eta = fmt.Sprintf(" ETA: %v", remaining.Round(time.Second))
		}
	}

	// Format output
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)%s", 
		pb.description, bar, percentage, pb.current, pb.total, eta)
}

// ScanProgress tracks scanning progress with detailed statistics
type ScanProgress struct {
	TotalFiles    int
	ProcessedFiles int
	NewAPKs       int
	UpdatedAPKs   int
	UnchangedAPKs int
	SkippedFiles  int
	ErrorCount    int
	StartTime     time.Time
	CurrentFile   string
}

// NewScanProgress creates a new scan progress tracker
func NewScanProgress() *ScanProgress {
	return &ScanProgress{
		StartTime: time.Now(),
	}
}

// SetTotalFiles sets the total number of files to process
func (sp *ScanProgress) SetTotalFiles(total int) {
	sp.TotalFiles = total
}

// SetCurrentFile sets the currently processing file
func (sp *ScanProgress) SetCurrentFile(filename string) {
	sp.CurrentFile = filename
	sp.ProcessedFiles++
}

// AddNewAPK increments new APK counter
func (sp *ScanProgress) AddNewAPK() {
	sp.NewAPKs++
}

// AddUpdatedAPK increments updated APK counter
func (sp *ScanProgress) AddUpdatedAPK() {
	sp.UpdatedAPKs++
}

// AddUnchangedAPK increments unchanged APK counter
func (sp *ScanProgress) AddUnchangedAPK() {
	sp.UnchangedAPKs++
}

// AddSkippedFile increments skipped file counter
func (sp *ScanProgress) AddSkippedFile() {
	sp.SkippedFiles++
}

// AddError increments error counter
func (sp *ScanProgress) AddError() {
	sp.ErrorCount++
}

// ShowProgress displays current progress
func (sp *ScanProgress) ShowProgress() {
	if sp.TotalFiles <= 0 {
		return
	}

	percentage := float64(sp.ProcessedFiles) / float64(sp.TotalFiles) * 100
	elapsed := time.Since(sp.StartTime)
	
	// Estimate remaining time
	var eta string
	if sp.ProcessedFiles > 0 {
		avgTimePerFile := elapsed / time.Duration(sp.ProcessedFiles)
		remaining := time.Duration(sp.TotalFiles-sp.ProcessedFiles) * avgTimePerFile
		eta = fmt.Sprintf(" ETA: %v", remaining.Round(time.Second))
	}

	// Show current file being processed
	currentFile := sp.CurrentFile
	if len(currentFile) > 40 {
		currentFile = "..." + currentFile[len(currentFile)-37:]
	}

	fmt.Printf("\r📊 Progress: %.1f%% (%d/%d) | Processing: %s%s", 
		percentage, sp.ProcessedFiles, sp.TotalFiles, currentFile, eta)
}

// ShowFinalStats displays final scanning statistics
func (sp *ScanProgress) ShowFinalStats() {
	elapsed := time.Since(sp.StartTime)
	
	fmt.Println("\n")
	fmt.Println("=== Scan Results ===")
	fmt.Printf("Files processed: %d\n", sp.ProcessedFiles)
	fmt.Printf("New APKs: %d\n", sp.NewAPKs)
	fmt.Printf("Updated APKs: %d\n", sp.UpdatedAPKs)
	fmt.Printf("Unchanged APKs: %d\n", sp.UnchangedAPKs)
	fmt.Printf("Skipped files: %d\n", sp.SkippedFiles)
	
	if sp.ErrorCount > 0 {
		fmt.Printf("Errors: %d\n", sp.ErrorCount)
	}
	
	fmt.Printf("Total time: %v\n", elapsed.Round(time.Millisecond))
	
	if sp.ProcessedFiles > 0 {
		avgTime := elapsed / time.Duration(sp.ProcessedFiles)
		fmt.Printf("Average time per file: %v\n", avgTime.Round(time.Millisecond))
	}
}