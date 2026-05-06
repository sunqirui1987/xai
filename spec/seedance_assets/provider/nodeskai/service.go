/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package nodeskai

import (
	"strings"

	seedancenodeskai "github.com/goplus/xai/spec/seedance/provider/nodeskai"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

// Service wraps seedanceassets.Service and holds the NoDesk platform client.
type Service struct {
	*seedanceassets.Service
	client *seedancenodeskai.Client
}

// SetApiKey updates the bearer access token on the underlying client.
func (s *Service) SetApiKey(accessToken string) {
	s.client.SetApiKey(accessToken)
}

// NewService constructs a Seedance assets service with a bearer access token.
func NewService(accessToken string, opts ...seedancenodeskai.ClientOption) *Service {
	client := seedancenodeskai.NewClient(strings.TrimSpace(accessToken), opts...)
	return &Service{
		Service: seedanceassets.NewWithBackend(newBackend(client)),
		client:  client,
	}
}

// NewServiceWithClientCredentials constructs a Seedance assets service that obtains
// access tokens via OAuth2 client_credentials.
func NewServiceWithClientCredentials(clientID, clientSecret string, opts ...seedancenodeskai.ClientOption) *Service {
	client := seedancenodeskai.NewClient("", append(opts, seedancenodeskai.WithOAuthClientCredentials(clientID, clientSecret))...)
	return &Service{
		Service: seedanceassets.NewWithBackend(newBackend(client)),
		client:  client,
	}
}
