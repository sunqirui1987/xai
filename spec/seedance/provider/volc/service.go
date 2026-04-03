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

package volc

import (
	"os"

	"github.com/goplus/xai/spec/seedance"
)

// Service wraps seedance.Service and holds the Volc HTTP client for SetApiKey.
type Service struct {
	*seedance.Service
	client *Client
}

// SetApiKey updates the Ark API key on the underlying client. Implements xai.ApiKeySetter.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
}

// SeedanceService returns the embedded *seedance.Service for GenVideo Operation.Call.
func (s *Service) SeedanceService() *seedance.Service { return s.Service }

// NewService constructs a Seedance service with Volc Ark backend.
// opts are applied to NewClient (e.g. WithBaseURL, WithDebugLog(false), WithRetry(3, time.Second)).
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("ARK_API_KEY")
	}
	client := NewClient(apiKey, opts...)
	return &Service{
		Service: seedance.NewWithBackend(newBackend(client)),
		client:  client,
	}
}

// Register registers seedance:// globally with a Volc-backed service.
func Register(apiKey string, opts ...ClientOption) {
	svc := NewService(apiKey, opts...)
	seedance.Register(svc)
}
