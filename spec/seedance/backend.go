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

	xai "github.com/goplus/xai/spec"
)

// Backend is the pluggable transport for async Seedance / Ark video tasks.
// Implementations (e.g. provider/volc) POST a task in Submit and poll in GetTaskStatus.
// Submit must return an AsyncOperationResponse (or SyncOperationResponse for mocks) with a stable TaskID for xai.Wait / xai.GetTask.
type Backend interface {
	Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error)
	GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error)
}
