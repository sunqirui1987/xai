/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package shared

import (
	"context"
	"fmt"
	"os"
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/volc"
)

// VideoProvider names which backend was selected for a video model.
type VideoProvider string

const (
	VideoProviderVolcArk        VideoProvider = "volc-ark"         // ARK_API_KEY set
	VideoProviderVolcArkMock    VideoProvider = "volc-ark-mock"    // Seedance model, no ARK_API_KEY
	VideoProviderQiniuKling     VideoProvider = "qiniu-kling"      // QINIU_API_KEY set
	VideoProviderQiniuKlingMock VideoProvider = "qiniu-kling-mock" // Kling video model, no key
)

// VideoGenServiceForModel returns an xai.Service for GenVideo on the given model.
//
// Routing (first match wins):
//   - doubao-seedance-* / known Seedance ids → Volc Ark if ARK_API_KEY else in-process mock
//   - Kling video models → Qiniu Kling if QINIU_API_KEY else existing Kling mock
//
// Env: ARK_API_KEY (火山方舟), QINIU_API_KEY (七牛 Qnagic / Kling).
func VideoGenServiceForModel(model xai.Model) (svc xai.Service, provider VideoProvider, err error) {
	m := strings.TrimSpace(string(model))
	if m == "" {
		return nil, "", fmt.Errorf("empty model")
	}
	if seedance.IsVideoModel(m) {
		if k := strings.TrimSpace(os.Getenv("ARK_API_KEY")); k != "" {
			return volc.NewService(k), VideoProviderVolcArk, nil
		}
		b := &mockSeedanceVideoBackend{}
		return seedance.NewWithBackend(b), VideoProviderVolcArkMock, nil
	}
	if kling.IsVideoModel(m) {
		ks, e := NewService()
		if e != nil {
			return nil, "", e
		}
		if strings.TrimSpace(os.Getenv("QINIU_API_KEY")) != "" {
			return ks, VideoProviderQiniuKling, nil
		}
		return ks, VideoProviderQiniuKlingMock, nil
	}
	return nil, "", fmt.Errorf("unsupported video model %q (not Seedance nor Kling video)", m)
}

type mockSeedanceVideoBackend struct{}

func (m *mockSeedanceVideoBackend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	_ = model
	_ = params
	return &seedance.SyncOperationResponse{
		R: seedance.NewOutputVideos([]string{"https://aitoken-public.qnaigc.com/example/generate-video/the-little-dog-is-running-on-the-lawn.mp4"}),
	}, nil
}

func (m *mockSeedanceVideoBackend) GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error) {
	_ = taskID
	return nil, xai.ErrNotSupported
}
