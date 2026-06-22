package turbo

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
)

var testClientOpts = []ClientOption{
	WithDebugLog(false),
	WithLogger(log.New(io.Discard, "", 0)),
}

func TestDreaminaSeedanceModelRecognized(t *testing.T) {
	if !seedance.IsVideoModel(seedance.ModelDreaminaSeedance20) {
		t.Fatalf("model not recognized: %s", seedance.ModelDreaminaSeedance20)
	}
}

func TestBuildTaskBodyMatchesReferenceVideoShape(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "7-second cinematic fantasy animation").
		Set(seedance.ParamReferenceVideoURLs, []string{"asset://asset-test-ref-video"}).
		Set(seedance.ParamGenerateAudio, true).
		Set(seedance.ParamRatio, "9:16").
		Set(seedance.ParamDuration, 7).
		Set("resolution", "1080p").
		Set(seedance.ParamWatermark, false)

	body, err := buildTaskBody(seedance.ModelDreaminaSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	if got := body["model"]; got != seedance.ModelDreaminaSeedance20 {
		t.Fatalf("model=%v", got)
	}
	if got := body["generate_audio"]; got != true {
		t.Fatalf("generate_audio=%v", got)
	}
	if got := body["watermark"]; got != false {
		t.Fatalf("watermark=%v", got)
	}
	content, ok := body["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content=%#v", body["content"])
	}
	video := content[1].(map[string]any)
	if got := video["type"]; got != "video_url" {
		t.Fatalf("video.type=%v", got)
	}
	if got := video["role"]; got != "reference_video" {
		t.Fatalf("video.role=%v", got)
	}
	videoURL := video["video_url"].(map[string]any)
	if got := videoURL["url"]; got != "asset://asset-test-ref-video" {
		t.Fatalf("video.url=%v", got)
	}
}

func TestBackendSubmitAndPollSuccess(t *testing.T) {
	var getCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathVideoGenerations:
			_, _ = w.Write([]byte(`{"id":"task_KLOXX4zqY0m4dMls6w38DuFXxb1vUHj8","status":"running"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/task_KLOXX4zqY0m4dMls6w38DuFXxb1vUHj8"):
			getCount++
			if getCount < 2 {
				_, _ = w.Write([]byte(`{"status":"running"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"completed","content":{"video_url":"https://example.com/out.mp4"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "hello").
		Set(seedance.ParamRatio, "16:9")
	resp, err := b.Submit(context.Background(), xai.Model(seedance.ModelDreaminaSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.TaskID(); got != "task_KLOXX4zqY0m4dMls6w38DuFXxb1vUHj8" {
		t.Fatalf("task id=%q", got)
	}

	var last xai.OperationResponse = resp
	for !last.Done() {
		last, err = last.Retry(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	if last.Results().Len() != 1 {
		t.Fatalf("len=%d", last.Results().Len())
	}
}

func TestParseTaskGetResponseFailure(t *testing.T) {
	status, videoURL, failMsg, err := parseTaskGetResponse([]byte(`{"status":"failed","error":{"code":"bad_request","message":"invalid video"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("status=%q", status)
	}
	if videoURL != "" {
		t.Fatalf("videoURL=%q", videoURL)
	}
	if failMsg != "invalid video" {
		t.Fatalf("failMsg=%q", failMsg)
	}
}

func TestSubmitUploadsExternalReferenceVideoThroughAssets(t *testing.T) {
	var createdAsset bool
	var submittedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == pathAssetGroups:
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == pathAssetGroups:
			_, _ = w.Write([]byte(`{"id":"group-test","name":"默认素材组"}`))
		case r.Method == http.MethodPost && r.URL.Path == pathAssets:
			createdAsset = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["asset_type"]; got != "Video" {
				t.Fatalf("asset_type=%v", got)
			}
			if got := body["url"]; got != "https://example.com/ref.mp4" {
				t.Fatalf("url=%v", got)
			}
			_, _ = w.Write([]byte(`{"id":"asset-video-1","status":"processing","group_id":"group-test"}`))
		case r.Method == http.MethodGet && r.URL.Path == pathAssets+"/asset-video-1":
			_, _ = w.Write([]byte(`{"id":"asset-video-1","status":"completed","url":"https://cdn.example.com/ref.mp4"}`))
		case r.Method == http.MethodPost && r.URL.Path == pathVideoGenerations:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			content := body["content"].([]any)
			video := content[1].(map[string]any)
			videoURL := video["video_url"].(map[string]any)
			submittedURL, _ = videoURL["url"].(string)
			_, _ = w.Write([]byte(`{"id":"task-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Setenv("TURBO_ASSETS_BASE_URL", srv.URL)
	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "use reference video").
		Set(seedance.ParamReferenceVideoURLs, []string{"https://example.com/ref.mp4"}).
		Set(ParamAssetPollInterval, 1).
		Set(ParamAssetPollAttempts, 1)
	_, err := b.Submit(context.Background(), xai.Model(seedance.ModelDreaminaSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if !createdAsset {
		t.Fatal("expected asset upload")
	}
	if submittedURL != "asset://asset-video-1" {
		t.Fatalf("submittedURL=%q", submittedURL)
	}
}

func TestSubmitSkipsUploadForAssetReference(t *testing.T) {
	var uploaded bool
	var submittedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathAssets:
			uploaded = true
			t.Fatal("unexpected asset upload")
		case r.Method == http.MethodPost && r.URL.Path == pathVideoGenerations:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			content := body["content"].([]any)
			video := content[1].(map[string]any)
			videoURL := video["video_url"].(map[string]any)
			submittedURL, _ = videoURL["url"].(string)
			_, _ = w.Write([]byte(`{"id":"task-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "use reference video").
		Set(seedance.ParamReferenceVideoURLs, []string{"asset://asset-test-ref-video"})
	_, err := b.Submit(context.Background(), xai.Model(seedance.ModelDreaminaSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded {
		t.Fatal("expected existing asset URL to skip upload")
	}
	if submittedURL != "asset://asset-test-ref-video" {
		t.Fatalf("submittedURL=%q", submittedURL)
	}
}

func TestSubmitSkipsUploadWhenAssetAutoReviewDisabled(t *testing.T) {
	var uploaded bool
	var submittedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathAssets:
			uploaded = true
			t.Fatal("unexpected asset upload")
		case r.Method == http.MethodPost && r.URL.Path == pathVideoGenerations:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			content := body["content"].([]any)
			video := content[1].(map[string]any)
			videoURL := video["video_url"].(map[string]any)
			submittedURL, _ = videoURL["url"].(string)
			_, _ = w.Write([]byte(`{"id":"task-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "use reference video").
		Set(seedance.ParamReferenceVideoURLs, []string{"https://example.com/ref.mp4"}).
		Set(ParamAssetAutoReview, false)
	_, err := b.Submit(context.Background(), xai.Model(seedance.ModelDreaminaSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded {
		t.Fatal("expected asset_auto_review=false to skip upload")
	}
	if submittedURL != "https://example.com/ref.mp4" {
		t.Fatalf("submittedURL=%q", submittedURL)
	}
}
