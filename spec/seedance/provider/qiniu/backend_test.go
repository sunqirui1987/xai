package qiniu

import (
	"context"
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
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v3/contents/generations/tasks"):
			_, _ = w.Write([]byte(`{"id":"qvideo-1"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/qvideo-1"):
			getCount++
			if getCount < 2 {
				_, _ = w.Write([]byte(`{"status":"running"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"succeeded","content":{"video_url":"https://example.com/out.mp4"}}`))
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
	if resp.TaskID() != "qvideo-1" {
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
	p.Set(seedance.ParamRatio, "16:9")
	p.Set(seedance.ParamDuration, 5)
	p.Set(seedance.ParamGenerateAudio, true)
	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	if got := body["model"]; got != "bytedance/doubao-seedance-2-0-260128" {
		t.Fatalf("model=%v", got)
	}
	content, ok := body["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content=%#v", body["content"])
	}
}

func TestBuildTaskBodyKeepsExplicitQiniuModel(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "test")
	body, err := buildTaskBody("bytedance/doubao-seedance-2-0-260128", p)
	if err != nil {
		t.Fatal(err)
	}
	if got := body["model"]; got != "bytedance/doubao-seedance-2-0-260128" {
		t.Fatalf("model=%v", got)
	}
}

func TestBuildTaskBodyReferenceImageURLsDefaultToReferenceImage(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "从首帧自然过渡到尾帧")
	p.Set(seedance.ParamReferenceImageURLs, []string{"https://example.com/first.png", "https://example.com/last.png"})
	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	content := body["content"].([]any)
	if len(content) != 3 {
		t.Fatalf("content_len=%d", len(content))
	}
	second := content[1].(map[string]any)
	if second["role"] != "reference_image" {
		t.Fatalf("second role=%v", second["role"])
	}
	third := content[2].(map[string]any)
	if third["role"] != "reference_image" {
		t.Fatalf("third role=%v", third["role"])
	}
}

func TestBuildTaskBodyUsesExplicitReferenceImageRoles(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "从首帧自然过渡到尾帧")
	p.Set(seedance.ParamReferenceImages, []map[string]any{
		{"url": "https://example.com/first.png", "role": "first_frame"},
		{"url": "https://example.com/last.png", "role": "last_frame"},
		{"url": "https://example.com/ref.png"},
	})
	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	content := body["content"].([]any)
	if len(content) != 4 {
		t.Fatalf("content_len=%d", len(content))
	}
	if got := content[1].(map[string]any)["role"]; got != "first_frame" {
		t.Fatalf("content[1].role=%v", got)
	}
	if got := content[2].(map[string]any)["role"]; got != "last_frame" {
		t.Fatalf("content[2].role=%v", got)
	}
	if got := content[3].(map[string]any)["role"]; got != "reference_image" {
		t.Fatalf("content[3].role=%v", got)
	}
}
