package microbatcher // or microbatcher_test if you're using an external package for tests

import (
	"context"
	"testing"
	"time"
)

// A simple BatchProcessor implementation for testing.
type MockBatchProcessor struct{}

func (mbp MockBatchProcessor) ProcessBatch(ctx context.Context, jobs []string) ([]string, error) {
	var results []string
	for _, job := range jobs {
		// For testing, you can directly return the job as the result, or apply some transformation if needed.
		result := job // Or apply any mock logic you want to simulate.
		results = append(results, result)
	}
	return results, nil
}

func TestSingleJobSubmission(t *testing.T) {
	processor := MockBatchProcessor{}
	mb := NewMicroBatcher(processor, 1, 1*time.Second, 1)

	job := "testJob"
	resultChan := mb.SubmitJob(job)

	select {
	case resultWrapper := <-resultChan:
		if resultWrapper.Error != nil {
			t.Fatalf("Expected no error, got %v", resultWrapper.Error)
		}
		if resultWrapper.Result != job {
			t.Fatalf("Expected result %v, got %v", job, resultWrapper.Result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Expected result was not received in time")
	}

	mb.Shutdown()
}

func TestMultipleJobSubmissions(t *testing.T) {
	processor := MockBatchProcessor{}
	mb := NewMicroBatcher(processor, 10, 500*time.Millisecond, 1)

	jobs := []string{"job1", "job2", "job3"}
	resultChans := make([]<-chan JobResultWrapper[string], len(jobs))

	for i, job := range jobs {
		resultChans[i] = mb.SubmitJob(job)
	}

	for i, resultChan := range resultChans {
		select {
		case resultWrapper := <-resultChan:
			if resultWrapper.Error != nil {
				t.Fatalf("Expected no error for job %d, got %v", i, resultWrapper.Error)
			}
			if resultWrapper.Result != jobs[i] {
				t.Fatalf("Expected result %v for job %d, got %v", jobs[i], i, resultWrapper.Result)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Expected result for job %d was not received in time", i)
		}
	}

	mb.Shutdown()
}

func TestBatchProcessing(t *testing.T) {
	processor := MockBatchProcessor{}
	mb := NewMicroBatcher(processor, 2, 500*time.Millisecond, 1)

	job1 := "batchJob1"
	job2 := "batchJob2"
	resultChan1 := mb.SubmitJob(job1)
	resultChan2 := mb.SubmitJob(job2)

	for _, rc := range []<-chan JobResultWrapper[string]{resultChan1, resultChan2} {
		select {
		case resultWrapper := <-rc:
			if resultWrapper.Error != nil {
				t.Fatalf("Expected no error, got %v", resultWrapper.Error)
			}
			// For simplicity, we're not checking specific job results here,
			// assuming our mock processor returns the job as the result.
		case <-time.After(2 * time.Second):
			t.Fatal("Expected result was not received in time")
		}
	}

	mb.Shutdown()
}
