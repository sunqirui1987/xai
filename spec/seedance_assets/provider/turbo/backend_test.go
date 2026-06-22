package turbo

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	seedanceturbo "github.com/goplus/xai/spec/seedance/provider/turbo"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

var testClientOpts = []ClientOption{
	WithDebugLog(false),
	WithLogger(log.New(io.Discard, "", 0)),
}

func TestCreateAndListGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathGroups:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["name"]; got != "demo" {
				t.Fatalf("name=%v", got)
			}
			_, _ = w.Write([]byte(`{"id":"group-1","name":"demo","description":"demo group"}`))
		case r.Method == http.MethodGet && r.URL.Path == pathGroups:
			_, _ = w.Write([]byte(`{"items":[{"group_id":"group-1","name":"demo","description":"demo group","created_at":"2026-06-16T20:00:00+08:00"}],"total":1,"page":1,"size":20}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := NewService("test-key", append([]ClientOption{WithBaseURL(srv.URL)}, testClientOpts...)...)
	ctx := context.Background()
	group, err := svc.CreateGroup(ctx, &seedanceassets.CreateAssetGroupRequest{Name: "demo", Description: "demo group"})
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != "group-1" {
		t.Fatalf("group id=%q", group.ID)
	}
	list, err := svc.ListGroups(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != "group-1" {
		t.Fatalf("list=%#v", list)
	}
}

func TestUploadAndAwaitAsset(t *testing.T) {
	var polled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == pathAssets:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["group_id"]; got != "group-1" {
				t.Fatalf("group_id=%v", got)
			}
			if got := body["asset_type"]; got != "Video" {
				t.Fatalf("asset_type=%v", got)
			}
			if got := body["url"]; got != "https://example.com/ref.mp4" {
				t.Fatalf("url=%v", got)
			}
			_, _ = w.Write([]byte(`{"id":"asset-1","status":"processing","group_id":"group-1","asset_type":"Video"}`))
		case r.Method == http.MethodGet && r.URL.Path == pathAssets+"/asset-1":
			polled = true
			_, _ = w.Write([]byte(`{"id":"asset-1","name":"ref","url":"https://cdn.example.com/ref.mp4","asset_type":"Video","group_id":"group-1","status":"completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := seedanceturbo.NewClient("test-key", append([]seedanceturbo.ClientOption{seedanceturbo.WithBaseURL(srv.URL)}, seedanceturbo.WithDebugLog(false), seedanceturbo.WithLogger(log.New(io.Discard, "", 0)))...)
	svc := seedanceassets.NewWithBackend(NewBackend(client))
	ref, err := svc.UploadAndAwaitAsset(context.Background(), &seedanceassets.UploadAssetRequest{
		GroupID:   "group-1",
		Name:      "ref",
		AssetType: seedanceassets.AssetTypeVideo,
		URL:       "https://example.com/ref.mp4",
	}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !polled {
		t.Fatal("expected status poll")
	}
	if ref.Asset != "asset://asset-1" {
		t.Fatalf("asset ref=%q", ref.Asset)
	}
	if ref.URL != "https://cdn.example.com/ref.mp4" {
		t.Fatalf("url=%q", ref.URL)
	}
}
