package yunshi

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
		case r.Method == http.MethodPost && r.URL.Path == pathCreateTask:
			_, _ = w.Write([]byte(`{"code":200,"message":"操作成功","data":{"id":"cgt-1","status":"running"}}`))
		case r.Method == http.MethodGet && r.URL.Path == pathCreateTask+"/cgt-1":
			getCount++
			if getCount < 2 {
				_, _ = w.Write([]byte(`{"code":200,"data":{"status":"running"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"cgt-1","status":"succeeded","content":{"video_url":"https://example.com/out.mp4"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	ctx := context.Background()

	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "老人走在大街上").
		Set(seedance.ParamRatio, "16:9")
	resp, err := b.Submit(ctx, xai.Model(seedance.ModelDoubaoSeedance20Fast), p)
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

func TestBuildTaskBody(t *testing.T) {
	cl := NewClient("test-key", testClientOpts...)
	b := newBackend(cl)
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "老人走在大街上")
	p.Set(seedance.ParamReferenceImageURLs, []string{"https://example.com/ref.jpg"})
	p.Set(seedance.ParamGenerateAudio, true)
	p.Set(seedance.ParamWatermark, false)
	p.Set(seedance.ParamDuration, 4)
	p.Set(ParamMaterialAutoPush, false)
	body, err := b.buildTaskBody(context.Background(), seedance.ModelDoubaoSeedance20Fast, p)
	if err != nil {
		t.Fatal(err)
	}
	if got := body["model"]; got != EndpointModelSeedance20Fast {
		t.Fatalf("model=%v", got)
	}
	if got := body["generate_audio"]; got != true {
		t.Fatalf("generate_audio=%v", got)
	}
	if got := body["watermark"]; got != false {
		t.Fatalf("watermark=%v", got)
	}
	content := body["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content=%#v", content)
	}
}

func TestNormalizeYunshiModel(t *testing.T) {
	tests := map[string]string{
		"":                                 EndpointModelSeedance20Fast,
		seedance.ModelDoubaoSeedance20:     EndpointModelSeedance20,
		seedance.ModelDoubaoSeedance20Fast: EndpointModelSeedance20Fast,
		EndpointModelSeedance20:            EndpointModelSeedance20,
		"custom-model":                     "custom-model",
	}
	for in, want := range tests {
		if got := normalizeYunshiModel(in); got != want {
			t.Fatalf("normalizeYunshiModel(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestAutoPushReferenceMaterialWhenGroupIDSet(t *testing.T) {
	var pushed bool
	var submittedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPushMaterial:
			pushed = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["group_id"]; got != "grp-1" {
				t.Fatalf("group_id=%v", got)
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"asset_id":"asset-1","volcengine_status":"Processing"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/contents/generations/materials/asset-1/volcengine-status":
			_, _ = w.Write([]byte(`{"code":200,"data":{"asset_id":"asset-1","status":"Active","error":null}}`))
		case r.Method == http.MethodPost && r.URL.Path == pathCreateTask:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			content, _ := body["content"].([]any)
			item, _ := content[1].(map[string]any)
			imageURL, _ := item["image_url"].(map[string]any)
			submittedURL, _ = imageURL["url"].(string)
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"cgt-1","status":"running"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(seedance.ParamPrompt, "让图片动起来").
		Set(seedance.ParamReferenceImageURLs, []string{"https://example.com/actor.jpg"}).
		Set(ParamAssetGroupID, "grp-1").
		Set(ParamMaterialPollInterval, 1)
	_, err := b.Submit(context.Background(), xai.Model(seedance.ModelDoubaoSeedance20Fast), p)
	if err != nil {
		t.Fatal(err)
	}
	if !pushed {
		t.Fatal("expected material push")
	}
	if submittedURL != "asset://asset-1" {
		t.Fatalf("submittedURL=%q", submittedURL)
	}
}

func TestAutoPushReferenceMaterialUsesEnvGroupID(t *testing.T) {
	t.Setenv("YUNSHI_GROUP_ID", "grp-env")
	var pushed bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPushMaterial:
			pushed = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["group_id"]; got != "grp-env" {
				t.Fatalf("group_id=%v", got)
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"asset_id":"asset-env","volcengine_status":"Processing"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/contents/generations/materials/asset-env/volcengine-status":
			_, _ = w.Write([]byte(`{"code":200,"data":{"asset_id":"asset-env","status":"Active","error":null}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(ParamMaterialPollInterval, 1).(*seedance.Params)
	got, err := b.prepareMaterialURL(context.Background(), p, "https://example.com/actor.jpg", "image", "参考图")
	if err != nil {
		t.Fatal(err)
	}
	if !pushed {
		t.Fatal("expected material push")
	}
	if got != "asset://asset-env" {
		t.Fatalf("url=%q", got)
	}
}

func TestAutoPushReferenceMaterialWithoutGroupID(t *testing.T) {
	var pushed bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathPushMaterial:
			pushed = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body["group_id"]; ok {
				t.Fatalf("unexpected group_id=%v", body["group_id"])
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"asset_id":"asset-default","volcengine_status":"Processing"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/contents/generations/materials/asset-default/volcengine-status":
			_, _ = w.Write([]byte(`{"code":200,"data":{"asset_id":"asset-default","status":"Active","error":null}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	p := seedance.NewParams().
		Set(ParamMaterialPollInterval, 1).(*seedance.Params)
	got, err := b.prepareMaterialURL(context.Background(), p, "https://example.com/actor.jpg", "image", "参考图")
	if err != nil {
		t.Fatal(err)
	}
	if !pushed {
		t.Fatal("expected material push")
	}
	if got != "asset://asset-default" {
		t.Fatalf("url=%q", got)
	}
}

func TestBuildTaskBodySupportsRawContent(t *testing.T) {
	cl := NewClient("test-key", testClientOpts...)
	b := newBackend(cl)
	p := seedance.NewParams()
	p.Set(ParamContent, []map[string]any{
		{"type": "text", "text": "老人走在大街上"},
		{"type": "image_url", "image_url": map[string]any{"url": "asset://asset-1"}, "role": "reference_image"},
	})
	body, err := b.buildTaskBody(context.Background(), seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	content, ok := body["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content=%#v", body["content"])
	}
}

func TestParseFailureMessage(t *testing.T) {
	status, videoURL, failMsg, err := parseTaskGetResponse([]byte(`{"code":200,"data":{"status":"failed","error":{"message":"invalid image"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("status=%q", status)
	}
	if videoURL != "" {
		t.Fatalf("videoURL=%q", videoURL)
	}
	if !strings.Contains(failMsg, "invalid image") {
		t.Fatalf("failMsg=%q", failMsg)
	}
}
