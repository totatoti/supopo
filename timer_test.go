package supopo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTimer(b *testing.B) {
	t := timer{}
	timeout := 1 * time.Millisecond

	b.Run("usePool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			timer := t.Get(timeout)
			t.Put(timer)
		}
	})

	b.Run("usePoolParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				timer := t.Get(timeout)
				t.Put(timer)
			}
		})
	})

	b.Run("unusePool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			timer := time.NewTimer(timeout)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
	})
}

func Test_timer_Get(t *testing.T) {
	var pool timer
	timer := pool.Get(10 * time.Millisecond)

	// Verify that the timer is not nil
	assert.NotNil(t, timer)
}

func Test_timer_Put_Reuse(t *testing.T) {
	var pool timer
	firstTimer := pool.Get(10 * time.Millisecond)
	pool.Put(firstTimer)

	secondTimer := pool.Get(10 * time.Millisecond)

	// Verify that the first and second timers are the same, indicating reuse.
	assert.Equal(t, firstTimer, secondTimer)
}

func Test_timer_Put_New(t *testing.T) {
	var pool timer
	firstTimer := pool.Get(10 * time.Millisecond)
	pool.Put(firstTimer)

	secondTimer := pool.Get(10 * time.Millisecond)
	thirdTimer := pool.Get(10 * time.Millisecond)

	// Verify that the first and second timers are the same, indicating reuse.
	assert.Equal(t, firstTimer, secondTimer)
	// Verify that a new timer is returned when the pool is empty.
	assert.NotEqual(t, secondTimer, thirdTimer)
}

func Test_timer_Put_ChannelCleared(t *testing.T) {
	var pool timer
	shortTimer := time.NewTimer(1 * time.Nanosecond)
	time.Sleep(10 * time.Nanosecond)

	pool.Put(shortTimer)
	select {
	case <-shortTimer.C:
		assert.Fail(t, "Timer channel was not cleared")
	default:
		// Timer channel was cleared
	}
}

func Test_timer_Put_Nil(t *testing.T) {
	// Passing nil to Put does not cause a panic
	var pool timer
	pool.Put(nil)
}

func Test_timer_Put_Other(t *testing.T) {
	// Passing a timer not obtained from Get to Put does not cause a panic
	var pool timer
	pool.Put(time.NewTimer(10 * time.Millisecond))
}
