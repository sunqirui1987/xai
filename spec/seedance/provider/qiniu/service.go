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

package qiniu

import (
	"os"

	"github.com/goplus/xai/spec/seedance"
)

// Service wraps seedance.Service and holds the Qiniu HTTP client for SetApiKey.
type Service struct {
	*seedance.Service
	client  *Client
	backend *backend
}

// SetApiKey updates the Qiniu API key on the underlying client.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
	if s.backend != nil && s.backend.assetClient != nil {
		s.backend.assetClient.SetApiKey(apiKey)
	}
}

// SeedanceService returns the embedded *seedance.Service for GenVideo Operation.Call.
func (s *Service) SeedanceService() *seedance.Service { return s.Service }

// NewService constructs a Seedance service with Qiniu backend.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("QINIU_API_KEY")
	}
	client := NewClient(apiKey, opts...)
	backend := newBackend(client)
	return &Service{
		Service: seedance.NewWithBackend(backend),
		client:  client,
		backend: backend,
	}
}

// Register registers seedance:// globally with a Qiniu-backed service.
func Register(apiKey string, opts ...ClientOption) {
	svc := NewService(apiKey, opts...)
	seedance.Register(svc)
}
