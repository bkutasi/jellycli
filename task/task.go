/*
 * Copyright 2019 Tero Vierimaa
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package task

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
)

// Tasker can be run on background
type Tasker interface {
	Start() error
	Stop() error
}

// Common fields for task
type Task struct {
	// Name of the task, for logging purposes
	Name string
	lock sync.RWMutex
	// initialized flag must be true in order to run the task
	initialized bool
	running     bool
	chanStop    chan bool
	loop        func()
}

//IsRunning returns whether task is running or not
func (t *Task) IsRunning() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.running
}

//StopChan returns stop channel that receives value when task stop is called
func (t *Task) StopChan() chan bool {
	return t.chanStop
}

func (t *Task) SetLoop(loop func()) {
	t.loop = loop
	t.initialized = true
}

//Start starts task. If task is already running, or task loop
//is missing, task returns error
func (t *Task) Start() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.running {
		return fmt.Errorf("task '%s' background task already running", t.Name)
	}

	if t.loop == nil {
		return fmt.Errorf("task '%s' has no loop function defined", t.Name)
	}

	if !t.initialized {
		return fmt.Errorf("task '%s' task not initialized properly", t.Name)
	}

	if t.chanStop == nil {
		t.init()
	}

	t.running = true
	go t.run()
	return nil
}

// Stop stops task. If task is not running, return error
func (t *Task) Stop() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if !t.running {
		return fmt.Errorf("task '%s' goroutine not running", t.Name)
	}

	logrus.Tracef("Stopping task: %s", t.Name)
	t.chanStop <- true
	return nil
}

func (t *Task) init() {
	t.chanStop = make(chan bool, 2)
}

func (t *Task) run() {
	t.loop()
	t.lock.Lock()
	t.running = false
	t.lock.Unlock()
	logrus.Tracef("Task %s stopped", t.Name)
}
