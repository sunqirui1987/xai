/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

// Package seedance defines the xai spec for Volc Ark Seedance / 豆包视频生成 (async GenVideo).
//
// Types: Backend (Submit + GetTaskStatus), Service (FeatureOperation), Params (Ark field names),
// SyncOperationResponse / AsyncOperationResponse (poll via xai.Wait; prefer a non-nil progress callback).
// Provider wrappers must implement SeedanceService() *Service for Operation.Call.
//
// Official Ark docs (Chinese):
//   - https://www.volcengine.com/docs/82379/1520757?lang=zh — create video generation task
//   - https://www.volcengine.com/docs/82379/1521309?lang=zh — get task status
//
// HTTP backend, curl-style debug logs, and task lifecycle logging: provider/volc (see README.md there).
package seedance

import (
	"context"
	"strings"

	xai "github.com/goplus/xai/spec"
)

// Scheme is the URI scheme for Seedance (Volc Ark): "seedance".
const Scheme = "seedance"

// Default video models (Seedance 2.0 on Volc Ark).
const (
	ModelDoubaoSeedance20     = "doubao-seedance-2-0-260128"
	ModelDoubaoSeedance20Fast = "doubao-seedance-2-0-fast-260128"

	ModelDreaminaSeedance20         = "dreamina-seedance-2-0-260128"
	ModelByteplusDreaminaSeedance20 = "byteplus/dreamina-seedance-2-0-260128"
)

var defaultVideoModels = []string{ModelDoubaoSeedance20, ModelDoubaoSeedance20Fast, ModelDreaminaSeedance20, ModelByteplusDreaminaSeedance20}

// IsVideoModel reports whether the model id is supported for GenVideo on this spec.
// Known IDs are listed in VideoModels; any model id with prefix "doubao-seedance-" or
// "dreamina-seedance-" is also accepted.
func IsVideoModel(model string) bool {
	m := strings.TrimSpace(strings.ToLower(model))
	if m == "" {
		return false
	}
	for _, id := range defaultVideoModels {
		if strings.EqualFold(m, strings.ToLower(id)) {
			return true
		}
	}
	return strings.HasPrefix(m, "doubao-seedance-") || strings.HasPrefix(m, "dreamina-seedance-")
}

// VideoModels returns built-in model IDs (excluding arbitrary doubao-seedance-* /
// dreamina-seedance-* variants).
func VideoModels() []string {
	out := make([]string, len(defaultVideoModels))
	copy(out, defaultVideoModels)
	return out
}

// Register registers the Seedance service with xai.
func Register(svc xai.Service) {
	xai.Register(Scheme, func(ctx context.Context, uri string) (xai.Service, error) {
		return svc, nil
	})
}
