package apimart

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	xai "github.com/goplus/xai/spec"
)

func TestNewServiceWithErrorRequiresAPIKey(t *testing.T) {
	t.Setenv("APIMART_API_KEY", "")

	_, err := NewServiceWithError("")
	if err == nil || !strings.Contains(err.Error(), "API key is required") {
		t.Fatalf("expected missing API key error, got %v", err)
	}
}

func TestGPTImageGenerateAsyncAndWait(t *testing.T) {
	const (
		apiKey = "apimart-token"
		taskID = "task_123"
	)
	t.Setenv("APIMART_API_KEY", apiKey)

	var postSeen bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Fatalf("unexpected auth header: %q", got)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/images/generations":
			postSeen = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request failed: %v", err)
			}
			if body["model"] != ModelGPTImage2 {
				t.Fatalf("unexpected model: %#v", body["model"])
			}
			if body["prompt"] != "一只橘猫坐在窗台上看夕阳，水彩画风格" {
				t.Fatalf("unexpected prompt: %#v", body["prompt"])
			}
			if body["size"] != "16:9" {
				t.Fatalf("unexpected size: %#v", body["size"])
			}
			if body["resolution"] != "2k" {
				t.Fatalf("unexpected resolution: %#v", body["resolution"])
			}
			if body["n"] != float64(1) {
				t.Fatalf("unexpected n: %#v", body["n"])
			}
			_, _ = w.Write([]byte(`{"code":200,"data":[{"status":"submitted","task_id":"` + taskID + `"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tasks/"+taskID:
			_, _ = w.Write([]byte(`{
				"code":200,
				"data":{
					"id":"` + taskID + `",
					"status":"completed",
					"progress":100,
					"created":1776748674,
					"completed":1776748726,
					"actual_time":52,
					"cost":0.05279,
					"estimated_time":100,
					"result":{
						"images":[
							{
								"url":["https://upload.apimart.ai/f/image/out.png"],
								"expires_at":1776835126
							}
						]
					}
				}
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer ts.Close()

	svc, err := NewServiceWithError("", WithBaseURL(ts.URL+"/v1/"), WithHTTPClient(ts.Client()))
	if err != nil {
		t.Fatalf("NewServiceWithError failed: %v", err)
	}

	op, err := svc.Operation(ModelGPTImage2, xai.GenImage)
	if err != nil {
		t.Fatalf("Operation failed: %v", err)
	}
	op.Params().
		Set("Prompt", "一只橘猫坐在窗台上看夕阳，水彩画风格").
		Set("Size", "16:9").
		Set("Resolution", "2k")

	resp, err := op.Call(context.Background(), svc, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if resp.Done() {
		t.Fatal("expected async response")
	}
	if resp.TaskID() != taskID {
		t.Fatalf("unexpected task id: %q", resp.TaskID())
	}

	results, err := xai.Wait(context.Background(), svc, resp, nil)
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if !postSeen {
		t.Fatal("expected submit request to be issued")
	}
	if results.Len() != 1 {
		t.Fatalf("unexpected results len: %d", results.Len())
	}
	imgOut := results.At(0).(*xai.OutputImage)
	if got := imgOut.URL(); got != "https://upload.apimart.ai/f/image/out.png" {
		t.Fatalf("unexpected image url: %q", got)
	}
}

func TestGPTImageEditUsesImageURLsAndFallback(t *testing.T) {
	const taskID = "task_edit_1"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/generations" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		if body["prompt"] != "把这张照片变成水彩画风格" {
			t.Fatalf("unexpected prompt: %#v", body["prompt"])
		}
		if body["official_fallback"] != true {
			t.Fatalf("unexpected official_fallback: %#v", body["official_fallback"])
		}
		refs, ok := body["image_urls"].([]any)
		if !ok || len(refs) != 2 {
			t.Fatalf("unexpected image_urls: %#v", body["image_urls"])
		}
		if refs[0] != "https://example.com/ref.jpg" {
			t.Fatalf("unexpected image_urls[0]: %#v", refs[0])
		}
		if !strings.HasPrefix(refs[1].(string), "data:image/png;base64,") {
			t.Fatalf("unexpected image_urls[1]: %#v", refs[1])
		}
		_, _ = w.Write([]byte(`{"code":200,"data":[{"status":"submitted","task_id":"` + taskID + `"}]}`))
	}))
	defer ts.Close()

	svc := NewService("token-1", WithBaseURL(ts.URL+"/v1/"), WithHTTPClient(ts.Client()))

	op, err := svc.Operation("apimart/gpt-image-2", xai.EditImage)
	if err != nil {
		t.Fatalf("Operation failed: %v", err)
	}
	op.Params().
		Set("Prompt", "把这张照片变成水彩画风格").
		Set("OfficialFallback", true).
		Set("Images", []any{
			"https://example.com/ref.jpg",
			svc.ImageFromBytes(xai.ImagePNG, []byte("hello")),
		})

	resp, err := op.Call(context.Background(), svc, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if resp.TaskID() != taskID {
		t.Fatalf("unexpected task id: %q", resp.TaskID())
	}
}

func TestValidationAndGetTask(t *testing.T) {
	svc := NewService("token-1")

	testErr := func(t *testing.T, action xai.Action, want string, setup func(xai.Operation)) {
		t.Helper()
		op, err := svc.Operation(ModelGPTImage2, action)
		if err != nil {
			t.Fatalf("Operation failed: %v", err)
		}
		setup(op)
		_, err = op.Call(context.Background(), svc, nil)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error containing %q, got %v", want, err)
		}
	}

	testErr(t, xai.GenImage, "Prompt is required", func(op xai.Operation) {})
	testErr(t, xai.EditImage, "Images is required", func(op xai.Operation) {
		op.Params().Set("Prompt", "hello")
	})
	testErr(t, xai.GenImage, "Resolution", func(op xai.Operation) {
		op.Params().Set("Prompt", "hello").Set("Resolution", "8k")
	})
	testErr(t, xai.GenImage, "4k only supports", func(op xai.Operation) {
		op.Params().Set("Prompt", "hello").Set("Resolution", "4k").Set("Size", "1:1")
	})
	testErr(t, xai.EditImage, "exceeds max 16", func(op xai.Operation) {
		refs := make([]string, 17)
		for i := range refs {
			refs[i] = "https://example.com/" + strconv.Itoa(i) + ".png"
		}
		op.Params().Set("Prompt", "hello").Set("Images", refs)
	})

	const taskID = "task_failed"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/tasks/"+taskID {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"code":200,
			"data":{
				"id":"` + taskID + `",
				"status":"failed",
				"error":{"code":400,"message":"参数错误","type":"invalid_request_error"}
			}
		}`))
	}))
	defer ts.Close()

	svc2 := NewService("token-1", WithBaseURL(ts.URL+"/v1/"), WithHTTPClient(ts.Client()))
	resp, err := svc2.GetTask(context.Background(), ModelGPTImage2, xai.GenImage, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if !resp.Done() {
		t.Fatal("expected failed task to be done")
	}
	errResp, ok := resp.(xai.OperationResponseWithError)
	if !ok {
		t.Fatalf("expected OperationResponseWithError, got %T", resp)
	}
	if gotErr := errResp.GetError(); gotErr == nil || !strings.Contains(gotErr.Error(), "参数错误") {
		t.Fatalf("unexpected task error: %v", gotErr)
	}
}

func TestRegisterUsesURIOverrides(t *testing.T) {
	const (
		overrideKey = "override-token"
		taskID      = "task_override"
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+overrideKey {
			t.Fatalf("unexpected auth header: %q", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/generations" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":200,"data":[{"status":"submitted","task_id":"` + taskID + `"}]}`))
	}))
	defer ts.Close()

	Register("fallback-token")

	svc, err := xai.New(context.Background(), "apimart:key="+overrideKey+"&base="+url.QueryEscape(ts.URL+"/v1/"))
	if err != nil {
		t.Fatalf("xai.New failed: %v", err)
	}
	opSvc, ok := svc.(interface {
		Operation(model xai.Model, action xai.Action) (xai.Operation, error)
	})
	if !ok {
		t.Fatal("service does not implement operation service")
	}
	op, err := opSvc.Operation(ModelGPTImage2, xai.GenImage)
	if err != nil {
		t.Fatalf("Operation failed: %v", err)
	}
	op.Params().Set("Prompt", "hello")
	resp, err := op.Call(context.Background(), svc, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if resp.TaskID() != taskID {
		t.Fatalf("unexpected task id: %q", resp.TaskID())
	}
}

func TestHTTPErrorFormatting(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"请求过于频繁，请稍后再试","type":"rate_limit_error"}}`))
	}))
	defer ts.Close()

	svc := NewService("token-1", WithBaseURL(ts.URL+"/v1/"), WithHTTPClient(ts.Client()))
	op, _ := svc.Operation(ModelGPTImage2, xai.GenImage)
	op.Params().Set("Prompt", "hello")
	_, err := op.Call(context.Background(), svc, nil)
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected formatted http error, got %v", err)
	}
}

func TestValueToImageInputFromBlob(t *testing.T) {
	svc := NewService("token-1")
	raw := valueToImageInput(svc.ImageFromBytes(xai.ImagePNG, []byte("hello")))
	if want := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("hello")); raw != want {
		t.Fatalf("unexpected image payload: %q", raw)
	}
}

func TestUnknownModelReturnsNotFound(t *testing.T) {
	svc := NewService("token-1")
	if _, err := svc.Operation("gpt-4o", xai.GenImage); !errors.Is(err, xai.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCurlAndDebugLogs(t *testing.T) {
	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_log_1"}]}`))
	}))
	defer ts.Close()

	svc := NewService("token-1",
		WithBaseURL(ts.URL+"/v1/"),
		WithHTTPClient(ts.Client()),
		WithLogger(logger),
		WithDebugLog(true),
	)
	op, _ := svc.Operation(ModelGPTImage2, xai.GenImage)
	op.Params().Set("Prompt", "hello").Set("Resolution", "2k")
	_, err := op.Call(context.Background(), svc, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "curl -X POST") {
		t.Fatalf("expected curl log, got: %s", logs)
	}
	if !strings.Contains(logs, "[apimart] response status: 200") {
		t.Fatalf("expected response status log, got: %s", logs)
	}
	if !strings.Contains(logs, `"task_id":"task_log_1"`) {
		t.Fatalf("expected response body log, got: %s", logs)
	}
}
