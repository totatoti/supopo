package supopo

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func stringPtr(s string) *string {
	return &s
}

type fn struct {
	count        int
	wait         func(count int) time.Duration
	returnString string
	returnError  error
}

func (f *fn) Run(ctx context.Context) (*string, error) {
	f.count++
	c := f.count
	if c > 0 {
		time.Sleep(f.wait(c))
	}
	res := fmt.Sprintf("%s_%d", f.returnString, c)
	return &res, f.returnError
}

func TestSupopo_Run(t *testing.T) {
	type args struct {
		ctx context.Context
		fn  fn
	}
	type testCase struct {
		name          string
		s             *Supopo[string]
		args          args
		want          *string
		wantCallCount int
		wantSpan      func(t *testing.T, exporter *tracetest.InMemoryExporter)
		wantErr       bool
	}

	tests := []testCase{
		{
			name: "normal",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 5 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 1,
			wantErr:       false,
		},

		{
			name: "delay second",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 15 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 2,
			wantErr:       false,
		},

		{
			name: "delay fast",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return time.Duration(30/count) * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_2"),
			wantCallCount: 2,
			wantErr:       false,
		},
		{
			name: "delay many call",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Second, 10)
				for i := 0; i < 100; i++ {
					s.record(1 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 20 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 10,
			wantErr:       false,
		},
		{
			name: "return error",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 5 * time.Millisecond },
					returnString: "",
					returnError:  fmt.Errorf("error"),
				},
			},
			want:          nil,
			wantCallCount: 1,
			wantErr:       true,
		},
		{
			name: "timeout",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 200 * time.Millisecond },
					returnString: "",
					returnError:  fmt.Errorf("error"),
				},
			},
			want:          nil,
			wantCallCount: 2,
			wantErr:       true,
		},

		{
			name: "span check fast",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)

				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 5 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 1,
			wantSpan: func(t *testing.T, exporter *tracetest.InMemoryExporter) {
				assert.Len(t, exporter.GetSpans(), 1)
				assert.Equal(t, "primary-request", exporter.GetSpans()[0].Name)
			},
			wantErr: false,
		},

		{
			name: "span check",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 5 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 1,
			wantSpan: func(t *testing.T, exporter *tracetest.InMemoryExporter) {
				// Wait until all executions are complete
				time.Sleep(5 * 3 * time.Millisecond)

				assert.Len(t, exporter.GetSpans(), 1)
				assert.Equal(t, "primary-request", exporter.GetSpans()[0].Name)
			},
			wantErr: false,
		},
		{
			name: "span check fast no wait",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 15 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 2,
			wantSpan: func(t *testing.T, exporter *tracetest.InMemoryExporter) {
				// If execution is not awaited, only the first executed span will be retrieved.
				assert.Len(t, exporter.GetSpans(), 1)
				assert.Equal(t, "primary-request", exporter.GetSpans()[0].Name)
			},
			wantErr: false,
		},
		{
			name: "span check fast hedge",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 15 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 2,
			wantSpan: func(t *testing.T, exporter *tracetest.InMemoryExporter) {
				// Wait until all executions are complete
				time.Sleep(15 * 3 * time.Millisecond)

				assert.Len(t, exporter.GetSpans(), 2)
				assert.Equal(t, "primary-request", exporter.GetSpans()[0].Name)
				assert.Equal(t, "hedge-request", exporter.GetSpans()[1].Name)
			},
			wantErr: false,
		},
		{
			name: "span check hedge",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)
				for i := 0; i < 100; i++ {
					s.record(10 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return time.Duration(30/count) * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_2"),
			wantCallCount: 2,
			wantSpan: func(t *testing.T, exporter *tracetest.InMemoryExporter) {
				// Wait until all executions are complete
				time.Sleep(30 * 3 * time.Millisecond)

				assert.Len(t, exporter.GetSpans(), 2)
				assert.Equal(t, "hedge-request", exporter.GetSpans()[0].Name)
				assert.Equal(t, "primary-request", exporter.GetSpans()[1].Name)
			},
			wantErr: false,
		},
		{
			name: "span check hedge many call",
			s: func() *Supopo[string] {
				s, _ := NewSupopo[string](0.9, 100*time.Second, 10)
				for i := 0; i < 100; i++ {
					s.record(1 * time.Millisecond)
				}
				return s
			}(),
			args: args{
				ctx: context.Background(),
				fn: fn{
					count:        0,
					wait:         func(count int) time.Duration { return 20 * time.Millisecond },
					returnString: "test",
					returnError:  nil,
				},
			},
			want:          stringPtr("test_1"),
			wantCallCount: 10,
			wantSpan: func(t *testing.T, exporter *tracetest.InMemoryExporter) {
				// Wait until all executions are complete
				time.Sleep(20 * 11 * time.Millisecond)

				assert.Len(t, exporter.GetSpans(), 10)
				assert.Equal(t, "primary-request", exporter.GetSpans()[0].Name)
				for _, span := range exporter.GetSpans()[1:] {
					assert.Equal(t, "hedge-request", span.Name)
				}
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spanChecker := tracetest.NewInMemoryExporter()
			tracerProvider := sdktrace.NewTracerProvider(
				sdktrace.WithSyncer(spanChecker),
			)

			s := tt.s
			s.tracer = tracerProvider.Tracer("test")

			// This span is a prerequisite and unnecessary for the test, so reset it
			spanChecker.Reset()

			got, err := s.Run(tt.args.ctx, tt.args.fn.Run)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.args.fn.count != tt.wantCallCount {
				t.Errorf("Run() call count = %v, want %v", tt.args.fn.count, tt.wantCallCount)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() got = %v, want %v", got, tt.want)
			}

			// If wantSpan is set, perform the check
			if tt.wantSpan != nil {
				tt.wantSpan(t, spanChecker)
			}

		})
	}
}

// Is there any issue with fast processing under 1 microsecond?
func TestSupopo_Run_SubMicrosecondExecution(t *testing.T) {
	supopo, _ := NewSupopo[string](0.9, 100*time.Millisecond, 2)

	for i := 0; i < 100; i++ {
		supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
			time.Sleep(1 * time.Nanosecond)
			return nil, nil
		})
	}

	got, err := supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
		time.Sleep(1 * time.Nanosecond)
		res := "test"
		return &res, nil
	})

	if err != nil {
		t.Errorf("Run() error = %v, wantErr %v", err, false)
		return
	}
	if !reflect.DeepEqual(*got, "test") {
		t.Errorf("Run() got = %v, want %v", *got, "test")
	}
}

// When hedged, does the response return with the minimum processing time?
func TestSupopo_Run_ReturnsResponseInMinimumTimeWhenHedged(t *testing.T) {
	supopo, _ := NewSupopo[string](0.9, 100*time.Second, 2)

	for i := 0; i < 100; i++ {
		supopo.record(1 * time.Microsecond)
	}
	var counter int64
	now := time.Now()
	got, err := supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
		atomic.AddInt64(&counter, 1)
		if counter == 1 {
			time.Sleep(1000 * time.Millisecond)
		} else {
			time.Sleep(10 * time.Millisecond)
		}
		res := fmt.Sprintf("test_%d", counter)

		return &res, nil
	})

	resultTime := time.Since(now)

	// Error if not completed within 20ms
	if resultTime > 20*time.Millisecond {
		t.Errorf("Run() time = %v, want under %v", resultTime, 20*time.Millisecond)
	}
	if err != nil {
		t.Errorf("Run() error = %v, wantErr %v", err, false)
		return
	}

	// Error if not completed on the second call
	if !reflect.DeepEqual(*got, "test_2") {
		t.Errorf("Run() got = %v, want %v", *got, "test_2")
	}

	// Error if not called twice
	if counter != 2 {
		t.Errorf("Run() count = %v, want %v", counter, 2)
	}
}

// If span is set, is it set with a single call?
func TestSupopo_Run_WithTracer(t *testing.T) {

	spanChecker := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(spanChecker),
	)

	supopo, _ := NewSupopo[string](
		0.9,
		100*time.Second,
		2,
		WithTracer(tracerProvider.Tracer("test")),
	)

	supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
		return nil, nil
	})

	// Since the span is set with WithTracer, one span is set
	assert.Len(t, spanChecker.GetSpans(), 1)
	assert.Equal(t, "primary-request", spanChecker.GetSpans()[0].Name)

}

func BenchmarkSupopo(b *testing.B) {
	supopo, _ := NewSupopo[string](0.9, 100*time.Second, 2)
	for i := 0; i < 100; i++ {
		supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
			time.Sleep(5 * time.Millisecond)
			return nil, nil
		})
	}
	b.Run("no_wait", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
				return nil, nil
			})
		}
	})

	supopo, _ = NewSupopo[string](0.9, 100*time.Second, 2)
	for i := 0; i < 100; i++ {
		supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
			time.Sleep(5 * time.Millisecond)
			return nil, nil
		})
	}
	b.Run("normal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
				time.Sleep(1 * time.Millisecond)
				return nil, nil
			})
		}
	})

	supopo, _ = NewSupopo[string](0.9, 100*time.Second, 2)
	for i := 0; i < 100; i++ {
		supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
			time.Sleep(5 * time.Millisecond)
			return nil, nil
		})
	}
	b.Run("delay", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			supopo.Run(context.Background(), func(ctx context.Context) (*string, error) {
				time.Sleep(10 * time.Millisecond)
				return nil, nil
			})
		}
	})

}
