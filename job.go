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

import "sync"

type Job struct {
	// identify for job(uuid?) for debug
	id string

	task func()

	// flag that specify that finished or not
	finished bool

	success bool
	result  interface{}

	wg *sync.WaitGroup
}

func newJob(task func(), wg *sync.WaitGroup) *Job {
	return &Job{task: task, wg: wg}
}

func (j *Job) Success() bool {
	j.wg.Wait()
	return j.success
}

func (j *Job) Result() interface{} {
	j.wg.Wait()
	return j.result
}
