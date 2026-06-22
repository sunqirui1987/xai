/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package turbo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	seedanceturbo "github.com/goplus/xai/spec/seedance/provider/turbo"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	pathGroups = "/api/v1/groups"
	pathAssets = "/api/v1/assets"
)

type backend struct {
	client *seedanceturbo.Client
}

func newBackend(client *seedanceturbo.Client) *backend {
	return &backend{client: client}
}

// NewBackend returns a seedanceassets.Backend backed by the Turbo/Yoofang asset API.
func NewBackend(client *seedanceturbo.Client) seedanceassets.Backend {
	return newBackend(client)
}

func (b *backend) CreateGroup(ctx context.Context, req *seedanceassets.CreateAssetGroupRequest) (*seedanceassets.AssetGroup, error) {
	body, err := buildCreateGroupBody(req)
	if err != nil {
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathGroups, body)
	if err != nil {
		return nil, err
	}
	return parseCreateGroupResponse(raw)
}

func (b *backend) ListGroups(ctx context.Context, req *seedanceassets.ListAssetGroupsRequest) (*seedanceassets.AssetGroupList, error) {
	r := req.Normalize()
	path := fmt.Sprintf("%s?page=%d&size=%d", pathGroups, r.PageNumber, r.PageSize)
	raw, err := b.client.GetJSON(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseListGroupsResponse(raw)
}

func (b *backend) UploadAsset(ctx context.Context, req *seedanceassets.UploadAssetRequest) (*seedanceassets.AssetUploadResult, error) {
	body, err := buildUploadAssetBody(req)
	if err != nil {
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathAssets, body)
	if err != nil {
		return nil, err
	}
	return parseUploadAssetResponse(raw)
}

func (b *backend) GetAsset(ctx context.Context, assetID string) (*seedanceassets.Asset, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil, fmt.Errorf("seedance_assets/turbo: asset id is required")
	}
	raw, err := b.client.GetJSON(ctx, pathAssets+"/"+url.PathEscape(assetID))
	if err != nil {
		return nil, err
	}
	return parseGetAssetResponse(raw)
}

func buildCreateGroupBody(req *seedanceassets.CreateAssetGroupRequest) (map[string]any, error) {
	if req == nil {
		return nil, fmt.Errorf("seedance_assets/turbo: nil create asset group request")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("seedance_assets/turbo: group name is required")
	}
	body := map[string]any{"name": name}
	if desc := strings.TrimSpace(req.Description); desc != "" {
		body["description"] = desc
	}
	return body, nil
}

func buildUploadAssetBody(req *seedanceassets.UploadAssetRequest) (map[string]any, error) {
	if req == nil {
		return nil, fmt.Errorf("seedance_assets/turbo: nil upload asset request")
	}
	groupID := strings.TrimSpace(req.GroupID)
	if groupID == "" {
		return nil, fmt.Errorf("seedance_assets/turbo: group_id is required")
	}
	assetURL := strings.TrimSpace(req.URL)
	if assetURL == "" {
		return nil, fmt.Errorf("seedance_assets/turbo: url is required")
	}
	assetType := normalizeAssetType(req.AssetType)
	if assetType == "" {
		assetType = seedanceassets.AssetTypeImage
	}
	body := map[string]any{
		"group_id":   groupID,
		"asset_type": assetType,
		"url":        assetURL,
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		body["name"] = name
	}
	return body, nil
}

func normalizeAssetType(assetType string) string {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case "image":
		return seedanceassets.AssetTypeImage
	case "video":
		return seedanceassets.AssetTypeVideo
	default:
		return strings.TrimSpace(assetType)
	}
}

func parseCreateGroupResponse(raw []byte) (*seedanceassets.AssetGroup, error) {
	var v struct {
		ID          string `json:"id"`
		GroupID     string `json:"group_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/turbo: parse create group response: %w", err)
	}
	id := strings.TrimSpace(v.ID)
	if id == "" {
		id = strings.TrimSpace(v.GroupID)
	}
	return &seedanceassets.AssetGroup{
		ID:          id,
		Name:        strings.TrimSpace(v.Name),
		Description: strings.TrimSpace(v.Description),
	}, nil
}

func parseListGroupsResponse(raw []byte) (*seedanceassets.AssetGroupList, error) {
	var v struct {
		Items []struct {
			ID          string `json:"id"`
			GroupID     string `json:"group_id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		} `json:"items"`
		Total int `json:"total"`
		Page  int `json:"page"`
		Size  int `json:"size"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/turbo: parse list groups response: %w", err)
	}
	items := make([]*seedanceassets.AssetGroup, 0, len(v.Items))
	for _, item := range v.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = strings.TrimSpace(item.GroupID)
		}
		items = append(items, &seedanceassets.AssetGroup{
			ID:          id,
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			CreateTime:  strings.TrimSpace(item.CreatedAt),
		})
	}
	return &seedanceassets.AssetGroupList{
		Items:      items,
		TotalCount: v.Total,
		PageNumber: v.Page,
		PageSize:   v.Size,
	}, nil
}

func parseUploadAssetResponse(raw []byte) (*seedanceassets.AssetUploadResult, error) {
	var v struct {
		ID        string `json:"id"`
		AssetID   string `json:"asset_id"`
		Status    string `json:"status"`
		GroupID   string `json:"group_id"`
		AssetType string `json:"asset_type"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/turbo: parse upload asset response: %w", err)
	}
	id := strings.TrimSpace(v.ID)
	if id == "" {
		id = strings.TrimSpace(v.AssetID)
	}
	return &seedanceassets.AssetUploadResult{
		Success:    !strings.EqualFold(strings.TrimSpace(v.Status), "failed"),
		AssetID:    id,
		AssetType:  strings.TrimSpace(v.AssetType),
		Status:     strings.TrimSpace(v.Status),
		GroupID:    strings.TrimSpace(v.GroupID),
		FailReason: strings.TrimSpace(v.Error),
	}, nil
}

func parseGetAssetResponse(raw []byte) (*seedanceassets.Asset, error) {
	var v struct {
		ID        string `json:"id"`
		AssetID   string `json:"asset_id"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		AssetType string `json:"asset_type"`
		GroupID   string `json:"group_id"`
		Status    string `json:"status"`
		Error     string `json:"error"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Data      struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			URL       string `json:"url"`
			AssetType string `json:"asset_type"`
			GroupID   string `json:"group_id"`
			Status    string `json:"status"`
			Error     string `json:"error"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/turbo: parse get asset response: %w", err)
	}
	if strings.TrimSpace(v.ID) == "" && strings.TrimSpace(v.Data.ID) != "" {
		v.ID = v.Data.ID
		v.Name = v.Data.Name
		v.URL = v.Data.URL
		v.AssetType = v.Data.AssetType
		v.GroupID = v.Data.GroupID
		v.Status = v.Data.Status
		v.Error = v.Data.Error
		v.CreatedAt = v.Data.CreatedAt
		v.UpdatedAt = v.Data.UpdatedAt
	}
	id := strings.TrimSpace(v.ID)
	if id == "" {
		id = strings.TrimSpace(v.AssetID)
	}
	return &seedanceassets.Asset{
		ID:         id,
		GroupID:    strings.TrimSpace(v.GroupID),
		Name:       strings.TrimSpace(v.Name),
		Type:       strings.TrimSpace(v.AssetType),
		Status:     strings.TrimSpace(v.Status),
		URL:        strings.TrimSpace(v.URL),
		FailReason: strings.TrimSpace(v.Error),
		CreateTime: strings.TrimSpace(v.CreatedAt),
	}, nil
}
