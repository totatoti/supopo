package supopo

import (
	"sync"
	"time"
)

type timer struct {
	pool sync.Pool
}

func (t *timer) Get(d time.Duration) *time.Timer {
	if v := t.pool.Get(); v != nil {
		timer := v.(*time.Timer)
		timer.Reset(d)
		return timer
	}

	return time.NewTimer(d)
}

func (t *timer) Put(timer *time.Timer) {
	if timer == nil {
		return
	}

	// If the timer has already expired or been stopped, the channel needs to be cleared.
	// Refer to: https://pkg.go.dev/time#Timer.Reset
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	t.pool.Put(timer)
}
