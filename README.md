# supopo

🚧 work in progress! 🚧

[![CI](https://github.com/totatoti/supopo/actions/workflows/ci.yml/badge.svg)](https://github.com/totatoti/supopo/actions/workflows/ci.yml)


## Overview
Supopo provides a "hedged request" approach to mitigate network latency. This approach aims to optimize user experience by sending additional requests if a request is deemed slower than usual and returning the result of the first successful request.

## Features


### Nanosecond Precision Processing Time Measurement
Supopo measures the processing time of each request with nanosecond precision. This enables highly accurate performance analysis, contributing to system optimization.

### Percentile-Based Retry
The request interval is dynamically adjusted based on the latency percentile of past requests. This allows the system to automatically choose the optimal request timing based on the historical behavior of the network.

### Timeout Support
Each request is subject to a specified timeout.

### Limit on Maximum Number of Requests
By limiting the number of requests sent at once, Supopo helps prevent unintended excessive requests from being sent simultaneously during the hedged request strategy. This ensures the system avoids issues arising from processing an unnecessarily large number of requests at once.

### Tracing Support with OpenTelemetry
Each request is traced using OpenTelemetry. Separate spans are generated for both the primary and hedge requests, allowing for detailed monitoring and analysis of request flow and performance.

* Primary Request: A span is generated for the first sent request, starting the trace.
* Hedge Request: A separate span is also generated for the hedge request sent after the primary request. This allows independent monitoring of each request's performance.

## Usage
Create an instance of Supopo and use the Run method to execute a function. The function takes a context with a timeout and returns either the result or an error.

```go
func sampleFunction(ctx context.Context) (*string, error) {
	randomDelay := time.Duration(rand.Intn(1000)) * time.Millisecond
	select {
	case <-time.After(randomDelay):
		result := "Success after " + randomDelay.String()
		return &result, nil
	case <-ctx.Done():
		return nil, errors.New("request timed out")
	}
}

func main() {
	// Initialize an instance of Supopo
	supopoInstance, err := NewSupopo[string](0.95, 1 * time.Second, 3, 1 * time.Second)
	if err != nil {
		fmt.Println("Error initializing Supopo:", err)
		return
	}

	// Execute the sampleFunction using Supopo
	result, err := supopoInstance.Run(context.Background(), sampleFunction)

	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Result:", *result)
	}
}
```
