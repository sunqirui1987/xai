/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package turbo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

func (b *backend) createAssetGroup(ctx context.Context, name, description string) (*seedanceassets.AssetGroup, error) {
	body := map[string]any{
		"name": strings.TrimSpace(name),
	}
	if desc := strings.TrimSpace(description); desc != "" {
		body["description"] = desc
	}
	raw, err := b.assetClient.PostJSON(ctx, pathAssetGroups, body)
	if err != nil {
		return nil, err
	}
	return parseCreateAssetGroupResponse(raw)
}

func (b *backend) listAssetGroups(ctx context.Context) (*seedanceassets.AssetGroupList, error) {
	raw, err := b.assetClient.GetJSON(ctx, pathAssetGroups)
	if err != nil {
		return nil, err
	}
	return parseListAssetGroupsResponse(raw)
}

func (b *backend) findAssetGroupByName(ctx context.Context, name string) (*seedanceassets.AssetGroup, error) {
	list, err := b.listAssetGroups(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range list.Items {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(name)) {
			return item, nil
		}
	}
	return nil, nil
}

func (b *backend) createAsset(ctx context.Context, groupID, rawURL, assetType, name string) (*seedanceassets.AssetUploadResult, error) {
	body := map[string]any{
		"group_id":   strings.TrimSpace(groupID),
		"asset_type": normalizeTurboAssetType(assetType),
		"url":        strings.TrimSpace(rawURL),
	}
	if n := strings.TrimSpace(name); n != "" {
		body["name"] = n
	}
	raw, err := b.assetClient.PostJSON(ctx, pathAssets, body)
	if err != nil {
		return nil, err
	}
	return parseCreateAssetResponse(raw)
}

func (b *backend) getAsset(ctx context.Context, assetID string) (*seedanceassets.Asset, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil, fmt.Errorf("turbo-seedance: asset id is required")
	}
	raw, err := b.assetClient.GetJSON(ctx, pathAssets+"/"+url.PathEscape(assetID))
	if err != nil {
		return nil, err
	}
	return parseGetAssetResponse(raw)
}

func (b *backend) awaitAsset(ctx context.Context, assetID string, interval time.Duration, attempts int) (*seedanceassets.Asset, error) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	if attempts <= 0 {
		attempts = 40
	}
	for i := 0; i < attempts; i++ {
		asset, err := b.getAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		status := normalizeAssetStatus(asset.Status)
		b.client.LogDebug("asset upload asset_id=%q status=%q", asset.ID, asset.Status)
		switch status {
		case "completed":
			return asset, nil
		case "failed":
			reason := strings.TrimSpace(asset.FailReason)
			if reason == "" {
				reason = "unknown error"
			}
			return nil, fmt.Errorf("turbo-seedance: asset upload failed: %s", reason)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
	return nil, fmt.Errorf("turbo-seedance: asset upload timeout: %s", assetID)
}

func parseCreateAssetGroupResponse(raw []byte) (*seedanceassets.AssetGroup, error) {
	var v struct {
		ID          string `json:"id"`
		GroupID     string `json:"group_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("turbo-seedance: parse create group response: %w", err)
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

func parseListAssetGroupsResponse(raw []byte) (*seedanceassets.AssetGroupList, error) {
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
		return nil, fmt.Errorf("turbo-seedance: parse list groups response: %w", err)
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

func parseCreateAssetResponse(raw []byte) (*seedanceassets.AssetUploadResult, error) {
	var v struct {
		ID        string `json:"id"`
		AssetID   string `json:"asset_id"`
		TaskID    string `json:"task_id"`
		Status    string `json:"status"`
		GroupID   string `json:"group_id"`
		AssetType string `json:"asset_type"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("turbo-seedance: parse create asset response: %w", err)
	}
	id := strings.TrimSpace(v.ID)
	if id == "" {
		id = strings.TrimSpace(v.AssetID)
	}
	return &seedanceassets.AssetUploadResult{
		Success:    normalizeAssetStatus(v.Status) != "failed",
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
		return nil, fmt.Errorf("turbo-seedance: parse get asset response: %w", err)
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

func normalizeTurboAssetType(assetType string) string {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case "image":
		return seedanceassets.AssetTypeImage
	case "video":
		return seedanceassets.AssetTypeVideo
	default:
		return strings.TrimSpace(assetType)
	}
}

func normalizeAssetStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func newAssetClient(client *Client) *Client {
	apiKey := ""
	debugLog := true
	logger := log.Default()
	if client != nil {
		apiKey = client.ApiKey()
		debugLog = client.debugLog
		logger = client.logger
	}
	baseURL := strings.TrimSpace(os.Getenv("TURBO_ASSETS_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultAssetsBaseURL
	}
	return NewClient(apiKey, WithBaseURL(baseURL), WithDebugLog(debugLog), WithLogger(logger))
}
