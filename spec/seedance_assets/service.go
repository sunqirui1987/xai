/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package seedanceassets

import (
	"context"
	"strings"
	"time"
)

// Service provides Seedance 2.0 digital asset operations.
type Service struct {
	backend Backend
}

// NewWithBackend creates a Service with the given backend.
func NewWithBackend(backend Backend) *Service {
	if backend == nil {
		panic("seedance_assets: nil backend")
	}
	return &Service{backend: backend}
}

// NewService is an alias for NewWithBackend.
func NewService(backend Backend) *Service {
	return NewWithBackend(backend)
}

// CreateGroup creates one asset group in the provider-managed digital asset library.
func (s *Service) CreateGroup(ctx context.Context, req *CreateAssetGroupRequest) (*AssetGroup, error) {
	return s.backend.CreateGroup(ctx, req)
}

// ListGroups lists asset groups in the provider-managed digital asset library.
func (s *Service) ListGroups(ctx context.Context, req *ListAssetGroupsRequest) (*AssetGroupList, error) {
	return s.backend.ListGroups(ctx, req)
}

// UploadAsset uploads one asset into the provider-managed digital asset library.
func (s *Service) UploadAsset(ctx context.Context, req *UploadAssetRequest) (*AssetUploadResult, error) {
	return s.backend.UploadAsset(ctx, req)
}

// GetAsset loads one asset from the provider-managed digital asset library.
func (s *Service) GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	return s.backend.GetAsset(ctx, assetID)
}

// AwaitAsset polls GetAsset until the asset leaves Processing state or the context ends.
func (s *Service) AwaitAsset(ctx context.Context, assetID string, interval time.Duration) (*Asset, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	for {
		asset, err := s.GetAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if !isAssetProcessing(asset.Status) {
			return asset, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// UploadAndAwaitAsset uploads one asset, waits until it leaves Processing, and returns
// a compact reference with both the asset URI and the resolved storage URL.
func (s *Service) UploadAndAwaitAsset(ctx context.Context, req *UploadAssetRequest, interval time.Duration) (*AssetRef, error) {
	uploaded, err := s.UploadAsset(ctx, req)
	if err != nil {
		return nil, err
	}
	asset, err := s.AwaitAsset(ctx, uploaded.AssetID, interval)
	if err != nil {
		return nil, err
	}
	assetID := strings.TrimSpace(asset.ID)
	ref := &AssetRef{
		URL:    strings.TrimSpace(asset.URL),
		Status: strings.TrimSpace(asset.Status),
	}
	if assetID != "" {
		ref.Asset = assetScheme(assetID) + assetID
	}
	return ref, nil
}

func isAssetProcessing(status string) bool {
	st := strings.TrimSpace(status)
	return strings.EqualFold(st, AssetStatusProcessing) ||
		strings.EqualFold(st, AssetStatusPending) ||
		strings.EqualFold(st, AssetStatusReviewing)
}

func assetScheme(assetID string) string {
	if strings.HasPrefix(strings.TrimSpace(assetID), "qasset-") {
		return "qasset://"
	}
	return "asset://"
}
