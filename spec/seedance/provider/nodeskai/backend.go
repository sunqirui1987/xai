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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const pathGenerate = "/v1/video/generate"

// Provider-private params supported by this backend.
const (
	ParamContent               = "content"
	ParamCallbackURL           = "callback_url"
	ParamReturnLastFrame       = "return_last_frame"
	ParamExecutionExpiresAfter = "execution_expires_after"
	ParamServiceTier           = "service_tier"
	ParamTools                 = "tools"
	ParamSafetyIdentifier      = "safety_identifier"
	ParamSeed                  = "seed"
	ParamAssetGroupID          = "asset_group_id"
)

// ErrTaskFailed is returned when NoDesk AI reports a terminal failure.
var ErrTaskFailed = errors.New("nodeskai: task failed")

type backend struct {
	client       *Client
	assetService *seedanceassets.Service
	assetURLMu   sync.RWMutex
	assetURLMap  map[string]string
}

func newBackend(client *Client) *backend {
	return &backend{
		client:       client,
		assetService: newAssetService(newAssetPlatformClient(client)),
		assetURLMap:  make(map[string]string),
	}
}

// NewBackend returns a seedance.Backend backed by the NoDesk AI HTTP API.
func NewBackend(client *Client) seedance.Backend {
	return newBackend(client)
}

// Submit creates an async generation task and returns a polling OperationResponse.
func (b *backend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	p, ok := params.(*seedance.Params)
	if !ok {
		return nil, fmt.Errorf("nodeskai: expected *seedance.Params, got %T", params)
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
	raw, err := b.client.PostJSON(ctx, pathGenerate, body)
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
	raw, err := b.client.GetJSON(ctx, pathGenerate+"/"+taskID)
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
		m = seedance.ModelDoubaoSeedance20
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
	copyStringField(body, p, ParamCallbackURL)
	copyStringField(body, p, ParamServiceTier)
	copyStringField(body, p, ParamSafetyIdentifier)
	copyBoolField(body, p, ParamReturnLastFrame)
	copyIntField(body, p, ParamExecutionExpiresAfter)
	copyIntField(body, p, ParamSeed)
	copyRawField(body, p, ParamTools)
	return body, nil
}

func buildContent(p *seedance.Params) ([]any, error) {
	if raw, ok := p.Get(ParamContent); ok {
		content := normalizeContent(raw)
		if len(content) == 0 {
			return nil, fmt.Errorf("nodeskai: content is empty")
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

func copyStringField(dst map[string]any, p *seedance.Params, key string) {
	if v := p.GetString(key); v != "" {
		dst[key] = v
	}
}

func copyBoolField(dst map[string]any, p *seedance.Params, key string) {
	if v := p.GetBool(key); v != nil {
		dst[key] = *v
	}
}

func copyIntField(dst map[string]any, p *seedance.Params, key string) {
	if v := p.GetInt(key); v != nil {
		dst[key] = *v
	}
}

func copyRawField(dst map[string]any, p *seedance.Params, key string) {
	if v, ok := p.Get(key); ok && v != nil {
		dst[key] = v
	}
}

func parseCreateTaskID(raw []byte) (string, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("nodeskai: parse create response: %w", err)
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
	return "", fmt.Errorf("nodeskai: no task id in create response")
}

func parseTaskGetResponse(raw []byte) (status, videoURL, failMsg string, err error) {
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", "", "", fmt.Errorf("nodeskai: parse get task response: %w", err)
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
