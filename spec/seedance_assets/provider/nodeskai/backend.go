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
	pathCreateGroup = "/api/v1/digital-assets/groups/create"
	pathListGroups  = "/api/v1/digital-assets/groups"
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

func parseCreateGroupResponse(raw []byte) (*seedanceassets.AssetGroup, error) {
	var v struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"Id"`
			Name        string `json:"Name"`
			Description string `json:"Description"`
			Status      string `json:"Status"`
			AssetCount  int    `json:"AssetCount"`
			CreateTime  string `json:"CreateTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: parse create group response: %w", err)
	}
	return &seedanceassets.AssetGroup{
		ID:          strings.TrimSpace(v.Data.ID),
		Name:        strings.TrimSpace(v.Data.Name),
		Description: strings.TrimSpace(v.Data.Description),
		Status:      strings.TrimSpace(v.Data.Status),
		AssetCount:  v.Data.AssetCount,
		CreateTime:  strings.TrimSpace(v.Data.CreateTime),
	}, nil
}

func parseListGroupsResponse(raw []byte) (*seedanceassets.AssetGroupList, error) {
	var v struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				ID          string `json:"Id"`
				Name        string `json:"Name"`
				Description string `json:"Description"`
				Status      string `json:"Status"`
				AssetCount  int    `json:"AssetCount"`
				CreateTime  string `json:"CreateTime"`
			} `json:"Items"`
			TotalCount int `json:"TotalCount"`
			PageNumber int `json:"PageNumber"`
			PageSize   int `json:"PageSize"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: parse list groups response: %w", err)
	}
	items := make([]*seedanceassets.AssetGroup, 0, len(v.Data.Items))
	for _, item := range v.Data.Items {
		items = append(items, &seedanceassets.AssetGroup{
			ID:          strings.TrimSpace(item.ID),
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			Status:      strings.TrimSpace(item.Status),
			AssetCount:  item.AssetCount,
			CreateTime:  strings.TrimSpace(item.CreateTime),
		})
	}
	return &seedanceassets.AssetGroupList{
		Items:      items,
		TotalCount: v.Data.TotalCount,
		PageNumber: v.Data.PageNumber,
		PageSize:   v.Data.PageSize,
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
			AssetType  string `json:"AssetType"`
			Status     string `json:"Status"`
			URL        string `json:"Url"`
			URLUpper   string `json:"URL"`
			CreateTime string `json:"CreateTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: parse get asset response: %w", err)
	}
	assetType := strings.TrimSpace(v.Data.Type)
	if assetType == "" {
		assetType = strings.TrimSpace(v.Data.AssetType)
	}
	assetURL := strings.TrimSpace(v.Data.URL)
	if assetURL == "" {
		assetURL = strings.TrimSpace(v.Data.URLUpper)
	}
	return &seedanceassets.Asset{
		ID:         strings.TrimSpace(v.Data.ID),
		GroupID:    strings.TrimSpace(v.Data.GroupID),
		Name:       strings.TrimSpace(v.Data.Name),
		Type:       assetType,
		Status:     strings.TrimSpace(v.Data.Status),
		URL:        assetURL,
		CreateTime: strings.TrimSpace(v.Data.CreateTime),
	}, nil
}

func (b *backend) CreateGroup(ctx context.Context, req *seedanceassets.CreateAssetGroupRequest) (*seedanceassets.AssetGroup, error) {
	return createGroup(ctx, b.client, req)
}

func (b *backend) ListGroups(ctx context.Context, req *seedanceassets.ListAssetGroupsRequest) (*seedanceassets.AssetGroupList, error) {
	return listGroups(ctx, b.client, req)
}

func (b *backend) UploadAsset(ctx context.Context, req *seedanceassets.UploadAssetRequest) (*seedanceassets.AssetUploadResult, error) {
	return uploadAsset(ctx, b.client, req)
}

func (b *backend) GetAsset(ctx context.Context, assetID string) (*seedanceassets.Asset, error) {
	return getAsset(ctx, b.client, assetID)
}
