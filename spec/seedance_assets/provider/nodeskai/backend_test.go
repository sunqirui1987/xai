package nodeskai

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	seedancenodeskai "github.com/goplus/xai/spec/seedance/provider/nodeskai"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

func testClient(t *testing.T, baseURL string) *seedancenodeskai.Client {
	t.Helper()
	return seedancenodeskai.NewClient("test-token",
		seedancenodeskai.WithBaseURL("https://video.example.com"),
		seedancenodeskai.WithPlatformBaseURL(baseURL),
		seedancenodeskai.WithExternalUserID("ext-user-123"),
		seedancenodeskai.WithDebugLog(false),
		seedancenodeskai.WithLogger(log.New(io.Discard, "", 0)),
	)
}

func TestUploadAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathUploadAsset {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("X-External-User-Id"); got != "ext-user-123" {
			t.Fatalf("external_user_id=%q", got)
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("group_id"); got != "grp_abc123" {
			t.Fatalf("group_id=%q", got)
		}
		file, fh, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if fh.Filename != "portrait.jpg" {
			t.Fatalf("filename=%q", fh.Filename)
		}
		_, _ = w.Write([]byte(`{"success":true,"asset_id":"asset_789xyz","asset_type":"Image","status":"Processing","tos_url":"https://tos.example.com/a.jpg","file_name":"portrait.jpg","file_size":2048576}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.UploadAsset(context.Background(), &seedanceassets.UploadAssetRequest{
		GroupID:  "grp_abc123",
		Name:     "女性正脸-01",
		FileName: "portrait.jpg",
		File:     bytes.NewBufferString("jpg-bytes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AssetID != "asset_789xyz" {
		t.Fatalf("asset_id=%q", got.AssetID)
	}
	if got.Status != seedanceassets.AssetStatusProcessing {
		t.Fatalf("status=%q", got.Status)
	}
}

func TestCreateGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathCreateGroup {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("X-External-User-Id"); got != "ext-user-123" {
			t.Fatalf("external_user_id=%q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(body, []byte(`"name":"默认素材组"`)) {
			t.Fatalf("body=%s", string(body))
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"Id":"grp_default001","Name":"默认素材组","Description":"Seedance 2.0 默认素材组","Status":"Active","AssetCount":0,"CreateTime":"2026-04-11T10:00:00Z"}}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.CreateGroup(context.Background(), &seedanceassets.CreateAssetGroupRequest{
		Name:        "默认素材组",
		Description: "Seedance 2.0 默认素材组",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "grp_default001" {
		t.Fatalf("id=%q", got.ID)
	}
	if got.Status != "Active" {
		t.Fatalf("status=%q", got.Status)
	}
}

func TestListGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathListGroups {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("X-External-User-Id"); got != "ext-user-123" {
			t.Fatalf("external_user_id=%q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"Items":[{"Id":"grp_default001","Name":"默认素材组","Description":"Seedance 2.0 默认素材组","Status":"Active","AssetCount":2,"CreateTime":"2026-04-11T10:00:00Z"}],"TotalCount":1,"PageNumber":1,"PageSize":20}}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.ListGroups(context.Background(), &seedanceassets.ListAssetGroupsRequest{
		Name: "默认素材组",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalCount != 1 {
		t.Fatalf("total=%d", got.TotalCount)
	}
	if len(got.Items) != 1 {
		t.Fatalf("items=%d", len(got.Items))
	}
	if got.Items[0].ID != "grp_default001" {
		t.Fatalf("id=%q", got.Items[0].ID)
	}
}

func TestGetAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathGetAsset+"asset_789xyz" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("X-External-User-Id"); got != "ext-user-123" {
			t.Fatalf("external_user_id=%q", got)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"Id":"asset_789xyz","GroupId":"grp_abc123","Name":"女性正脸-01","AssetType":"Image","Status":"Active","URL":"https://tos.example.com/a.jpg","CreateTime":"2026-04-05T14:30:00Z"}}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.GetAsset(context.Background(), "asset_789xyz")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "asset_789xyz" {
		t.Fatalf("id=%q", got.ID)
	}
	if got.Type != seedanceassets.AssetTypeImage {
		t.Fatalf("type=%q", got.Type)
	}
	if got.URL != "https://tos.example.com/a.jpg" {
		t.Fatalf("url=%q", got.URL)
	}
	if got.Status != seedanceassets.AssetStatusActive {
		t.Fatalf("status=%q", got.Status)
	}
}
