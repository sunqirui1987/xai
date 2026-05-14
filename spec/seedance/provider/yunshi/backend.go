/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package yunshi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
)

const (
	pathCreateTask     = "/api/v3/contents/generations/tasks"
	pathPushMaterial   = "/api/v3/contents/generations/materials/push-volcengine"
	pathMaterialStatus = "/api/v3/contents/generations/materials/%s/volcengine-status"

	EndpointModelSeedance20     = "ep-20260325195111-lsfzx"
	EndpointModelSeedance20Fast = "ep-20260326172546-nzmk4"
)

// Provider-private params supported by this backend.
const (
	ParamContent              = "content"
	ParamResolution           = "resolution"
	ParamAssetGroupID         = "asset_group_id"
	ParamMaterialAutoPush     = "material_auto_push"
	ParamMaterialPollInterval = "material_poll_interval"
	ParamMaterialPollAttempts = "material_poll_attempts"
)

// ErrTaskFailed is returned when Yunshi reports a terminal task failure.
var ErrTaskFailed = errors.New("yunshi: task failed")

// ErrMaterialFailed is returned when a pushed material reaches a failed state.
var ErrMaterialFailed = errors.New("yunshi: material failed")

type backend struct {
	client *Client
}

func newBackend(client *Client) *backend {
	return &backend{client: client}
}

// NewBackend returns a seedance.Backend backed by Yunshi Cloud CATS HTTP API.
func NewBackend(client *Client) seedance.Backend {
	return newBackend(client)
}

// Submit creates an async generation task and returns a polling OperationResponse.
func (b *backend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	p, ok := params.(*seedance.Params)
	if !ok {
		return nil, fmt.Errorf("yunshi: expected *seedance.Params, got %T", params)
	}
	m := strings.TrimSpace(string(model))
	if m == "" {
		m = seedance.ModelDoubaoSeedance20Fast
	}
	b.client.LogDebug("Submit GenVideo model=%q", m)

	body, err := b.buildTaskBody(ctx, m, p)
	if err != nil {
		b.client.LogDebug("Submit buildTaskBody error: %v", err)
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathCreateTask, body)
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
	b.client.LogDebug("GetTaskStatus task_id=%q", taskID)
	raw, err := b.getTaskJSON(ctx, taskID)
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

func (b *backend) getTaskJSON(ctx context.Context, taskID string) ([]byte, error) {
	path := pathCreateTask + "/" + url.PathEscape(taskID)
	raw, err := b.client.GetJSON(ctx, path)
	if err == nil {
		return raw, nil
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		return nil, err
	}
	return b.client.GetJSON(ctx, pathCreateTask+"?id="+url.QueryEscape(taskID))
}

func (b *backend) buildTaskBody(ctx context.Context, model string, p *seedance.Params) (map[string]any, error) {
	apiModel := normalizeYunshiModel(model)
	content, err := b.buildContent(ctx, p)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"model":   apiModel,
		"content": content,
	}
	if d := p.GetInt(seedance.ParamDuration); d != nil {
		if err := seedance.ValidateVideoDuration(model, *d); err != nil {
			return nil, err
		}
		body["duration"] = *d
	}
	if resolution := p.GetString(ParamResolution); resolution != "" {
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

func normalizeYunshiModel(model string) string {
	m := strings.TrimSpace(model)
	switch strings.ToLower(m) {
	case "":
		return EndpointModelSeedance20Fast
	case seedance.ModelDoubaoSeedance20:
		return EndpointModelSeedance20
	case seedance.ModelDoubaoSeedance20Fast:
		return EndpointModelSeedance20Fast
	case EndpointModelSeedance20, EndpointModelSeedance20Fast:
		return m
	default:
		return m
	}
}

func (b *backend) buildContent(ctx context.Context, p *seedance.Params) ([]any, error) {
	if raw, ok := p.Get(ParamContent); ok {
		content := normalizeContent(raw)
		if len(content) == 0 {
			return nil, fmt.Errorf("yunshi: content is empty")
		}
		return content, nil
	}

	text := p.PrimaryText()
	if text == "" {
		return nil, seedance.ErrTextRequired
	}
	content := []any{map[string]any{"type": "text", "text": text}}

	explicitRefs := p.GetReferenceImages(seedance.ParamReferenceImages)
	if len(explicitRefs) > 0 {
		for _, ref := range explicitRefs {
			role := strings.TrimSpace(ref.Role)
			if role == "" {
				role = "reference_image"
			}
			u, err := b.prepareMaterialURL(ctx, p, ref.URL, "image", "参考图")
			if err != nil {
				return nil, err
			}
			content = append(content, mediaURLContent("image_url", "image_url", u, role))
		}
	} else {
		for _, refURL := range p.GetStringSlice(seedance.ParamReferenceImageURLs) {
			u, err := b.prepareMaterialURL(ctx, p, refURL, "image", "参考图")
			if err != nil {
				return nil, err
			}
			content = append(content, mediaURLContent("image_url", "image_url", u, "reference_image"))
		}
	}
	for _, refURL := range p.GetStringSlice(seedance.ParamReferenceVideoURLs) {
		u, err := b.prepareMaterialURL(ctx, p, refURL, "video", "参考视频")
		if err != nil {
			return nil, err
		}
		content = append(content, mediaURLContent("video_url", "video_url", u, "reference_video"))
	}
	for _, refURL := range p.GetStringSlice(seedance.ParamReferenceAudioURLs) {
		u, err := b.prepareMaterialURL(ctx, p, refURL, "audio", "参考音频")
		if err != nil {
			return nil, err
		}
		content = append(content, mediaURLContent("audio_url", "audio_url", u, "reference_audio"))
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
		field:  map[string]any{"url": u},
	}
	if strings.TrimSpace(role) != "" {
		item["role"] = role
	}
	return item
}

func (b *backend) prepareMaterialURL(ctx context.Context, p *seedance.Params, rawURL, typ, name string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "asset://") {
		return rawURL, nil
	}
	groupID := defaultMaterialGroupID(p)
	if v := p.GetBool(ParamMaterialAutoPush); v != nil && !*v {
		return rawURL, nil
	}
	assetID, err := b.PushMaterial(ctx, &PushMaterialRequest{
		URL:     rawURL,
		Type:    typ,
		Name:    name,
		GroupID: groupID,
	})
	if err != nil {
		return "", err
	}
	if err := b.AwaitMaterialActive(ctx, assetID, materialPollInterval(p), materialPollAttempts(p)); err != nil {
		return "", err
	}
	return "asset://" + assetID, nil
}

func defaultMaterialGroupID(p *seedance.Params) string {
	if groupID := p.GetString(ParamAssetGroupID); groupID != "" {
		return groupID
	}
	return strings.TrimSpace(os.Getenv("YUNSHI_GROUP_ID"))
}

func materialPollInterval(p *seedance.Params) time.Duration {
	if ms := p.GetInt(ParamMaterialPollInterval); ms != nil && *ms > 0 {
		return time.Duration(*ms) * time.Millisecond
	}
	return 3 * time.Second
}

func materialPollAttempts(p *seedance.Params) int {
	if n := p.GetInt(ParamMaterialPollAttempts); n != nil && *n > 0 {
		return *n
	}
	return 40
}

// PushMaterialRequest is the request body for push-volcengine.
type PushMaterialRequest struct {
	URL     string `json:"url"`
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	GroupID string `json:"group_id"`
}

// PushMaterial pushes a remote material URL and returns the created asset id.
func (b *backend) PushMaterial(ctx context.Context, req *PushMaterialRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("yunshi: nil material request")
	}
	body := map[string]any{
		"url":  strings.TrimSpace(req.URL),
		"type": strings.TrimSpace(req.Type),
	}
	if body["url"] == "" || body["type"] == "" {
		return "", fmt.Errorf("yunshi: material url and type are required")
	}
	if strings.TrimSpace(req.Name) != "" {
		body["name"] = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.GroupID) != "" {
		body["group_id"] = strings.TrimSpace(req.GroupID)
	}
	raw, err := b.client.PostJSON(ctx, pathPushMaterial, body)
	if err != nil {
		return "", err
	}
	assetID, err := parseAssetID(raw)
	if err != nil {
		return "", err
	}
	return assetID, nil
}

// MaterialStatus returns the current Volcengine material status and optional error.
func (b *backend) MaterialStatus(ctx context.Context, assetID string) (string, string, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return "", "", fmt.Errorf("yunshi: asset id is required")
	}
	raw, err := b.client.GetJSON(ctx, fmt.Sprintf(pathMaterialStatus, url.PathEscape(assetID)))
	if err != nil {
		return "", "", err
	}
	return parseMaterialStatus(raw)
}

// AwaitMaterialActive polls material status until it becomes Active.
func (b *backend) AwaitMaterialActive(ctx context.Context, assetID string, interval time.Duration, attempts int) error {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	if attempts <= 0 {
		attempts = 40
	}
	for i := 0; i < attempts; i++ {
		status, msg, err := b.MaterialStatus(ctx, assetID)
		if err != nil {
			return err
		}
		st := normalizeStatus(status)
		if st == "active" || isSuccessStatus(st) {
			return nil
		}
		if isFailedStatus(st) {
			if msg == "" {
				msg = "unknown error"
			}
			return fmt.Errorf("%w: %s", ErrMaterialFailed, msg)
		}
		if i == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return fmt.Errorf("yunshi: material %s did not become Active after %d attempts", assetID, attempts)
}

func parseCreateTaskID(raw []byte) (string, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("yunshi: parse create response: %w", err)
	}
	if id := stringVal(v, "id"); id != "" {
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
	return "", fmt.Errorf("yunshi: no task id in create response")
}

func parseTaskGetResponse(raw []byte) (status, videoURL, failMsg string, err error) {
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", "", "", fmt.Errorf("yunshi: parse get task response: %w", err)
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
	}
	return status, videoURL, failMsg, nil
}

func parseAssetID(raw []byte) (string, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("yunshi: parse material response: %w", err)
	}
	if id := stringVal(v, "asset_id"); id != "" {
		return id, nil
	}
	if d, ok := v["data"].(map[string]any); ok {
		if id := stringVal(d, "asset_id"); id != "" {
			return id, nil
		}
		if id := stringVal(d, "id"); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("yunshi: no asset_id in material response")
}

func parseMaterialStatus(raw []byte) (status, msg string, err error) {
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", "", fmt.Errorf("yunshi: parse material status response: %w", err)
	}
	status = firstStatus(v)
	msg = firstErrorMessage(v)
	if d, ok := v["data"].(map[string]any); ok {
		if status == "" {
			status = firstStatus(d)
		}
		if msg == "" {
			msg = firstErrorMessage(d)
		}
	}
	return status, msg, nil
}

func firstStatus(m map[string]any) string {
	return stringVal(m, "status")
}

func firstErrorMessage(m map[string]any) string {
	if s := stringVal(m, "message"); s != "" && s != "操作成功" {
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
	case "failed", "error", "cancelled", "canceled", "expired", "inactive", "rejected":
		return true
	default:
		return false
	}
}
