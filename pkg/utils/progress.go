package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressBar represents a progress bar
type ProgressBar struct {
	total       int64
	current     int64
	width       int
	prefix      string
	suffix      string
	output      io.Writer
	mutex       sync.Mutex
	startTime   time.Time
	lastUpdate  time.Time
	finished    bool
	showPercent bool
	showSpeed   bool
	showETA     bool
	showBar     bool
	template    string
}

// ProgressConfig contains progress bar configuration
type ProgressConfig struct {
	Total       int64
	Width       int
	Prefix      string
	Suffix      string
	Output      io.Writer
	ShowPercent bool
	ShowSpeed   bool
	ShowETA     bool
	ShowBar     bool
	Template    string
}

// DefaultProgressConfig returns a default progress configuration
func DefaultProgressConfig() *ProgressConfig {
	return &ProgressConfig{
		Width:       50,
		Output:      os.Stdout,
		ShowPercent: true,
		ShowSpeed:   true,
		ShowETA:     true,
		ShowBar:     true,
		Template:    "{prefix} {bar} {percent} {speed} {eta} {suffix}",
	}
}

// NewProgressBar creates a new progress bar
func NewProgressBar(config *ProgressConfig) *ProgressBar {
	if config == nil {
		config = DefaultProgressConfig()
	}
	
	return &ProgressBar{
		total:       config.Total,
		width:       config.Width,
		prefix:      config.Prefix,
		suffix:      config.Suffix,
		output:      config.Output,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		showPercent: config.ShowPercent,
		showSpeed:   config.ShowSpeed,
		showETA:     config.ShowETA,
		showBar:     config.ShowBar,
		template:    config.Template,
	}
}

// Set sets the current progress value
func (p *ProgressBar) Set(current int64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.current = current
	if current >= p.total {
		p.finished = true
	}
	
	p.render()
}

// Add adds to the current progress value
func (p *ProgressBar) Add(delta int64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.current += delta
	if p.current >= p.total {
		p.finished = true
	}
	
	p.render()
}

// Increment increments the progress by 1
func (p *ProgressBar) Increment() {
	p.Add(1)
}

// SetTotal sets the total value
func (p *ProgressBar) SetTotal(total int64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.total = total
	p.render()
}

// SetPrefix sets the prefix text
func (p *ProgressBar) SetPrefix(prefix string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.prefix = prefix
	p.render()
}

// SetSuffix sets the suffix text
func (p *ProgressBar) SetSuffix(suffix string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.suffix = suffix
	p.render()
}

// Finish marks the progress as finished
func (p *ProgressBar) Finish() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.current = p.total
	p.finished = true
	p.render()
	fmt.Fprintln(p.output) // Add newline after completion
}

// render renders the progress bar
func (p *ProgressBar) render() {
	// Throttle updates to avoid excessive rendering
	now := time.Now()
	if !p.finished && now.Sub(p.lastUpdate) < 100*time.Millisecond {
		return
	}
	p.lastUpdate = now
	
	// Calculate percentage
	var percent float64
	if p.total > 0 {
		percent = float64(p.current) / float64(p.total) * 100
	}
	
	// Build progress bar
	var bar string
	if p.showBar {
		bar = p.buildBar(percent)
	}
	
	// Build components
	components := map[string]string{
		"{prefix}":  p.prefix,
		"{suffix}":  p.suffix,
		"{bar}":     bar,
		"{percent}": p.buildPercent(percent),
		"{speed}":   p.buildSpeed(),
		"{eta}":     p.buildETA(percent),
		"{current}": fmt.Sprintf("%d", p.current),
		"{total}":   fmt.Sprintf("%d", p.total),
	}
	
	// Apply template
	output := p.template
	for placeholder, value := range components {
		output = strings.ReplaceAll(output, placeholder, value)
	}
	
	// Clean up extra spaces
	output = strings.Join(strings.Fields(output), " ")
	
	// Output with carriage return for overwriting
	fmt.Fprintf(p.output, "\r%s", output)
}

// buildBar builds the progress bar visual
func (p *ProgressBar) buildBar(percent float64) string {
	if p.width <= 0 {
		return ""
	}
	
	filled := int(percent / 100 * float64(p.width))
	if filled > p.width {
		filled = p.width
	}
	
	bar := strings.Repeat("█", filled)
	empty := strings.Repeat("░", p.width-filled)
	
	return fmt.Sprintf("[%s%s]", bar, empty)
}

// buildPercent builds the percentage display
func (p *ProgressBar) buildPercent(percent float64) string {
	if !p.showPercent {
		return ""
	}
	return fmt.Sprintf("%.1f%%", percent)
}

// buildSpeed builds the speed display
func (p *ProgressBar) buildSpeed() string {
	if !p.showSpeed {
		return ""
	}
	
	elapsed := time.Since(p.startTime).Seconds()
	if elapsed <= 0 {
		return ""
	}
	
	speed := float64(p.current) / elapsed
	return fmt.Sprintf("%.1f/s", speed)
}

// buildETA builds the estimated time of arrival display
func (p *ProgressBar) buildETA(percent float64) string {
	if !p.showETA || percent <= 0 {
		return ""
	}
	
	elapsed := time.Since(p.startTime)
	if elapsed.Seconds() <= 0 {
		return ""
	}
	
	remaining := time.Duration(float64(elapsed) * (100 - percent) / percent)
	
	if remaining < time.Minute {
		return fmt.Sprintf("ETA: %ds", int(remaining.Seconds()))
	} else if remaining < time.Hour {
		return fmt.Sprintf("ETA: %dm%ds", int(remaining.Minutes()), int(remaining.Seconds())%60)
	} else {
		return fmt.Sprintf("ETA: %dh%dm", int(remaining.Hours()), int(remaining.Minutes())%60)
	}
}

// Spinner represents a spinning progress indicator
type Spinner struct {
	chars    []string
	current  int
	prefix   string
	suffix   string
	output   io.Writer
	mutex    sync.Mutex
	running  bool
	stopChan chan bool
}

// SpinnerConfig contains spinner configuration
type SpinnerConfig struct {
	Chars  []string
	Prefix string
	Suffix string
	Output io.Writer
	Delay  time.Duration
}

// DefaultSpinnerConfig returns a default spinner configuration
func DefaultSpinnerConfig() *SpinnerConfig {
	return &SpinnerConfig{
		Chars:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Output: os.Stdout,
		Delay:  100 * time.Millisecond,
	}
}

// NewSpinner creates a new spinner
func NewSpinner(config *SpinnerConfig) *Spinner {
	if config == nil {
		config = DefaultSpinnerConfig()
	}
	
	return &Spinner{
		chars:    config.Chars,
		prefix:   config.Prefix,
		suffix:   config.Suffix,
		output:   config.Output,
		stopChan: make(chan bool),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.running {
		return
	}
	
	s.running = true
	go s.spin()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.running {
		return
	}
	
	s.running = false
	s.stopChan <- true
	
	// Clear the spinner line
	fmt.Fprintf(s.output, "\r%s\r", strings.Repeat(" ", len(s.prefix)+len(s.suffix)+10))
}

// SetPrefix sets the prefix text
func (s *Spinner) SetPrefix(prefix string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.prefix = prefix
}

// SetSuffix sets the suffix text
func (s *Spinner) SetSuffix(suffix string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.suffix = suffix
}

// spin runs the spinning animation
func (s *Spinner) spin() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mutex.Lock()
			if !s.running {
				s.mutex.Unlock()
				return
			}
			
			char := s.chars[s.current%len(s.chars)]
			fmt.Fprintf(s.output, "\r%s %s %s", s.prefix, char, s.suffix)
			s.current++
			s.mutex.Unlock()
		}
	}
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	bars   []*ProgressBar
	output io.Writer
	mutex  sync.Mutex
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress(output io.Writer) *MultiProgress {
	if output == nil {
		output = os.Stdout
	}
	
	return &MultiProgress{
		output: output,
	}
}

// AddBar adds a progress bar to the multi-progress
func (mp *MultiProgress) AddBar(config *ProgressConfig) *ProgressBar {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	if config == nil {
		config = DefaultProgressConfig()
	}
	config.Output = mp.output
	
	bar := NewProgressBar(config)
	mp.bars = append(mp.bars, bar)
	
	return bar
}

// Render renders all progress bars
func (mp *MultiProgress) Render() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	// Move cursor up to overwrite previous bars
	if len(mp.bars) > 1 {
		fmt.Fprintf(mp.output, "\033[%dA", len(mp.bars)-1)
	}
	
	// Render each bar
	for i, bar := range mp.bars {
		bar.render()
		if i < len(mp.bars)-1 {
			fmt.Fprintln(mp.output)
		}
	}
}

// ProgressTracker provides a simple interface for tracking progress
type ProgressTracker struct {
	name     string
	total    int64
	current  int64
	bar      *ProgressBar
	spinner  *Spinner
	useBar   bool
	logger   Logger
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(name string, total int64, useBar bool) *ProgressTracker {
	tracker := &ProgressTracker{
		name:    name,
		total:   total,
		useBar:  useBar,
		logger:  GetGlobalLogger(),
	}
	
	if useBar && total > 0 {
		config := DefaultProgressConfig()
		config.Total = total
		config.Prefix = name
		tracker.bar = NewProgressBar(config)
	} else {
		config := DefaultSpinnerConfig()
		config.Prefix = name
		tracker.spinner = NewSpinner(config)
		tracker.spinner.Start()
	}
	
	return tracker
}

// Update updates the progress
func (pt *ProgressTracker) Update(current int64, message string) {
	pt.current = current
	
	if pt.useBar && pt.bar != nil {
		if message != "" {
			pt.bar.SetSuffix(message)
		}
		pt.bar.Set(current)
	} else if pt.spinner != nil {
		if message != "" {
			pt.spinner.SetSuffix(message)
		}
	}
	
	// Log progress at intervals
	if pt.total > 0 {
		percent := float64(current) / float64(pt.total) * 100
		if int(percent)%10 == 0 { // Log every 10%
			pt.logger.Debug("Progress update: %s %.1f%% (%d/%d)", pt.name, percent, current, pt.total)
		}
	}
}

// Finish finishes the progress tracking
func (pt *ProgressTracker) Finish(message string) {
	if pt.useBar && pt.bar != nil {
		if message != "" {
			pt.bar.SetSuffix(message)
		}
		pt.bar.Finish()
	} else if pt.spinner != nil {
		pt.spinner.Stop()
		if message != "" {
			fmt.Printf("%s %s\n", pt.name, message)
		}
	}
	
	pt.logger.Info("Completed: %s", pt.name)
}

// Error stops progress tracking with an error
func (pt *ProgressTracker) Error(err error) {
	if pt.useBar && pt.bar != nil {
		pt.bar.SetSuffix(fmt.Sprintf("Error: %v", err))
		pt.bar.Finish()
	} else if pt.spinner != nil {
		pt.spinner.Stop()
		fmt.Printf("%s Error: %v\n", pt.name, err)
	}
	
	pt.logger.Error("Failed: %s - %v", pt.name, err)
}