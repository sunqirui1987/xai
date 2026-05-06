package nodeskai

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
