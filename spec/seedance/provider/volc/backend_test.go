package volc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
)

// testClientOpts silences curl/HTTP debug logs in unit tests (default client is verbose like Qiniu).
var testClientOpts = []ClientOption{
	WithDebugLog(false),
	WithLogger(log.New(io.Discard, "", 0)),
}

func TestBackendSubmitAndPollSuccess(t *testing.T) {
	var getCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/api/v3/contents/generations/tasks"):
			_, _ = w.Write([]byte(`{"id":"cgt-test-1"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "cgt-test-1"):
			getCount++
			if getCount < 2 {
				_, _ = w.Write([]byte(`{"status":"running"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"succeeded","video_url":"https://example.com/out.mp4"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := NewClient("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	ctx := context.Background()

	p := seedance.NewParams().Set(seedance.ParamPrompt, "hello").Set(seedance.ParamRatio, "16:9")
	resp, err := b.Submit(ctx, xai.Model(seedance.ModelDoubaoSeedance20), p)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID() != "cgt-test-1" {
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

func TestBuildTaskBodySynthesizedContent(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "x").Set(seedance.ParamDuration, 5)
	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	arr, ok := body["content"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("content: %#v", body["content"])
	}
}

func TestBuildTaskBodyDurationMinusOne(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "hi").Set(seedance.ParamDuration, -1)
	body, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err != nil {
		t.Fatal(err)
	}
	if body["duration"] != -1 {
		t.Fatalf("duration: %v", body["duration"])
	}
}

func TestBuildTaskBodyInvalidDurationZero(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "x").Set(seedance.ParamDuration, 0)
	_, err := buildTaskBody("m", p)
	if err == nil {
		t.Fatal("expected error for duration 0")
	}
}

func TestBuildTaskBodyDuration20OutOfRange(t *testing.T) {
	p := seedance.NewParams()
	p.Set(seedance.ParamPrompt, "x").Set(seedance.ParamDuration, 3)
	_, err := buildTaskBody(seedance.ModelDoubaoSeedance20, p)
	if err == nil {
		t.Fatal("expected error for duration 3 on 2.0 model")
	}
}

func TestGetTaskFallbackQueryID(t *testing.T) {
	var pathLog []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathLog = append(pathLog, r.URL.String())
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"id":"tid-2"}`))
			return
		}
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/tid-2") {
			http.Error(w, "nf", http.StatusNotFound)
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "id=tid-2") {
			_, _ = w.Write([]byte(`{"status":"succeeded","video_url":"https://v"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cl := NewClient("k", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	b := newBackend(cl)
	ctx := context.Background()
	resp, err := b.GetTaskStatus(ctx, "tid-2")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Done() || resp.Results().Len() != 1 {
		t.Fatalf("done=%v len=%d", resp.Done(), resp.Results().Len())
	}
	if len(pathLog) < 2 {
		t.Fatalf("expected fallback GET, paths=%v", pathLog)
	}
}

func TestParseCreateTaskIDNested(t *testing.T) {
	id, err := parseCreateTaskID([]byte(`{"data":{"task_id":"abc"}}`))
	if err != nil || id != "abc" {
		t.Fatalf("%q %v", id, err)
	}
}

func TestIsHTTPStatusErr(t *testing.T) {
	err := fmt.Errorf("volc: HTTP 404: x")
	if !isHTTPStatusErr(err, 404) {
		t.Fatal("expected 404")
	}
	if isHTTPStatusErr(err, 500) {
		t.Fatal("not 500")
	}
}
