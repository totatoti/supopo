package supopo

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkPercentile(b *testing.B) {
	p, err := newPercentile()
	if err != nil {
		b.Fatalf("failed to create percentile: %v", err)
	}

	// Record the percentile
	b.Run("record", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := p.recordMicroseconds(1 * time.Millisecond)
			if err != nil {
				b.Fatalf("failed to record percentile: %v", err)
			}
		}
	})

	// Measure the percentile
	b.Run("percentile_1%", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := p.percentileMicroseconds(0.01)
			if err != nil {
				b.Fatalf("failed to get percentile: %v", err)
			}
		}
	})

}

func Test_percentile_percentileMicroseconds(t *testing.T) {
	type args struct {
		percentile float64
	}
	tests := []struct {
		name        string
		percentile  percentileTracker
		args        args
		want        time.Duration
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Test for invalid percentile value below 0",
			percentile: func() percentileTracker {
				p, _ := newPercentile()
				p.recordMicroseconds(100 * time.Microsecond)
				return p
			}(),
			args:        args{percentile: -0.001},
			want:        0,
			wantErr:     true,
			wantErrType: fmt.Errorf("percentile value must be between 0.0 and 1.0, received: %f", -0.001),
		},
		{
			name: "Test for valid percentile value at lower limit",
			percentile: func() percentileTracker {
				p, _ := newPercentile()
				p.recordMicroseconds(100 * time.Microsecond)
				return p
			}(),
			args:    args{percentile: 0.001},
			want:    100 * time.Microsecond,
			wantErr: false,
		},
		{
			name: "Test for invalid percentile value above 1",
			percentile: func() percentileTracker {
				p, _ := newPercentile()
				p.recordMicroseconds(100 * time.Microsecond)
				return p
			}(),
			args:        args{percentile: 1.001},
			want:        0,
			wantErr:     true,
			wantErrType: fmt.Errorf("percentile value must be between 0.0 and 1.0, received: %f", 1.001),
		},
		{
			name: "Test for valid percentile value at upper limit",
			percentile: func() percentileTracker {
				p, _ := newPercentile()
				p.recordMicroseconds(100 * time.Microsecond)
				return p
			}(),
			args:    args{percentile: 1.0},
			want:    100 * time.Microsecond,
			wantErr: false,
		},
		{
			name: "Test for no records",
			percentile: func() percentileTracker {
				p, _ := newPercentile()
				return p
			}(),
			args:        args{percentile: 0.5},
			want:        0,
			wantErr:     true,
			wantErrType: fmt.Errorf("failed to get value at percentile 0.500000: no such element exists"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.percentile.percentileMicroseconds(tt.args.percentile)
			if (err != nil) != tt.wantErr {
				t.Errorf("percentileMicroseconds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.wantErrType.Error() {
				t.Errorf("percentileMicroseconds() error = %v, wantErrType %v", err, tt.wantErrType)
			}
			if got != tt.want {
				t.Errorf("percentileMicroseconds() got = %v, want %v", got, tt.want)
			}
		})
	}
}
