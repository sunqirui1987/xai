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

package volc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
)

// pathCreateTask is the Ark API path for creating a contents generation task (Seedance / video).
// See https://www.volcengine.com/docs/82379/1520757?lang=zh
const pathCreateTask = "/api/v3/contents/generations/tasks"

// ErrTaskFailed is returned when Ark reports a terminal failure.
var ErrTaskFailed = errors.New("volc: task failed")

type backend struct {
	client *Client
}

func newBackend(client *Client) *backend {
	return &backend{client: client}
}

// NewBackend returns a seedance.Backend backed by Volc Ark HTTP API.
func NewBackend(client *Client) seedance.Backend {
	return newBackend(client)
}

// Submit creates an async generation task via POST pathCreateTask and returns a polling OperationResponse.
func (b *backend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	p, ok := params.(*seedance.Params)
	if !ok {
		return nil, fmt.Errorf("volc: expected *seedance.Params, got %T", params)
	}
	m := strings.TrimSpace(string(model))
	b.client.LogDebug("Submit GenVideo model=%q", m)

	body, err := buildTaskBody(m, p)
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

// GetTaskStatus GETs task state; returns SyncOperationResponse when succeeded, error when failed, or AsyncOperationResponse while queued/running.
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
			b.client.LogDebug("GetTaskStatus succeeded but missing video_url")
			return nil, fmt.Errorf("%w: succeeded but no video_url in response", ErrTaskFailed)
		}
		b.client.LogDebug("GetTaskStatus task_id=%q done=succeeded", taskID)
		return &seedance.SyncOperationResponse{R: seedance.NewOutputVideos([]string{videoURL})}, nil
	case isFailedStatus(st):
		msg := failMsg
		if msg == "" {
			msg = "unknown error"
		}
		b.client.LogDebug("GetTaskStatus task_id=%q terminal_fail msg=%q", taskID, truncateLog(msg, 256))
		return nil, fmt.Errorf("%w: %s", ErrTaskFailed, msg)
	default:
		b.client.LogDebug("GetTaskStatus task_id=%q still_in_progress (will poll again)", taskID)
		return b.newPollingResponse(taskID), nil
	}
}

func (b *backend) newPollingResponse(taskID string) xai.OperationResponse {
	return seedance.NewAsyncOperationResponse(func(ctx context.Context) (xai.OperationResponse, error) {
		return b.GetTaskStatus(ctx, taskID)
	}, taskID)
}

// getTaskJSON loads a single task. Tries path-style URL first (per Ark docs), then ?id= query (some gateways).
func (b *backend) getTaskJSON(ctx context.Context, taskID string) ([]byte, error) {
	path := pathCreateTask + "/" + url.PathEscape(taskID)
	raw, err := b.client.GetJSON(ctx, path)
	if err == nil {
		return raw, nil
	}
	if !isHTTPStatusErr(err, 404) {
		return nil, err
	}
	b.client.LogDebug("getTaskJSON path GET 404, retry with query id= for task_id=%q", taskID)
	q := pathCreateTask + "?id=" + url.QueryEscape(taskID)
	return b.client.GetJSON(ctx, q)
}

func isHTTPStatusErr(err error, code int) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), fmt.Sprintf("HTTP %d", code))
}

func truncateLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// buildTaskBody maps seedance.Params to the Ark JSON body: model, content[], duration, ratio, generate_audio, watermark.
// When ark_content_json is set, its array replaces the synthesized content from text/reference URLs.
func buildTaskBody(model string, p *seedance.Params) (map[string]any, error) {
	m := strings.TrimSpace(model)
	if m == "" {
		return nil, fmt.Errorf("volc: empty model")
	}

	arkItems, err := p.ArkContentFromJSON()
	if err != nil {
		return nil, fmt.Errorf("volc: ark_content_json: %w", err)
	}

	var content []any
	if len(arkItems) > 0 {
		content = arkItems
	} else {
		text := p.PrimaryText()
		if text == "" {
			return nil, seedance.ErrTextRequired
		}
		content = append(content, map[string]any{
			"type": "text",
			"text": text,
		})
		for _, u := range p.GetStringSlice(seedance.ParamReferenceImageURLs) {
			content = append(content, map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url": u,
				},
				"role": "reference_image",
			})
		}
		for _, u := range p.GetStringSlice(seedance.ParamReferenceVideoURLs) {
			content = append(content, map[string]any{
				"type": "video_url",
				"video_url": map[string]any{
					"url": u,
				},
				"role": "reference_video",
			})
		}
		for _, u := range p.GetStringSlice(seedance.ParamReferenceAudioURLs) {
			content = append(content, map[string]any{
				"type": "audio_url",
				"audio_url": map[string]any{
					"url": u,
				},
				"role": "reference_audio",
			})
		}
	}

	body := map[string]any{
		"model":   m,
		"content": content,
	}
	if d := p.GetInt(seedance.ParamDuration); d != nil {
		if *d < 1 {
			return nil, fmt.Errorf("volc: duration must be >= 1")
		}
		body["duration"] = *d
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

func parseCreateTaskID(raw []byte) (string, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("volc: parse create response: %w", err)
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
	return "", fmt.Errorf("volc: no task id in create response")
}

func parseTaskGetResponse(raw []byte) (status, videoURL, failMsg string, err error) {
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", "", "", fmt.Errorf("volc: parse get task response: %w", err)
	}
	status = firstStatus(v)
	videoURL = findVideoURL(v)
	failMsg = firstErrorMessage(v)
	if d, ok := v["data"].(map[string]any); ok {
		if videoURL == "" {
			videoURL = findVideoURL(d)
		}
		if status == "" {
			status = firstStatus(d)
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
	if s := stringVal(m, "error"); s != "" {
		return s
	}
	if em, ok := m["error"].(map[string]any); ok {
		return stringVal(em, "message")
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
	case "failed", "error", "cancelled", "canceled", "not_found", "notfound":
		return true
	default:
		return false
	}
}
