# GoMicroBatcher

GoMicroBatcher is a Go package designed for efficient and asynchronous batching and processing of jobs. It allows developers to aggregate multiple jobs into batches for processing at specified intervals or when a batch size is reached, optimizing resource usage and processing time. This package is ideal for applications that need to process large numbers of tasks or jobs with controlled concurrency.

## Features

- **Asynchronous Job Processing**: Process jobs asynchronously for improved performance.
- **Custom Batch Sizes**: Configure the size of batches to suit your application needs.
- **Timed Batching**: Automatically process batches at fixed intervals.
- **Concurrent Batch Processing**: Support for processing multiple batches concurrently, with a configurable limit.
- **Graceful Shutdown**: Safely shutdown the processing, ensuring all in-flight jobs are completed.

## Getting Started

### Prerequisites

Ensure you have Go installed on your system (version 1.13 or later is required for module support).

### Installing

To use GoMicroBatcher in your Go project, start by adding it as a dependency:

```
go get github.com/quantumagi/gomicrobatcher
```


Then, import it in your Go code:

```
import "github.com/quantumagi/gomicrobatcher"
```

### Usage Example
Here's a basic example to demonstrate how to use GoMicroBatcher:

```
package main

import (
    "context"
    "fmt"
    "github.com/quantumagi/gomicrobatcher"
    "time"
)

type MyJob int // Define your job type.
type MyJobResult int // Define your job result type.

// Define a batch processor for your job type.
type MyBatchProcessor struct{}

func (p *MyBatchProcessor) ProcessBatch(ctx context.Context, jobs []MyJob) ([]MyJobResult, error) {
    // Process the batch of jobs. Replace this with your actual batch processing logic.
    results := make([]MyJobResult, len(jobs))
    for i, job := range jobs {
        results[i] = MyJobResult(job * 2) // Example processing logic.
    }
    return results, nil
}

func main() {
    processor := &MyBatchProcessor{} // Initialize your batch processor.
    batcher := microbatcher.NewMicroBatcher(processor, 10, 5*time.Second, 2) // Create a new MicroBatcher instance.

    // Submit a job to the batcher.
    job := MyJob(1)
    resultChan := batcher.SubmitJob(job)

    // Wait for and print the result of the job.
    resultWrapper := <-resultChan
    if resultWrapper.Error != nil {
        fmt.Println("Job processing error:", resultWrapper.Error)
    } else {
        fmt.Println("Job result:", resultWrapper.Result)
    }

    // Shutdown the batcher gracefully.
    batcher.Shutdown()
}
```

## Contributing
Contributions are welcome! Please feel free to submit pull requests, report bugs, and suggest features.

## License
This project is licensed under the MIT License - see the LICENSE file for details.
