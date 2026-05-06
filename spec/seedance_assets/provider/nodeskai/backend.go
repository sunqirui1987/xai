/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package nodeskai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	seedancenodeskai "github.com/goplus/xai/spec/seedance/provider/nodeskai"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	pathUploadAsset = "/api/v1/digital-assets/upload"
	pathGetAsset    = "/api/v1/digital-assets/"
)

type backend struct {
	client *seedancenodeskai.Client
}

func newBackend(client *seedancenodeskai.Client) *backend {
	return &backend{client: client}
}

// NewBackend returns a seedanceassets.Backend backed by the NoDesk platform API.
func NewBackend(client *seedancenodeskai.Client) seedanceassets.Backend {
	return newBackend(client)
}

func parseUploadAssetResponse(raw []byte) (*seedanceassets.AssetUploadResult, error) {
	var v struct {
		Success   bool   `json:"success"`
		AssetID   string `json:"asset_id"`
		AssetType string `json:"asset_type"`
		Status    string `json:"status"`
		TOSURL    string `json:"tos_url"`
		FileName  string `json:"file_name"`
		FileSize  int64  `json:"file_size"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: parse upload asset response: %w", err)
	}
	return &seedanceassets.AssetUploadResult{
		Success:   v.Success,
		AssetID:   strings.TrimSpace(v.AssetID),
		AssetType: strings.TrimSpace(v.AssetType),
		Status:    strings.TrimSpace(v.Status),
		TOSURL:    strings.TrimSpace(v.TOSURL),
		FileName:  strings.TrimSpace(v.FileName),
		FileSize:  v.FileSize,
	}, nil
}

func parseGetAssetResponse(raw []byte) (*seedanceassets.Asset, error) {
	var v struct {
		Success bool `json:"success"`
		Data    struct {
			ID         string `json:"Id"`
			GroupID    string `json:"GroupId"`
			Name       string `json:"Name"`
			Type       string `json:"Type"`
			Status     string `json:"Status"`
			URL        string `json:"Url"`
			CreateTime string `json:"CreateTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: parse get asset response: %w", err)
	}
	return &seedanceassets.Asset{
		ID:         strings.TrimSpace(v.Data.ID),
		GroupID:    strings.TrimSpace(v.Data.GroupID),
		Name:       strings.TrimSpace(v.Data.Name),
		Type:       strings.TrimSpace(v.Data.Type),
		Status:     strings.TrimSpace(v.Data.Status),
		URL:        strings.TrimSpace(v.Data.URL),
		CreateTime: strings.TrimSpace(v.Data.CreateTime),
	}, nil
}

func (b *backend) UploadAsset(ctx context.Context, req *seedanceassets.UploadAssetRequest) (*seedanceassets.AssetUploadResult, error) {
	return uploadAsset(ctx, b.client, req)
}

func (b *backend) GetAsset(ctx context.Context, assetID string) (*seedanceassets.Asset, error) {
	return getAsset(ctx, b.client, assetID)
}
