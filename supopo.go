//go:generate mockgen -source=$GOFILE -destination=./mock/$GOFILE
package supopo

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// percentileTracker defines an interface for recording and retrieving latency percentiles.
type percentileTracker interface {
	// recordMicroseconds records a time duration in microseconds.
	// It returns an error if the recording fails.
	recordMicroseconds(v time.Duration) error

	// percentileMicroseconds returns the time duration at the specified percentile.
	// It returns an error if the retrieval fails.
	percentileMicroseconds(percentile float64) (time.Duration, error)
}

// Option is a type used to implement the Functional Options Pattern.
// It represents an option that can be passed to the NewSupopo function.
type Option func(*config)

// WithTracer configures OpenTelemetry tracing for Supopo.
//
// Parameters:
// - tracer: An OpenTelemetry trace.Tracer instance.
//
// Returns:
// - An Option function that sets the internal tracer.
//
// Usage:
// Pass this to NewSupopo to enable tracing.
func WithTracer(tracer trace.Tracer) Option {
	return func(c *config) {
		c.tracer = tracer
	}
}

// WithMinDataPoints sets the minimum number of data points required for Supopo's asynchronous execution.
// If below this minimum, Supopo will only execute the primary request without sending hedge requests.
//
// Parameters:
// - minDataPoints: Minimum number of data points required (uint).
//
// Returns:
// - Option function to set the internal minimum data points threshold in a Supopo instance.
//
// Usage:
// Configure a new Supopo instance to ensure sufficient data for meaningful asynchronous execution.
func WithMinDataPoints(minDataPoints uint) Option {
	return func(c *config) {
		c.minDataPoints = minDataPoints
	}
}

// config holds the configuration options for a Supopo instance.
// It includes settings for latency tracking, tracing, and the minimum data points required for asynchronous execution.
type config struct {
	percentileTracker percentileTracker
	tracer            trace.Tracer
	minDataPoints     uint
}

// Supopo is a generic type that provides latency-aware execution of functions.
// It supports executing functions synchronously or asynchronously with hedging based on latency percentiles.
type Supopo[T any] struct {
	config
	percentile  float64
	timeout     time.Duration
	maxAttempts uint
	count       uint
	timer       timer
}

// NewSupopo creates a new instance of Supopo with the given configuration options.
//
// Parameters:
// - percentile: The desired percentile for latency tracking (float64).
// - timeout: The maximum duration for each request (time.Duration).
// - maxAttempts: The maximum number of hedge requests to send (uint).
// - options: Optional configuration options for Supopo (Option...).
//
// Returns:
// - A pointer to a new Supopo instance and an error, if any.
//
// Usage:
// Call this function to create a new Supopo instance with the desired configuration.
func NewSupopo[T any](
	percentile float64,
	timeout time.Duration,
	maxAttempts uint,
	options ...Option) (*Supopo[T], error) {

	p, err := newPercentile()
	if err != nil {
		return nil, err
	}

	s := &Supopo[T]{
		config: config{
			percentileTracker: p,
			tracer:            nil,
			minDataPoints:     100,
		},
		percentile:  percentile,
		timeout:     timeout,
		maxAttempts: maxAttempts,
		timer:       timer{},
	}

	for _, option := range options {
		option(&s.config)
	}

	return s, nil
}

// record logs the duration to the internal percentile tracker of the Supopo instance.
// It also increments the internal count of executed functions.
//
// Parameter:
//   - v: A value of type time.Duration. The execution time to be recorded.
//
// Note:
// The record method records the given duration in microseconds to the internal percentile tracker
// and increments the count of executed functions. This allows the Supopo instance to track
// the distribution of function execution times for later analysis.
func (s *Supopo[T]) record(v time.Duration) {
	s.percentileTracker.recordMicroseconds(v)
	s.count++
}

// executeFn executes a provided function within a context and records its execution time.
// This method is responsible for executing a given function synchronously, measuring its latency,
// and incrementing the internal count of executed functions. If tracing is enabled, it also starts
// and ends a tracing span with the provided span name.
//
// Parameters:
//   - ctx: The context within which the function should be executed. This allows for deadline, cancellation
//     signals, and other request-scoped values to be passed through the application's layers.
//   - spanName: A string representing the name of the tracing span. This is used for observability and
//     should be descriptive of the operation being performed.
//   - fn: The function to be executed. It must accept a context and return a pointer to a result of type T
//     and an error. The function represents the operation whose latency is being measured.
//
// Returns:
//   - A pointer to a result of the generic type T and an error. These are the return values of the executed
//     function. The function's result is returned directly to the caller.
//
// Note:
// This method directly contributes to the latency tracking of the Supopo instance by recording the
// execution time of the provided function. It also increments the internal count of executed functions,
// which is used to determine when to switch from synchronous to asynchronous execution mode.
func (s *Supopo[T]) executeFn(ctx context.Context, spanName string, fn func(context.Context) (*T, error)) (*T, error) {
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, spanName)
		defer span.End()
	}

	t := time.Now()
	result, err := fn(ctx)
	s.record(time.Since(t))

	return result, err
}

// executeAsyncFn executes a provided function asynchronously within a context and sends the result or error to the respective channels.
//
// Parameters:
//   - ctx: The context within which the function should be executed. This allows for deadline, cancellation
//     signals, and other request-scoped values to be passed through the application's layers.
//   - spanName: A string representing the name of the tracing span. This is used for observability and
//     should be descriptive of the operation being performed.
//   - fn: The function to be executed. It must accept a context and return a pointer to a result of type T
//     and an error. The function represents the operation whose latency is being measured.
//   - results: A channel to send the result of the executed function.
//   - errs: A channel to send any error that occurred during the execution of the function.
func (s *Supopo[T]) executeAsyncFn(ctx context.Context, spanName string, fn func(context.Context) (*T, error), results chan *T, errs chan error) {
	result, err := s.executeFn(ctx, spanName, fn)

	if err != nil {
		errs <- err
		return
	}
	results <- result
}

// Run executes the provided function with Supopo's latency-aware execution strategy.
// It uses the "hedged request" approach to reduce network latency by sending multiple requests simultaneously
// and returning the result of the first successful request.
//
// Parameters:
//   - ctx: The context within which the function should be executed. This allows for deadline, cancellation
//     signals, and other request-scoped values to be passed through the application's layers.
//   - fn: The function to be executed. It must accept a context and return a pointer to a result of type T
//     and an error. The function represents the operation whose latency is being measured.
//
// Returns:
//   - A pointer to a result of the generic type T and an error. These are the return values of the executed
//     function. The function's result is returned directly to the caller.
//
// Note:
// The Run method determines whether to execute the function synchronously or asynchronously based on the
// number of executed functions and the configured minimum data points threshold. If the count is below the
// threshold, the function is executed synchronously. If the count is above the threshold, the function is
// executed asynchronously with hedging.
func (s *Supopo[T]) Run(ctx context.Context, fn func(context.Context) (*T, error)) (*T, error) {
	// If the count is less than or equal to minDataPoints, executeFn is called to execute the function and return the result.
	// If the count is greater than minDataPoints, executeFn is called asynchronously using a goroutine.
	if s.count < s.minDataPoints {
		return s.executeFn(ctx, "primary-request", fn)
	}

	percentile, err := s.percentileTracker.percentileMicroseconds(s.percentile)
	if err != nil {
		return nil, err
	}
	if percentile < 1 {
		// Sets the percentile to 1 microsecond if the calculated latency is extremely low,
		// assuming that the aggregated latency is very fast and thus setting a floor of 1μs.
		percentile = 1 * time.Microsecond
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	results := make(chan *T, s.maxAttempts)
	errs := make(chan error, s.maxAttempts)

	sentRequests := uint(0)

	go s.executeAsyncFn(timeoutCtx, "primary-request", fn, results, errs)
	sentRequests++

	timer := s.timer.Get(percentile)
	defer s.timer.Put(timer)

	for {
		select {
		case res := <-results:
			return res, nil
		case err := <-errs:
			return nil, err
		case <-timer.C:
			if sentRequests < s.maxAttempts {
				go s.executeAsyncFn(timeoutCtx, "hedge-request", fn, results, errs)
				sentRequests++
				timer.Reset(percentile)
			}
		case <-timeoutCtx.Done():
			return nil, errors.New("all requests timed out")
		}
	}
}
