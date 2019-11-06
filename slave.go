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

package main

import (
	"sync"
	"time"

	"github.com/xuwei0455/slaves/utils"
)

// Usage:
//
//  <SYNC MODEL>:
//  pool := Pool(5)
//  for _, task := range Tasks {
//  	job := pool.Add(task)
//      // access of every property of job will block the main goroutine
//      if !job.Success {
//      	// deal with the result of task
//			fmt.Println(job.Result)
//      }
//  }
//
//  <ASYNC MODEL>:
//  pool := Pool(5)
//  for _, task := range Tasks {
//  	pool.Add(task)
//  }
//  // will wait all goroutine finish.
//  jobs := pool.Join()
//  // jobs it's unordered list of job
//	for _, job := range jobs {
//		if !job.Success {
//			// deal with the result of task
//			fmt.Println(job.Result)
//		}
//	}

// Worker manage the goroutines, it's can block the main
// goroutine until the workers be done.
type Slave struct {
	// a man with work never finished
	Jobs []*Job

	pool *Pool

	stop chan bool

	wg *sync.WaitGroup
}

var (
	DefaultEvilMaxCount        = 512 * 1024
	DefaultEvilMaxIdleDuration = time.Duration(10) * time.Minute
)

func New(optFunc ...OptionFunc) *Slave {
	opt := &Options{
		EvilMaxCount:        DefaultEvilMaxCount,
		EvilMaxIdleDuration: DefaultEvilMaxIdleDuration,
	}

	for _, f := range optFunc {
		f(opt)
	}

	s := &Slave{
		pool: newPool(opt),
		stop: make(chan bool),
		wg:   utils.AcquireWG(),
	}
	return s
}

func (s *Slave) Add(task func()) *Job {
	job := newJob(task, s.wg)
	// add to wait group
	s.wg.Add(1)
	s.Jobs = append(s.Jobs, job)
	// add to pool
	s.pool.do(job)
	return job
}

func (s *Slave) Join() []*Job {
	s.wg.Wait()

	// release
	utils.ReleaseWG(s.wg)
	return s.Jobs
}
