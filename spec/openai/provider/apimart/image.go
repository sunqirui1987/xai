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

package apimart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/openai"
	"github.com/goplus/xai/types"
)

const (
	endpointImageGenerate = "images/generations"
	endpointTasks         = "tasks"
	pollInterval          = 5 * time.Second

	ParamResolution       = "Resolution"
	ParamOfficialFallback = "OfficialFallback"
)

var (
	enumImageSize = &xai.StringEnum{
		Values: []string{
			"auto", "1:1", "3:2", "2:3", "4:3", "3:4", "5:4", "4:5",
			"16:9", "9:16", "2:1", "1:2", "21:9", "9:21",
		},
	}
	enumResolution   = &xai.StringEnum{Values: []string{"1k", "2k", "4k"}}
	supported4KSizes = map[string]bool{
		"16:9": true,
		"9:16": true,
		"2:1":  true,
		"1:2":  true,
		"21:9": true,
		"9:21": true,
	}
)

type imageInputSchema struct {
	edit bool
}

func (s imageInputSchema) Fields() []xai.Field {
	fields := []xai.Field{
		{Name: openai.ParamPrompt, Kind: types.String},
		{Name: openai.ParamSize, Kind: types.String},
		{Name: ParamResolution, Kind: types.String},
		{Name: ParamOfficialFallback, Kind: types.Bool},
		{Name: openai.ParamImage, Kind: types.String | types.Image},
		{Name: openai.ParamImages, Kind: types.List},
	}
	if s.edit {
		return fields
	}
	return fields
}

func (s imageInputSchema) Restrict(name string) *xai.Restriction {
	switch name {
	case openai.ParamPrompt:
		return &xai.Restriction{Required: true}
	case openai.ParamSize:
		return &xai.Restriction{Limit: enumImageSize}
	case ParamResolution:
		return &xai.Restriction{Limit: enumResolution}
	case openai.ParamImages:
		if s.edit {
			return &xai.Restriction{Required: true}
		}
	}
	return nil
}

type genImage struct {
	model  string
	params *imageParams
}

func (p *genImage) InputSchema() xai.InputSchema { return imageInputSchema{} }

func (p *genImage) Params() xai.Params {
	if p.params == nil {
		p.params = &imageParams{}
	}
	return p.params
}

func (p *genImage) Call(ctx context.Context, svc xai.Service, opts xai.OptionBuilder) (xai.OperationResponse, error) {
	s, ok := svc.(*Service)
	if !ok {
		return nil, xai.ErrNotSupported
	}
	params := p.Params().(*imageParams)
	if err := params.validate(false); err != nil {
		return nil, err
	}
	return s.submitImageTask(ctx, s.operationBaseURL(opts), p.model, params)
}

type editImage struct {
	model  string
	params *imageParams
}

func (p *editImage) InputSchema() xai.InputSchema { return imageInputSchema{edit: true} }

func (p *editImage) Params() xai.Params {
	if p.params == nil {
		p.params = &imageParams{}
	}
	return p.params
}

func (p *editImage) Call(ctx context.Context, svc xai.Service, opts xai.OptionBuilder) (xai.OperationResponse, error) {
	s, ok := svc.(*Service)
	if !ok {
		return nil, xai.ErrNotSupported
	}
	params := p.Params().(*imageParams)
	if err := params.validate(true); err != nil {
		return nil, err
	}
	return s.submitImageTask(ctx, s.operationBaseURL(opts), p.model, params)
}

type imageParams struct {
	Prompt              string
	Size                string
	Resolution          string
	OfficialFallback    bool
	hasOfficialFallback bool
	Image               string
	Images              []string
}

func (p *imageParams) Set(name string, val any) xai.Params {
	switch name {
	case openai.ParamPrompt:
		p.Prompt = valueToString(val)
	case openai.ParamSize:
		p.Size = valueToString(val)
	case ParamResolution:
		p.Resolution = strings.ToLower(valueToString(val))
	case ParamOfficialFallback:
		p.OfficialFallback, p.hasOfficialFallback = valueToBool(val)
	case openai.ParamImage:
		p.Image = valueToImageInput(val)
	case openai.ParamImages:
		p.Images = valueToImageInputs(val)
	}
	return p
}

func (p *imageParams) validate(edit bool) error {
	if strings.TrimSpace(p.Prompt) == "" {
		return fmt.Errorf("apimart: Prompt is required")
	}
	if err := (&xai.Restriction{Limit: enumImageSize}).ValidateString(openai.ParamSize, p.Size); err != nil {
		return err
	}
	if err := (&xai.Restriction{Limit: enumResolution}).ValidateString(ParamResolution, p.Resolution); err != nil {
		return err
	}
	refs := p.referenceImages()
	if edit && len(refs) == 0 {
		return fmt.Errorf("apimart: Images is required")
	}
	if len(refs) > 16 {
		return fmt.Errorf("apimart: image_urls exceeds max 16")
	}
	if strings.EqualFold(p.Resolution, "4k") && p.Size != "" && p.Size != "auto" && !supported4KSizes[p.Size] {
		return fmt.Errorf("apimart: Resolution 4k only supports %v", ordered4KSizes())
	}
	return nil
}

func (p *imageParams) referenceImages() []string {
	ret := make([]string, 0, len(p.Images)+1)
	if strings.TrimSpace(p.Image) != "" {
		ret = append(ret, strings.TrimSpace(p.Image))
	}
	for _, item := range p.Images {
		if s := strings.TrimSpace(item); s != "" {
			ret = append(ret, s)
		}
	}
	return ret
}

type createTaskEnvelope struct {
	Code  int              `json:"code"`
	Data  []createTaskItem `json:"data"`
	Error *apiError        `json:"error,omitempty"`
}

type createTaskItem struct {
	Status string `json:"status"`
	TaskID string `json:"task_id"`
}

type apiErrorEnvelope struct {
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code    any    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type taskStatusEnvelope struct {
	Code  int        `json:"code"`
	Data  *imageTask `json:"data"`
	Error *apiError  `json:"error,omitempty"`
}

type imageTask struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"`
	Progress      int              `json:"progress"`
	Created       int64            `json:"created"`
	Completed     int64            `json:"completed"`
	ActualTime    int64            `json:"actual_time"`
	Cost          float64          `json:"cost"`
	EstimatedTime int64            `json:"estimated_time"`
	Result        *imageTaskResult `json:"result,omitempty"`
	Error         *apiError        `json:"error,omitempty"`
}

type imageTaskResult struct {
	Images []imageTaskResultImage `json:"images"`
}

type imageTaskResultImage struct {
	URL       []string `json:"url"`
	ExpiresAt int64    `json:"expires_at"`
}

func (s *Service) submitImageTask(ctx context.Context, baseURL, model string, params *imageParams) (xai.OperationResponse, error) {
	body := map[string]any{
		"model":  normalizeModel(xai.Model(model)),
		"prompt": params.Prompt,
		"n":      1,
	}
	if params.Size != "" {
		body["size"] = params.Size
	}
	if params.Resolution != "" {
		body["resolution"] = params.Resolution
	}
	if params.hasOfficialFallback {
		body["official_fallback"] = params.OfficialFallback
	}
	if refs := params.referenceImages(); len(refs) > 0 {
		body["image_urls"] = refs
	}

	var resp createTaskEnvelope
	if err := s.doJSON(ctx, http.MethodPost, baseURL, endpointImageGenerate, body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || strings.TrimSpace(resp.Data[0].TaskID) == "" {
		return nil, fmt.Errorf("apimart: empty task_id in response")
	}
	return &imageResp{
		service:  s,
		baseURL:  normalizeBaseURL(baseURL),
		task:     &imageTask{ID: resp.Data[0].TaskID, Status: resp.Data[0].Status},
		sleepDur: pollInterval,
	}, nil
}

func (s *Service) getTask(ctx context.Context, baseURL, taskID string) (xai.OperationResponse, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("apimart: taskID is required")
	}

	var resp taskStatusEnvelope
	if err := s.doJSON(ctx, http.MethodGet, baseURL, endpointTasks+"/"+url.PathEscape(taskID), nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("apimart: empty task data in response")
	}
	if strings.TrimSpace(resp.Data.ID) == "" {
		resp.Data.ID = taskID
	}
	return &imageResp{
		service:  s,
		baseURL:  normalizeBaseURL(baseURL),
		task:     resp.Data,
		sleepDur: pollInterval,
	}, nil
}

func (s *Service) doJSON(ctx context.Context, method, baseURL, endpoint string, body any, out any) error {
	fullURL := normalizeBaseURL(baseURL) + strings.TrimPrefix(endpoint, "/")
	var (
		requestBody []byte
	)
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = data
	}

	cl := s.httpClient
	if cl == nil {
		cl = http.DefaultClient
	}
	s.logCurl(buildCurlCommand(method, fullURL, s.apiKey, requestBody))

	var lastErr error
	maxAttempts := s.maxRetries + 1
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := s.baseRetryDelay * time.Duration(1<<uint(attempt-1))
			s.logDebug("retry %d/%d after %v", attempt, s.maxRetries, delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(requestBody))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.apiKey))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := cl.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("apimart: request failed: %w", err)
			s.logDebug("network error (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			continue
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("apimart: failed to read response body: %w", readErr)
			continue
		}

		s.logDebug("response status: %d", resp.StatusCode)
		if s.debugLog && len(data) > 0 {
			if s.logger != nil {
				s.logger.Printf("[apimart] response: %s", string(data))
			} else {
				fmt.Printf("[apimart] response: %s\n", string(data))
			}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(data, out); err != nil {
				return fmt.Errorf("apimart: decode response: %w", err)
			}
			return nil
		}

		apiErr := decodeAPIError(resp.StatusCode, data)
		if isRetryableStatus(resp.StatusCode) && attempt < s.maxRetries {
			lastErr = apiErr
			s.logDebug("retryable status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxAttempts)
			continue
		}
		return apiErr
	}
	return lastErr
}

func decodeAPIError(statusCode int, data []byte) error {
	var resp apiErrorEnvelope
	if err := json.Unmarshal(data, &resp); err == nil && resp.Error != nil {
		msg := strings.TrimSpace(resp.Error.Message)
		if msg == "" {
			msg = http.StatusText(statusCode)
		}
		if code := strings.TrimSpace(fmt.Sprint(resp.Error.Code)); code != "" {
			return fmt.Errorf("apimart: [%s] %s", code, msg)
		}
		return fmt.Errorf("apimart: %s", msg)
	}
	return fmt.Errorf("apimart: request failed with status %d", statusCode)
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == 429 || statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
}

func buildCurlCommand(method, fullURL, apiKey string, body []byte) string {
	var cmd bytes.Buffer
	cmd.WriteString("curl -X ")
	cmd.WriteString(method)
	cmd.WriteString(fmt.Sprintf(" -H 'Authorization: Bearer %s'", strings.TrimSpace(apiKey)))
	if len(body) > 0 {
		cmd.WriteString(" -H 'Content-Type: application/json'")
		cmd.WriteString(fmt.Sprintf(" -d '%s'", string(body)))
	}
	cmd.WriteString(fmt.Sprintf(" '%s'", fullURL))
	return cmd.String()
}

type imageResp struct {
	service  *Service
	baseURL  string
	task     *imageTask
	sleepDur time.Duration
}

func (p *imageResp) Done() bool {
	status := strings.ToLower(strings.TrimSpace(p.taskStatus()))
	return status == "completed" || status == "failed"
}

func (p *imageResp) Sleep() {
	if p.sleepDur > 0 {
		time.Sleep(p.sleepDur)
	}
}

func (p *imageResp) Retry(ctx context.Context, svc xai.Service) (xai.OperationResponse, error) {
	if s, ok := svc.(*Service); ok {
		return s.getTask(ctx, p.baseURL, p.TaskID())
	}
	if p.service != nil {
		return p.service.getTask(ctx, p.baseURL, p.TaskID())
	}
	return nil, xai.ErrNotSupported
}

func (p *imageResp) Results() xai.Results {
	return newImageResults(p.task)
}

func (p *imageResp) TaskID() string {
	if p.task == nil {
		return ""
	}
	return strings.TrimSpace(p.task.ID)
}

func (p *imageResp) GetError() error {
	if p == nil || p.task == nil {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(p.task.Status), "failed") {
		return nil
	}
	if p.task.Error != nil && strings.TrimSpace(p.task.Error.Message) != "" {
		if code := strings.TrimSpace(fmt.Sprint(p.task.Error.Code)); code != "" {
			return fmt.Errorf("apimart: [%s] %s", code, strings.TrimSpace(p.task.Error.Message))
		}
		return fmt.Errorf("apimart: %s", strings.TrimSpace(p.task.Error.Message))
	}
	return fmt.Errorf("apimart: image generation failed")
}

func (p *imageResp) taskStatus() string {
	if p == nil || p.task == nil {
		return ""
	}
	return p.task.Status
}

type imageResults struct {
	task  *imageTask
	items []*xai.OutputImage
}

func newImageResults(task *imageTask) *imageResults {
	ret := &imageResults{task: task}
	if task == nil || task.Result == nil {
		return ret
	}
	for _, item := range task.Result.Images {
		for _, rawURL := range item.URL {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				continue
			}
			ret.items = append(ret.items, &xai.OutputImage{
				Image: &image{
					mime: guessOutputImageType(rawURL),
					uri:  rawURL,
				},
			})
		}
	}
	return ret
}

func (p *imageResults) XGo_Attr(name string) any {
	if p.task == nil {
		return nil
	}
	switch name {
	case "ID":
		return p.task.ID
	case "Status":
		return p.task.Status
	case "Progress":
		return p.task.Progress
	case "Created":
		return p.task.Created
	case "Completed":
		return p.task.Completed
	case "ActualTime":
		return p.task.ActualTime
	case "EstimatedTime":
		return p.task.EstimatedTime
	case "Cost":
		return p.task.Cost
	default:
		return nil
	}
}

func (p *imageResults) Len() int {
	return len(p.items)
}

func (p *imageResults) At(i int) xai.Generated {
	n := len(p.items)
	if i < 0 || i >= n {
		panic(fmt.Sprintf("apimart: imageResults.At index out of range [%d] with length %d", i, n))
	}
	return p.items[i]
}

type image struct {
	mime xai.ImageType
	uri  string
}

func (p *image) Type() xai.ImageType { return p.mime }
func (p *image) Blob() xai.BlobData  { return nil }
func (p *image) StgUri() string      { return p.uri }

func normalizeModel(model xai.Model) string {
	switch strings.ToLower(strings.TrimSpace(string(model))) {
	case "apimart/gpt-image-2", "gpt-image-2":
		return ModelGPTImage2
	default:
		return strings.TrimSpace(string(model))
	}
}

func isGPTImageModel(model xai.Model) bool {
	switch strings.ToLower(strings.TrimSpace(string(model))) {
	case "gpt-image-2", "apimart/gpt-image-2":
		return true
	default:
		return false
	}
}

func valueToString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func valueToBool(val any) (bool, bool) {
	switch v := val.(type) {
	case bool:
		return v, true
	case string:
		s := strings.TrimSpace(strings.ToLower(v))
		switch s {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func valueToImageInput(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case xai.Image:
		if u := strings.TrimSpace(v.StgUri()); u != "" {
			return u
		}
		if b := v.Blob(); b != nil {
			return "data:" + string(v.Type()) + ";base64," + b.Base64()
		}
		return ""
	default:
		return valueToString(v)
	}
}

func valueToImageInputs(val any) []string {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case []string:
		ret := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(item); s != "" {
				ret = append(ret, s)
			}
		}
		return ret
	case []xai.Image:
		ret := make([]string, 0, len(v))
		for _, item := range v {
			if s := valueToImageInput(item); s != "" {
				ret = append(ret, s)
			}
		}
		return ret
	case []any:
		ret := make([]string, 0, len(v))
		for _, item := range v {
			if s := valueToImageInput(item); s != "" {
				ret = append(ret, s)
			}
		}
		return ret
	default:
		if s := valueToImageInput(v); s != "" {
			return []string{s}
		}
		return nil
	}
}

func guessOutputImageType(rawURL string) xai.ImageType {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return xai.ImagePNG
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".jpg", ".jpeg":
		return xai.ImageJPEG
	case ".gif":
		return xai.ImageGIF
	case ".webp":
		return xai.ImageWebP
	default:
		return xai.ImagePNG
	}
}

func ordered4KSizes() []string {
	return []string{"16:9", "9:16", "2:1", "1:2", "21:9", "9:21"}
}
