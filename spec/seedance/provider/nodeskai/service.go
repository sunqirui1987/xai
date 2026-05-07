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

package nodeskai

import (
	"os"
	"strings"

	"github.com/goplus/xai/spec/seedance"
)

// Service wraps seedance.Service and holds the NoDesk AI HTTP client for SetApiKey.
type Service struct {
	*seedance.Service
	client *Client
}

// SetApiKey updates the NoDesk AI bearer access token on the underlying client.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
}

// SeedanceService returns the embedded *seedance.Service for GenVideo Operation.Call.
func (s *Service) SeedanceService() *seedance.Service { return s.Service }

// NewService constructs a Seedance service with NoDesk AI backend.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("NODESKAI_ACCESS_TOKEN")
	}
	if apiKey == "" {
		apiKey = os.Getenv("NODESK_ACCESS_TOKEN")
	}
	client := NewClient(apiKey, opts...)
	return newServiceWithClient(client)
}

// NewServiceWithClientCredentials constructs a Seedance service that obtains bearer
// access tokens via OAuth2 client_credentials.
func NewServiceWithClientCredentials(apiKey string, clientID, clientSecret string, opts ...ClientOption) *Service {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	apiKey = strings.TrimSpace(apiKey)
	client := NewClient(apiKey, append(opts, WithOAuthClientCredentials(clientID, clientSecret))...)
	return newServiceWithClient(client)
}

func newServiceWithClient(client *Client) *Service {
	return &Service{
		Service: seedance.NewWithBackend(newBackend(client)),
		client:  client,
	}
}

// Register registers seedance:// globally with a NoDesk AI-backed service.
func Register(apiKey string, opts ...ClientOption) {
	svc := NewService(apiKey, opts...)
	seedance.Register(svc)
}

// RegisterWithClientCredentials registers seedance:// globally with a NoDesk AI-backed
// service that uses OAuth2 client_credentials.
func RegisterWithClientCredentials(apiKey string, clientID, clientSecret string, opts ...ClientOption) {
	svc := NewServiceWithClientCredentials(apiKey, clientID, clientSecret, opts...)
	seedance.Register(svc)
}
