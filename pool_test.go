package slaves

import (
	"testing"
	"time"
)

func TestPool_Run_SYNC(t *testing.T) {
	result := 0

	w := New()
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		w.Add(func() {
			time.Sleep(time.Second * 1)
			result += n
		})
	}
	if result != 0 {
		t.Error(`be zero! man?`)
	}
}

func TestPool_Run_ASYNC(t *testing.T) {
	result := 0

	w := New()
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		n := n
		w.Add(func() {
			time.Sleep(time.Second * 1)
			result += n
		})
	}
	w.Join()

	if result != 55 {
		t.Error(`be 55! man?`)
	}
}
