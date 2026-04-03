/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package seedance

import (
	"context"
	"time"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/vidu/video"
)

// NewOutputVideos builds xai.Results for completed GenVideo (reuses spec/vidu/video output types).
func NewOutputVideos(urls []string) xai.Results {
	return video.NewOutputVideos(urls)
}

// SyncOperationResponse is a completed xai.OperationResponse.
type SyncOperationResponse struct {
	R xai.Results
}

func (p *SyncOperationResponse) Done() bool { return true }
func (p *SyncOperationResponse) Sleep()     {}
func (p *SyncOperationResponse) Retry(ctx context.Context, svc xai.Service) (xai.OperationResponse, error) {
	return p, nil
}
func (p *SyncOperationResponse) Results() xai.Results { return p.R }
func (p *SyncOperationResponse) TaskID() string       { return "" }

// AsyncOperationResponse represents a running Ark task; Retry calls Backend.GetTaskStatus.
// Default SleepDur is 2s between polls. Volc backend logs each GetTaskStatus via Client.LogDebug.
type AsyncOperationResponse struct {
	RetryFunc func(ctx context.Context) (xai.OperationResponse, error)
	SleepDur  time.Duration
	taskID    string
}

// NewAsyncOperationResponse creates an async polling response.
func NewAsyncOperationResponse(retryFunc func(ctx context.Context) (xai.OperationResponse, error), taskID string) *AsyncOperationResponse {
	return &AsyncOperationResponse{
		RetryFunc: retryFunc,
		SleepDur:  2 * time.Second,
		taskID:    taskID,
	}
}

func (p *AsyncOperationResponse) Done() bool { return false }

func (p *AsyncOperationResponse) Sleep() {
	if p.SleepDur > 0 {
		time.Sleep(p.SleepDur)
	}
}

func (p *AsyncOperationResponse) Retry(ctx context.Context, svc xai.Service) (xai.OperationResponse, error) {
	if p.RetryFunc == nil {
		return p, nil
	}
	return p.RetryFunc(ctx)
}

func (p *AsyncOperationResponse) Results() xai.Results { return nil }
func (p *AsyncOperationResponse) TaskID() string       { return p.taskID }
