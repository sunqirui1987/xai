package nodeskai

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

func TestBackendSubmitAndPollSuccess(t *testing.T) {
	var getCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, pathGenerate):
			_, _ = w.Write([]byte(`{"id":"cgt-1","status":"running"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/cgt-1"):
			getCount++
			if getCount < 2 {
				_, _ = w.Write([]byte(`{"status":"running"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"succeeded","content":{"video_url":"https://example.com/out.mp4","duration":5}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	ctx := context.Background()

	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "hello").
		Set(seedance.ParamRatio, "16:9")
	resp, err := b.Submit(ctx, xai.Model(seedance.ModelDoubaoSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID() != "cgt-1" {
		t.Fatalf("task id: %q", resp.TaskID())
	}

	var last xai.OperationResponse = resp
	for !last.Done() {
		last.Sleep()
		last, err = last.Retry(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	if last.Results().Len() != 1 {
		t.Fatalf("len=%d", last.Results().Len())
	}
}

func TestBuildTaskBodyTextToVideo(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "夕阳下的城市街道")
	p.Set("resolution", "720p")
	p.Set(seedance.ParamRatio, "16:9")
	p.Set(seedance.ParamDuration, 5)
	body, err := buildTaskBody("", p)
	if err != nil {
		t.Fatal(err)
	}
	if got := body["model"]; got != seedance.ModelDoubaoSeedance20 {
		t.Fatalf("model=%v", got)
	}
	if got := body["resolution"]; got != "720p" {
		t.Fatalf("resolution=%v", got)
	}
	content, ok := body["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content=%#v", body["content"])
	}
}

func TestBuildTaskBodySupportsRawContent(t *testing.T) {
	p := seedance.NewParams()
	p.Set(ParamContent, []map[string]any{
		{"type": "text", "text": "让角色抬头看向镜头"},
		{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/char.png"}, "role": "character_reference"},
	})
	p.Set(ParamCallbackURL, "https://example.com/callback")
	p.Set(ParamTools, []map[string]any{{"type": "foo"}})

	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	content, ok := body["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content=%#v", body["content"])
	}
	if got := body[ParamCallbackURL]; got != "https://example.com/callback" {
		t.Fatalf("callback_url=%v", got)
	}
}

func TestBuildTaskBodySupportsReferenceInputs(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "从首帧自然过渡到尾帧")
	p.Set(seedance.ParamReferenceImages, []map[string]any{
		{"url": "https://example.com/first.png", "role": "first_frame"},
		{"url": "https://example.com/last.png", "role": "last_frame"},
	})
	p.Set(seedance.ParamReferenceVideoURLs, []string{"https://example.com/ref.mp4"})
	p.Set(seedance.ParamReferenceAudioURLs, []string{"https://example.com/ref.mp3"})
	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	content := body["content"].([]any)
	if len(content) != 5 {
		t.Fatalf("content_len=%d", len(content))
	}
	if got := content[1].(map[string]any)["role"]; got != "first_frame" {
		t.Fatalf("content[1].role=%v", got)
	}
	if got := content[2].(map[string]any)["role"]; got != "last_frame" {
		t.Fatalf("content[2].role=%v", got)
	}
	if got := content[3].(map[string]any)["type"]; got != "video_url" {
		t.Fatalf("content[3].type=%v", got)
	}
	if got := content[4].(map[string]any)["type"]; got != "audio_url" {
		t.Fatalf("content[4].type=%v", got)
	}
}

func TestParseTaskGetResponseFailure(t *testing.T) {
	status, videoURL, failMsg, err := parseTaskGetResponse([]byte(`{"status":"failed","error":{"code":"bad_request","message":"invalid image"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("status=%q", status)
	}
	if videoURL != "" {
		t.Fatalf("videoURL=%q", videoURL)
	}
	if failMsg != "invalid image" {
		t.Fatalf("failMsg=%q", failMsg)
	}
}

func TestSubmitUploadsReferenceImagesThroughAssets(t *testing.T) {
	var uploaded bool
	var awaited bool
	var submittedURL string
	var createdGroup bool

	t.Setenv("NODESKAI_CLIENT_ID", "ndapp_asset")
	t.Setenv("NODESKAI_CLIENT_SECRET", "asset_secret")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/source.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-bytes"))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/oauth/token":
			_, _ = w.Write([]byte(`{"access_token":"asset-token","expires_in":3600}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/digital-assets/groups":
			_, _ = w.Write([]byte(`{"success":true,"data":{"Items":[],"TotalCount":0,"PageNumber":1,"PageSize":20}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/digital-assets/groups/create":
			createdGroup = true
			_, _ = w.Write([]byte(`{"success":true,"data":{"Id":"grp_test_123","Name":"默认素材组","Description":"Seedance 2.0 默认素材组","Status":"Active","AssetCount":0,"CreateTime":"2026-04-11T10:00:00Z"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/digital-assets/upload":
			uploaded = true
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				t.Fatal(err)
			}
			if got := r.FormValue("group_id"); got != "grp_test_123" {
				t.Fatalf("group_id=%q", got)
			}
			_, _ = w.Write([]byte(`{"success":true,"asset_id":"asset_789xyz","asset_type":"Image","status":"Processing","tos_url":"https://tos.example.com/asset.png","file_name":"source.png","file_size":9}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/digital-assets/asset_789xyz":
			awaited = true
			_, _ = w.Write([]byte(`{"success":true,"data":{"Id":"asset_789xyz","GroupId":"grp_test_123","Name":"source.png","AssetType":"Image","Status":"Active","URL":"https://tos.example.com/asset.png","CreateTime":"2026-04-05T14:30:00Z"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, pathGenerate):
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			content, _ := body["content"].([]any)
			if len(content) < 2 {
				t.Fatalf("content=%#v", body["content"])
			}
			item, _ := content[1].(map[string]any)
			imageURL, _ := item["image_url"].(map[string]any)
			submittedURL, _ = imageURL["url"].(string)
			_, _ = w.Write([]byte(`{"id":"cgt-1","status":"running"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("video-key",
		append([]ClientOption{
			WithBaseURL(srv.URL),
			WithPlatformBaseURL(srv.URL),
			WithOAuthClientCredentials("ndapp_asset", "asset_secret"),
		}, testClientOpts...)...,
	)
	b := newBackend(cl)
	ctx := context.Background()

	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "让图片动起来").
		Set(seedance.ParamReferenceImageURLs, []string{srv.URL + "/source.png"})
	_, err := b.Submit(ctx, xai.Model(seedance.ModelDoubaoSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if !createdGroup {
		t.Fatal("expected default asset group creation")
	}
	if !uploaded {
		t.Fatal("expected reference image upload before submit")
	}
	if !awaited {
		t.Fatal("expected asset await before submit")
	}
	if submittedURL != "asset://asset_789xyz" {
		t.Fatalf("submittedURL=%q", submittedURL)
	}
}
