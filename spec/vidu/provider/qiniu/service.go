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

	"github.com/goplus/xai/spec/vidu"
)

// Service wraps vidu.Service with ApiKeySetter for runtime configuration.
type Service struct {
	*vidu.Service
	client *Client
}

// SetApiKey updates the API key at runtime. Implements xai.ApiKeySetter.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
}

// ViduService returns the embedded *vidu.Service for operation.Call.
func (s *Service) ViduService() *vidu.Service { return s.Service }

// NewService creates a Vidu Service with Qiniu backend.
// The returned *Service supports SetApiKey(apiKey) for runtime API key updates.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("QINIU_API_KEY")
	}
	client := NewClient(apiKey, opts...)
	return &Service{
		Service: vidu.NewWithBackend(newBackend(client)),
		client:  client,
	}
}

// Register creates a vidu.Service with Qiniu backend and registers it with xai.
func Register(apiKey string, opts ...ClientOption) {
	svc := NewService(apiKey, opts...)
	vidu.Register(svc)
}
