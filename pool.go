// MIT License

// Copyright (c) 2019 William Hsu

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package slaves

import (
	"runtime"
	"sync"
	"time"
)

// Pool is an singleton object that manage all the workers
// and groups.
// When first invoke slave.New(), a new slave and a new pool will
// create, the other times of invoke slave.New() will just create
// a slave with the pool will no create again, and pool.Join() will
// only block the worker which belongs to the group.
type Pool struct {
	// EvilMaxCount
	EvilMaxCount int

	// rest or die
	EvilMaxIdleDuration time.Duration

	// root of the evil
	evils []*Evil
	// available evil count
	evCount int

	stopCh   chan bool
	mustStop bool

	// lock of the resource `evils` slice and `evCount`
	lock *sync.Mutex
	cond *sync.Cond
}

type Evil struct {
	lastUseTime time.Time
	ch          chan *Job
}

var (
	p    *Pool
	once sync.Once
)

func newPool(opt *Options) *Pool {
	once.Do(func() {
		p = &Pool{
			EvilMaxCount:        opt.EvilMaxCount,
			EvilMaxIdleDuration: opt.EvilMaxIdleDuration,
			lock:                &sync.Mutex{},
		}
		p.cond = sync.NewCond(p.lock)
	})

	go func() {
		var scratch []*Evil
		for {
			p.clean(&scratch)
			select {
			case <-p.stopCh:
				return
			default:
				time.Sleep(p.getMaxIdleWorkerDuration())
			}
		}
	}()
	return p
}

func (p *Pool) run(ev *Evil) {
	for job := range ev.ch {
		if job == nil {
			panic("BUG: no job!")
		}

		job.task()
		job.wg.Done()

		job.finished = true

		if !p.release(ev) {
			break
		}
	}

	p.lock.Lock()
	p.evCount--
	p.lock.Unlock()
}

func (p *Pool) release(ev *Evil) bool {
	ev.lastUseTime = time.Now()

	p.lock.Lock()
	if p.mustStop {
		p.lock.Unlock()
		return false
	}

	p.evils = append(p.evils, ev)
	p.cond.Signal()
	p.lock.Unlock()
	return true
}

func (p *Pool) do(job *Job) bool {
loop:
	// get evil from pool
	ev := p.get()

	if ev == nil {
		p.cond.Wait()
		p.lock.Unlock()
		goto loop
	}

	ev.ch <- job
	return true
}

var jobChanSize = func() int {
	// Use blocking workerChan if GOMAXPROCS=1.
	// This immediately switches Serve to WorkerFunc, which results
	// in higher performance (under go1.5 at least).
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}

	// Use non-blocking workerChan if GOMAXPROCS>1,
	// since otherwise the Serve caller (Acceptor) may lag accepting
	// new connections if WorkerFunc is CPU-bound.
	return 1
}()

func (p *Pool) get() *Evil {
	var ev *Evil
	var create bool

	p.lock.Lock()
	if len(p.evils) < 1 {
		if p.evCount < p.EvilMaxCount {
			create = true
			p.evCount++
		}
	} else {
		ev, p.evils = p.evils[0], p.evils[1:]
	}

	if create {
		ev = &Evil{
			ch: make(chan *Job, jobChanSize),
		}
		go p.run(ev)
	}

	if ev == nil {
		return nil
	}
	p.lock.Unlock()
	return ev
}

func (p *Pool) getMaxIdleWorkerDuration() time.Duration {
	if p.EvilMaxIdleDuration <= 0 {
		return 10 * time.Second
	}
	return p.EvilMaxIdleDuration
}

func (p *Pool) clean(scratch *[]*Evil) {
	maxIdleWorkerDuration := p.getMaxIdleWorkerDuration()

	// Clean least recently used workers if they didn't serve connections
	// for more than maxIdleWorkerDuration.
	criticalTime := time.Now().Add(-maxIdleWorkerDuration)

	p.lock.Lock()
	n := len(p.evils)

	// Use binary-search algorithm to find out the index of the least recently worker which can be cleaned up.
	l, r, mid := 0, n-1, 0
	for l <= r {
		mid = (l + r) / 2
		if criticalTime.After(p.evils[mid].lastUseTime) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	i := r
	if i == -1 {
		p.lock.Unlock()
		return
	}

	*scratch = append((*scratch)[:0], p.evils[:i+1]...)
	m := copy(p.evils, p.evils[i+1:])
	for i = m; i < n; i++ {
		p.evils[i] = nil
	}
	p.evils = p.evils[:m]
	p.lock.Unlock()

	// Notify obsolete workers to stop.
	// This notification must be outside the wp.lock, since ch.ch
	// may be blocking and may consume a lot of time if many workers
	// are located on non-local CPUs.
	tmp := *scratch
	for i := range tmp {
		tmp[i].ch <- nil
		tmp[i] = nil
	}
}
