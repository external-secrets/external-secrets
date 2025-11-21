// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Scheduler is an interface for scheduling periodic tasks.
type Scheduler interface {
	// ScheduleInterval schedules a function to run every d time.
	ScheduleInterval(key string, interval, timeout time.Duration, fn func(context.Context, logr.Logger))

	// Cancel cancels a scheduled function
	Cancel(key string)

	Start(ctx context.Context) error
}

type job struct {
	stop     context.CancelFunc
	interval time.Duration
}

// Impl implements the Scheduler interface.
type Impl struct {
	log    logr.Logger
	ctx    context.Context
	mu     sync.Mutex
	leader atomic.Bool
	jobs   map[string]job
	client client.Client
}

// New creates a new scheduler.
func New(client client.Client, log logr.Logger) Scheduler {
	return &Impl{
		jobs:   map[string]job{},
		client: client,
		log:    log,
	}
}

// ScheduleInterval schedules a function to run at regular intervals.
func (s *Impl) ScheduleInterval(key string, interval, timeout time.Duration, fn func(context.Context, logr.Logger)) {
	s.mu.Lock()

	if currentJob, ok := s.jobs[key]; ok {
		if currentJob.interval <= interval {
			s.mu.Unlock()
			return
		}
		currentJob.stop()
	}
	parent := s.ctx
	s.mu.Unlock()

	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.runWithTimeout(fn, timeout)
			case <-ctx.Done():
				return
			}
		}
	}()

	s.log.Info("Scheduled job", "key", key, "interval", interval)
	s.jobs[key] = job{stop: cancel, interval: interval}
}

// Cancel cancels a scheduled job.
func (s *Impl) Cancel(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.jobs[key]; ok {
		e.stop()
		delete(s.jobs, key)
	}
	s.log.Info("Canceled job", "key", key)
}

// NeedLeaderElection returns whether the scheduler needs leader election.
func (s *Impl) NeedLeaderElection() bool { return true }

// Start starts the scheduler.
func (s *Impl) Start(ctx context.Context) error {
	s.log.Info("Starting scheduler")
	s.leader.Store(true)

	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()

	defer func() {
		s.leader.Store(false)
		s.mu.Lock()
		for _, e := range s.jobs {
			e.stop()
		}
		s.mu.Unlock()
	}()

	<-ctx.Done()
	return nil
}

// IsLeader returns whether the scheduler is the leader.
func (s *Impl) IsLeader() bool { return s.leader.Load() }

func (s *Impl) runWithTimeout(fn func(ctx context.Context, log logr.Logger), maxDuration time.Duration) {
	parent := s.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, maxDuration)
	defer cancel()
	fn(ctx, s.log)
}
