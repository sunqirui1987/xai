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
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	pathVideoGenerations = "/v1/video/generations"
	pathAssetGroups      = "/api/v1/groups"
	pathAssets           = "/api/v1/assets"

	defaultAssetsBaseURL  = "https://inference.turbo-api.com"
	defaultAssetGroupName = "默认素材组"
	defaultAssetGroupDesc = "Seedance 2.0 默认素材组"

	ParamContent            = "content"
	ParamAssetGroupID       = "asset_group_id"
	ParamAssetAutoReview    = "asset_auto_review"
	ParamAssetPollInterval  = "asset_poll_interval"
	ParamAssetPollAttempts  = "asset_poll_attempts"
	ParamAssetReviewRetries = "asset_review_retries"
)

// ErrTaskFailed is returned when Turbo reports a terminal failure.
var ErrTaskFailed = errors.New("turbo-seedance: task failed")

type backend struct {
	client      *Client
	assetClient *Client
	assetURLMu  sync.RWMutex
	assetURLMap map[string]string
}

func newBackend(client *Client) *backend {
	return &backend{
		client:      client,
		assetClient: newAssetClient(client),
		assetURLMap: make(map[string]string),
	}
}

// NewBackend returns a seedance.Backend backed by the Turbo HTTP API.
func NewBackend(client *Client) seedance.Backend {
	return newBackend(client)
}

// Submit creates an async generation task and returns a polling OperationResponse.
func (b *backend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	p, ok := params.(*seedance.Params)
	if !ok {
		return nil, fmt.Errorf("turbo-seedance: expected *seedance.Params, got %T", params)
	}
	m := strings.TrimSpace(string(model))
	b.client.LogDebug("Submit GenVideo model=%q", m)

	prepared, err := b.prepareParams(ctx, p)
	if err != nil {
		b.client.LogDebug("Submit prepareParams error: %v", err)
		return nil, err
	}
	body, err := buildTaskBody(m, prepared)
	if err != nil {
		b.client.LogDebug("Submit buildTaskBody error: %v", err)
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathVideoGenerations, body)
	if err != nil {
		b.client.LogDebug("Submit POST error: %v", err)
		return nil, err
	}
	taskID, err := parseCreateTaskID(raw)
	if err != nil {
		b.client.LogDebug("Submit parse task id error: %v", err)
		return nil, err
	}
	b.client.LogDebug("Submit created task_id=%q (async polling)", taskID)
	return b.newPollingResponse(taskID), nil
}

// GetTaskStatus loads task state and converts terminal success/failure into xai.OperationResponse.
func (b *backend) GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("turbo-seedance: task id is required")
	}
	b.client.LogDebug("GetTaskStatus task_id=%q", taskID)
	raw, err := b.client.GetJSON(ctx, pathVideoGenerations+"/"+url.PathEscape(taskID))
	if err != nil {
		b.client.LogDebug("GetTaskStatus request error: %v", err)
		return nil, err
	}
	status, videoURL, failMsg, err := parseTaskGetResponse(raw)
	if err != nil {
		b.client.LogDebug("GetTaskStatus parse error: %v", err)
		return nil, err
	}
	st := normalizeStatus(status)
	b.client.LogDebug("GetTaskStatus task_id=%q raw_status=%q normalized=%q video_url_present=%v", taskID, status, st, videoURL != "")

	switch {
	case isSuccessStatus(st):
		if videoURL == "" {
			return nil, fmt.Errorf("%w: succeeded but no video_url in response", ErrTaskFailed)
		}
		return &seedance.SyncOperationResponse{R: seedance.NewOutputVideos([]string{videoURL})}, nil
	case isFailedStatus(st):
		if failMsg == "" {
			failMsg = "unknown error"
		}
		return nil, fmt.Errorf("%w: %s", ErrTaskFailed, failMsg)
	default:
		return b.newPollingResponse(taskID), nil
	}
}

func (b *backend) newPollingResponse(taskID string) xai.OperationResponse {
	return seedance.NewAsyncOperationResponse(func(ctx context.Context) (xai.OperationResponse, error) {
		return b.GetTaskStatus(ctx, taskID)
	}, taskID)
}

func buildTaskBody(model string, p *seedance.Params) (map[string]any, error) {
	m := strings.TrimSpace(model)
	if m == "" {
		return nil, fmt.Errorf("turbo-seedance: empty model")
	}

	content, err := buildContent(p)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"model":   m,
		"content": content,
	}
	if d := p.GetInt(seedance.ParamDuration); d != nil {
		if err := seedance.ValidateVideoDuration(m, *d); err != nil {
			return nil, err
		}
		body["duration"] = *d
	}
	if resolution := p.GetString("resolution"); resolution != "" {
		body["resolution"] = resolution
	}
	if r := p.GetString(seedance.ParamRatio); r != "" {
		body["ratio"] = r
	}
	if ga := p.GetBool(seedance.ParamGenerateAudio); ga != nil {
		body["generate_audio"] = *ga
	}
	if w := p.GetBool(seedance.ParamWatermark); w != nil {
		body["watermark"] = *w
	}
	return body, nil
}

func buildContent(p *seedance.Params) ([]any, error) {
	if raw, ok := p.Get(ParamContent); ok {
		content := normalizeContent(raw)
		if len(content) == 0 {
			return nil, fmt.Errorf("turbo-seedance: content is empty")
		}
		return content, nil
	}

	text := p.PrimaryText()
	if text == "" {
		return nil, seedance.ErrTextRequired
	}
	content := []any{
		map[string]any{
			"type": "text",
			"text": text,
		},
	}
	for _, ref := range p.GetReferenceImages(seedance.ParamReferenceImages) {
		role := strings.TrimSpace(ref.Role)
		if role == "" {
			role = "reference_image"
		}
		content = append(content, mediaURLContent("image_url", "image_url", ref.URL, role))
	}
	if len(content) == 1 {
		for _, refURL := range p.GetStringSlice(seedance.ParamReferenceImageURLs) {
			content = append(content, mediaURLContent("image_url", "image_url", refURL, "reference_image"))
		}
	}
	for _, refURL := range p.GetStringSlice(seedance.ParamReferenceVideoURLs) {
		content = append(content, mediaURLContent("video_url", "video_url", refURL, "reference_video"))
	}
	for _, refURL := range p.GetStringSlice(seedance.ParamReferenceAudioURLs) {
		content = append(content, mediaURLContent("audio_url", "audio_url", refURL, "reference_audio"))
	}
	return content, nil
}

func normalizeContent(raw any) []any {
	switch x := raw.(type) {
	case []any:
		return x
	case []map[string]any:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func mediaURLContent(contentType string, field string, u string, role string) map[string]any {
	item := map[string]any{
		"type": contentType,
		field: map[string]any{
			"url": u,
		},
	}
	if strings.TrimSpace(role) != "" {
		item["role"] = role
	}
	return item
}

func parseCreateTaskID(raw []byte) (string, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("turbo-seedance: parse create response: %w", err)
	}
	if id := stringVal(v, "id"); id != "" {
		return id, nil
	}
	if id := stringVal(v, "task_id"); id != "" {
		return id, nil
	}
	if d, ok := v["data"].(map[string]any); ok {
		if id := stringVal(d, "id"); id != "" {
			return id, nil
		}
		if id := stringVal(d, "task_id"); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("turbo-seedance: no task id in create response")
}

func parseTaskGetResponse(raw []byte) (status, videoURL, failMsg string, err error) {
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", "", "", fmt.Errorf("turbo-seedance: parse get task response: %w", err)
	}
	status = firstStatus(v)
	videoURL = findVideoURL(v)
	failMsg = firstErrorMessage(v)
	if d, ok := v["data"].(map[string]any); ok {
		if status == "" {
			status = firstStatus(d)
		}
		if videoURL == "" {
			videoURL = findVideoURL(d)
		}
		if failMsg == "" {
			failMsg = firstErrorMessage(d)
		}
		if out, ok := d["output"].(map[string]any); ok && videoURL == "" {
			videoURL = stringVal(out, "video_url")
		}
	}
	return status, videoURL, failMsg, nil
}

func firstStatus(m map[string]any) string {
	return stringVal(m, "status")
}

func firstErrorMessage(m map[string]any) string {
	if s := stringVal(m, "message"); s != "" {
		return s
	}
	if em, ok := m["error"].(map[string]any); ok {
		if s := stringVal(em, "message"); s != "" {
			return s
		}
		if s := stringVal(em, "code"); s != "" {
			return s
		}
	}
	if s := stringVal(m, "error"); s != "" {
		return s
	}
	return ""
}

func findVideoURL(m map[string]any) string {
	if s := stringVal(m, "video_url"); s != "" {
		return s
	}
	if c, ok := m["content"].(map[string]any); ok {
		if s := stringVal(c, "video_url"); s != "" {
			return s
		}
		if nested, ok := c["video_url"].(map[string]any); ok {
			if s := stringVal(nested, "url"); s != "" {
				return s
			}
		}
	}
	if arr, ok := m["content"].([]any); ok {
		for _, item := range arr {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if s := stringVal(im, "video_url"); s != "" {
				return s
			}
			if nested, ok := im["video_url"].(map[string]any); ok {
				if s := stringVal(nested, "url"); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func stringVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func normalizeStatus(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isSuccessStatus(s string) bool {
	switch s {
	case "succeeded", "success", "completed", "complete", "done":
		return true
	default:
		return false
	}
}

func isFailedStatus(s string) bool {
	switch s {
	case "failed", "error", "cancelled", "canceled", "expired":
		return true
	default:
		return false
	}
}

func (b *backend) prepareParams(ctx context.Context, p *seedance.Params) (*seedance.Params, error) {
	if v := p.GetBool(ParamAssetAutoReview); v != nil && !*v {
		return p, nil
	}
	if !paramsContainUploadableRefs(p) {
		return p, nil
	}
	groupID, err := b.resolveAssetGroupID(ctx, p)
	if err != nil {
		return nil, err
	}
	if groupID == "" {
		return p, nil
	}

	out := seedance.NewParams()
	for k, v := range p.Export() {
		out.Set(k, v)
	}

	if raw, ok := p.Get(ParamContent); ok {
		content := normalizeContent(raw)
		if len(content) == 0 {
			return nil, fmt.Errorf("turbo-seedance: content is empty")
		}
		resolved, err := b.resolveContentMedia(ctx, p, content, groupID)
		if err != nil {
			return nil, err
		}
		out.Set(ParamContent, resolved)
		return out, nil
	}

	if refs := p.GetReferenceImages(seedance.ParamReferenceImages); len(refs) > 0 {
		items := make([]map[string]any, 0, len(refs))
		for _, ref := range refs {
			resolvedURL, err := b.ensureAssetURL(ctx, p, ref.URL, groupID, seedanceassets.AssetTypeImage)
			if err != nil {
				return nil, err
			}
			items = append(items, map[string]any{
				"url":  resolvedURL,
				"role": ref.Role,
			})
		}
		out.Set(seedance.ParamReferenceImages, items)
	}
	if refURLs := p.GetStringSlice(seedance.ParamReferenceImageURLs); len(refURLs) > 0 {
		resolved := make([]string, 0, len(refURLs))
		for _, refURL := range refURLs {
			u, err := b.ensureAssetURL(ctx, p, refURL, groupID, seedanceassets.AssetTypeImage)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, u)
		}
		out.Set(seedance.ParamReferenceImageURLs, resolved)
	}
	if refURLs := p.GetStringSlice(seedance.ParamReferenceVideoURLs); len(refURLs) > 0 {
		resolved := make([]string, 0, len(refURLs))
		for _, refURL := range refURLs {
			u, err := b.ensureAssetURL(ctx, p, refURL, groupID, seedanceassets.AssetTypeVideo)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, u)
		}
		out.Set(seedance.ParamReferenceVideoURLs, resolved)
	}
	return out, nil
}

func paramsContainUploadableRefs(p *seedance.Params) bool {
	if p == nil {
		return false
	}
	if containsExternalURL(p.GetStringSlice(seedance.ParamReferenceImageURLs)) ||
		containsExternalURL(p.GetStringSlice(seedance.ParamReferenceVideoURLs)) {
		return true
	}
	for _, ref := range p.GetReferenceImages(seedance.ParamReferenceImages) {
		if isExternalAssetSource(ref.URL) {
			return true
		}
	}
	if raw, ok := p.Get(ParamContent); ok {
		for _, item := range normalizeContent(raw) {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			typ := strings.TrimSpace(stringVal(m, "type"))
			if typ != "image_url" && typ != "video_url" {
				continue
			}
			payload, _ := m[typ].(map[string]any)
			if isExternalAssetSource(stringVal(payload, "url")) {
				return true
			}
		}
	}
	return false
}

func containsExternalURL(urls []string) bool {
	for _, u := range urls {
		if isExternalAssetSource(u) {
			return true
		}
	}
	return false
}

func isExternalAssetSource(rawURL string) bool {
	u := strings.TrimSpace(rawURL)
	return u != "" && !isAssetURL(u)
}

func isAssetURL(rawURL string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(rawURL)), "asset://")
}

func (b *backend) resolveContentMedia(ctx context.Context, p *seedance.Params, content []any, groupID string) ([]any, error) {
	out := make([]any, 0, len(content))
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		itemType := strings.TrimSpace(stringVal(m, "type"))
		if itemType != "image_url" && itemType != "video_url" {
			out = append(out, item)
			continue
		}
		payload, _ := m[itemType].(map[string]any)
		if payload == nil {
			out = append(out, item)
			continue
		}
		rawURL := stringVal(payload, "url")
		if rawURL == "" || isAssetURL(rawURL) {
			out = append(out, item)
			continue
		}

		assetType := seedanceassets.AssetTypeImage
		if itemType == "video_url" {
			assetType = seedanceassets.AssetTypeVideo
		}
		resolvedURL, err := b.ensureAssetURL(ctx, p, rawURL, groupID, assetType)
		if err != nil {
			return nil, err
		}

		cloned := make(map[string]any, len(m))
		for k, v := range m {
			cloned[k] = v
		}
		mediaURL := make(map[string]any, len(payload))
		for k, v := range payload {
			mediaURL[k] = v
		}
		mediaURL["url"] = resolvedURL
		cloned[itemType] = mediaURL
		out = append(out, cloned)
	}
	return out, nil
}

func (b *backend) resolveAssetGroupID(ctx context.Context, p *seedance.Params) (string, error) {
	groupID := strings.TrimSpace(firstNonEmptyParamOrEnv(p, ParamAssetGroupID, "TURBO_ASSET_GROUP_ID"))
	if groupID != "" {
		return groupID, nil
	}
	groupName := strings.TrimSpace(os.Getenv("TURBO_DEFAULT_ASSET_GROUP_NAME"))
	if groupName == "" {
		groupName = defaultAssetGroupName
	}
	group, err := b.findAssetGroupByName(ctx, groupName)
	if err != nil {
		return "", err
	}
	if group != nil && strings.TrimSpace(group.ID) != "" {
		return group.ID, nil
	}
	created, err := b.createAssetGroup(ctx, groupName, defaultAssetGroupDesc)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func firstNonEmptyParamOrEnv(p *seedance.Params, param string, env string) string {
	if p != nil {
		if v := strings.TrimSpace(p.GetString(param)); v != "" {
			return v
		}
	}
	return strings.TrimSpace(os.Getenv(env))
}

func (b *backend) ensureAssetURL(ctx context.Context, p *seedance.Params, rawURL, groupID, assetType string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	groupID = strings.TrimSpace(groupID)
	if rawURL == "" || groupID == "" || isAssetURL(rawURL) {
		return rawURL, nil
	}
	if cached, ok := b.getCachedAssetURL(groupID, rawURL); ok {
		return cached, nil
	}
	retries := assetReviewRetries(p)
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			b.client.LogDebug("retry asset review url=%q attempt=%d/%d after error: %v", rawURL, attempt+1, retries+1, lastErr)
		}
		asset, err := b.createAsset(ctx, groupID, rawURL, assetType, safeAssetName(rawURL))
		if err == nil {
			var active *seedanceassets.Asset
			active, err = b.awaitAsset(ctx, asset.AssetID, assetPollInterval(p), assetPollAttempts(p))
			if err == nil {
				assetID := strings.TrimSpace(active.ID)
				if assetID == "" {
					assetID = strings.TrimSpace(asset.AssetID)
				}
				if assetID == "" {
					return "", fmt.Errorf("turbo-seedance: asset id is empty after review")
				}
				resolved := "asset://" + assetID
				b.setCachedAssetURL(groupID, rawURL, resolved)
				return resolved, nil
			}
		}
		lastErr = err
		if attempt >= retries || !isRetryableAssetReviewError(err) {
			return "", err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(assetPollInterval(p)):
		}
	}
	return "", lastErr
}

func (b *backend) getCachedAssetURL(groupID, rawURL string) (string, bool) {
	b.assetURLMu.RLock()
	defer b.assetURLMu.RUnlock()
	v, ok := b.assetURLMap[groupID+"\x00"+rawURL]
	return v, ok
}

func (b *backend) setCachedAssetURL(groupID, rawURL, assetURL string) {
	b.assetURLMu.Lock()
	b.assetURLMap[groupID+"\x00"+rawURL] = assetURL
	b.assetURLMu.Unlock()
}

func assetPollInterval(p *seedance.Params) time.Duration {
	if p != nil {
		if ms := p.GetInt(ParamAssetPollInterval); ms != nil && *ms > 0 {
			return time.Duration(*ms) * time.Millisecond
		}
	}
	return 3 * time.Second
}

func assetPollAttempts(p *seedance.Params) int {
	if p != nil {
		if n := p.GetInt(ParamAssetPollAttempts); n != nil && *n > 0 {
			return *n
		}
	}
	return 40
}

func assetReviewRetries(p *seedance.Params) int {
	if p != nil {
		if n := p.GetInt(ParamAssetReviewRetries); n != nil && *n >= 0 {
			return *n
		}
	}
	return 2
}

func isRetryableAssetReviewError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"timeout",
		"temporary",
		"temporarily",
		"connection",
		"request failed",
		"http 429",
		"http 500",
		"http 502",
		"http 503",
		"http 504",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func safeAssetName(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err == nil {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) > 0 {
			if name := strings.TrimSpace(parts[len(parts)-1]); name != "" {
				return name
			}
		}
	}
	return "seedance-reference"
}
