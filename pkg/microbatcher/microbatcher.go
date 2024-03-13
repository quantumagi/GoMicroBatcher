package microbatcher

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Job is a placeholder for any type that represents a job to be processed.
type Job interface{}

// JobResult is a placeholder for any type that represents the result of a job.
type JobResult interface{}

// BatchProcessor defines an interface for processing a batch of jobs.
// It requires a single method, ProcessBatch, that takes a context and a slice of jobs,
// and returns a slice of results or an error if the batch could not be processed.
type BatchProcessor[T Job, R JobResult] interface {
	ProcessBatch(ctx context.Context, jobs []T) ([]R, error)
}

// JobResultWrapper is a struct that wraps the result of a job with its potential error.
// This allows for individual job results to be returned along with an error that might have occurred during processing.
type JobResultWrapper[R JobResult] struct {
	Result R
	Error  error
}

// jobWrapper is an internal struct used to wrap a job along with a channel to send its result back.
// This struct enables asynchronous processing and result reporting for each job.
type jobWrapper[T Job, R JobResult] struct {
	job           T
	resultWrapper chan JobResultWrapper[R]
}

// MicroBatcher is the main struct for batching and processing jobs asynchronously.
// It manages the lifecycle of job processing, including batching, execution, and shutdown.
type MicroBatcher[T Job, R JobResult] struct {
	batchProcessor   BatchProcessor[T, R]  // Processor to handle each batch of jobs.
	jobQueue         chan jobWrapper[T, R] // Channel to queue jobs for processing.
	batchSize        int                   // Maximum number of jobs to process in a single batch.
	batchFrequency   time.Duration         // How often to process batches.
	maxAsyncBatches  int                   // Maximum number of asynchronous batches to process concurrently.
	shutdown         chan struct{}         // Channel to signal shutdown.
	wg               sync.WaitGroup        // WaitGroup to ensure all goroutines complete on shutdown.
	shutdownComplete chan struct{}         // Channel to signal that shutdown process is complete.
}

// NewMicroBatcher creates a new MicroBatcher instance configured with the provided batch processor,
// batch size, batch processing frequency, and the maximum number of asynchronous batches allowed.
// It initializes the internal job queue and starts the goroutine responsible for processing the jobs.
// The function returns a pointer to the created MicroBatcher instance.
func NewMicroBatcher[T Job, R JobResult](processor BatchProcessor[T, R], batchSize int, batchFrequency time.Duration, maxAsyncBatches int) *MicroBatcher[T, R] {
	mb := &MicroBatcher[T, R]{
		batchProcessor:   processor,
		jobQueue:         make(chan jobWrapper[T, R], 100), // Channel size can be adjusted as needed.
		batchSize:        batchSize,
		batchFrequency:   batchFrequency,
		maxAsyncBatches:  maxAsyncBatches,
		shutdown:         make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}
	mb.start() // Begin processing jobs.
	return mb
}

// start begins the batch processing in a separate goroutine.
// It uses a ticker to trigger batch processing at the configured frequency.
func (mb *MicroBatcher[T, R]) start() {
	mb.wg.Add(1)
	go func() {
		defer mb.wg.Done()
		ticker := time.NewTicker(mb.batchFrequency)
		defer ticker.Stop()

		for {
			select {
			case <-mb.shutdown:
				// Handle shutdown: process all remaining jobs then exit.
				for len(mb.jobQueue) > 0 {
					mb.processBatch()
				}
				close(mb.shutdownComplete)
				return
			case <-ticker.C:
				// Process a batch of jobs at each tick.
				mb.processBatch()
			}
		}
	}()
}

// processBatch processes a single batch of jobs.
// It collects jobs from the jobQueue up to the batchSize, then processes them and sends the results back.
func (mb *MicroBatcher[T, R]) processBatch() {
	var batch []jobWrapper[T, R]

	collectJobs := func() bool {
		select {
		case jobWrapper := <-mb.jobQueue:
			batch = append(batch, jobWrapper)
			return true
		default:
			return false
		}
	}

	// Attempt to collect at least one job from the queue.
	if !collectJobs() {
		return // Exit if no jobs are available.
	}

	// Continue collecting jobs until reaching the batchSize or the jobQueue is empty.
	for len(batch) < mb.batchSize && collectJobs() {
	}

	// Prepare the slice of jobs to be processed.
	jobs := make([]T, len(batch))
	for i, wrapper := range batch {
		jobs[i] = wrapper.job
	}

	// Process the batch and handle the results.
	ctx := context.Background() // Future versions could allow for a customizable context.
	results, err := mb.batchProcessor.ProcessBatch(ctx, jobs)

	for i, wrapper := range batch {
		if err != nil {
			wrapper.resultWrapper <- JobResultWrapper[R]{Error: err}
			continue
		}
		if i < len(results) {
			wrapper.resultWrapper <- JobResultWrapper[R]{Result: results[i]}
		} else {
			wrapper.resultWrapper <- JobResultWrapper[R]{Error: errors.New("missing result for job")}
		}
	}
}

// SubmitJob adds a job to the MicroBatcher queue.
func (mb *MicroBatcher[T, R]) SubmitJob(job T) <-chan JobResultWrapper[R] {
	resultWrapperChan := make(chan JobResultWrapper[R], 1)

	select {
	case <-mb.shutdown:
		resultWrapperChan <- JobResultWrapper[R]{Error: errors.New("microbatcher is shutting down")}
	case mb.jobQueue <- jobWrapper[T, R]{job: job, resultWrapper: resultWrapperChan}:
	}

	return resultWrapperChan
}

// Shutdown signals the MicroBatcher to stop processing and wait for in-flight jobs to finish.
func (mb *MicroBatcher[T, R]) Shutdown() {
	close(mb.shutdown)
	mb.wg.Wait() // Wait for all processing to complete
}
