package supopo

import (
	"fmt"
	"time"

	"github.com/DataDog/sketches-go/ddsketch"
)

// percentile represents a tracker for calculating percentiles of time durations.
type percentile struct {
	read *ddsketch.DDSketch
}

// newPercentile returns a new percentileTracker that implements the percentileTracker interface.
func newPercentile() (LatencyTracker, error) {

	/*
		> If the values are time in seconds, maxNumBins = 2048 covers a time range from 80 microseconds to 1 year.
		https://github.com/DataDog/sketches-go
	*/
	r, err := ddsketch.LogCollapsingLowestDenseDDSketch(0.01, 2048)
	if err != nil {
		return nil, err

	}

	return &percentile{
		read: r,
	}, nil
}

// recordMicroseconds records a time duration in microseconds.
func (p *percentile) recordMicroseconds(v time.Duration) error {
	return p.read.Add(float64(v.Microseconds()))
}

// percentileMicroseconds returns the time duration at the specified percentile.
func (p *percentile) percentileMicroseconds(percentile float64) (time.Duration, error) {
	if percentile < 0.0 || percentile > 1.0 {
		return 0, fmt.Errorf("percentile value must be between 0.0 and 1.0, received: %f", percentile)
	}

	d, err := p.read.GetValueAtQuantile(percentile)
	if err != nil {
		return 0, fmt.Errorf("failed to get value at percentile %f: %v", percentile, err)
	}
	return time.Duration(d) * time.Microsecond, nil
}

