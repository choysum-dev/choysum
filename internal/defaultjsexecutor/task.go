// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

// taskStatus represents the current status of a task.
type taskStatus int

const (
	taskStatusPending   taskStatus = iota // Task is waiting to be executed
	taskStatusRunning                     // Task is currently being executed
	taskStatusCompleted                   // Task execution has completed
)

// taskResult represents the result of task execution.
type taskResult struct {
	response *jsengine.JsResponse // JavaScript execution response (nil if error occurred)
	err      error                // Error that occurred during execution (nil if successful)
}

// task represents a queued execution request processed by a worker thread.
type task struct {
	ctx        context.Context     // Go request context (trusted; Go-only)
	request    *jsengine.JsRequest // JavaScript request to execute
	resultChan chan *taskResult    // Channel to receive the execution result
	status     taskStatus          // Current status of the task
}

// newTask creates a new task instance for the given request.
func newTask(ctx context.Context, request *jsengine.JsRequest) *task {
	if ctx == nil {
		ctx = context.Background()
	}
	return &task{
		ctx:        ctx,
		request:    request,
		resultChan: make(chan *taskResult, 1), // Buffered channel to prevent blocking
		status:     taskStatusPending,
	}
}
