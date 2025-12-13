package device

import (
	"context"
	"runtime"
	"sync"
)

// TaskFunc represents a function executed for a specific device ID.
type TaskFunc[T any] func(ctx context.Context, deviceID string) (T, error)

// Result contains the outcome of a task for a device.
type Result[T any] struct {
	DeviceID string
	Value    T
	Err      error
}

// Manager controls concurrent execution of device-scoped tasks.
type Manager[T any] struct {
	workerLimit int
}

// Option configures a Manager.
type Option[T any] func(*Manager[T])

// WithWorkerLimit sets the maximum number of concurrent workers.
func WithWorkerLimit[T any](limit int) Option[T] {
	return func(m *Manager[T]) {
		m.workerLimit = limit
	}
}

// NewManager creates a Manager with optional configuration.
func NewManager[T any](opts ...Option[T]) *Manager[T] {
	m := &Manager[T]{
		workerLimit: runtime.NumCPU(),
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.workerLimit <= 0 {
		m.workerLimit = runtime.NumCPU()
	}

	return m
}

// Run executes the provided task for each device ID concurrently and returns the collected results.
func (m *Manager[T]) Run(ctx context.Context, deviceIDs []string, task TaskFunc[T]) []Result[T] {
	results := make([]Result[T], 0, len(deviceIDs))
	if len(deviceIDs) == 0 {
		return results
	}

	workerCount := m.workerLimit
	if workerCount > len(deviceIDs) {
		workerCount = len(deviceIDs)
	}

	idCh := make(chan string)
	resCh := make(chan Result[T], len(deviceIDs))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for deviceID := range idCh {
				value, err := task(ctx, deviceID)
				resCh <- Result[T]{
					DeviceID: deviceID,
					Value:    value,
					Err:      err,
				}
			}
		}()
	}

	go func() {
		for _, deviceID := range deviceIDs {
			select {
			case <-ctx.Done():
				close(idCh)
				wg.Wait()
				close(resCh)
				return
			case idCh <- deviceID:
			}
		}
		close(idCh)
		wg.Wait()
		close(resCh)
	}()

	for res := range resCh {
		results = append(results, res)
	}

	return results
}
