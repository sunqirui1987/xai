package qiniu

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const testModel = "bytedance/doubao-seedance-2-0-260128"

func testClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	return NewService("test-token",
		WithBaseURL(baseURL),
		WithDebugLog(false),
		WithLogger(log.New(io.Discard, "", 0)),
	).client
}

func TestCreateGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathCreateGroup {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["model"]; got != testModel {
			t.Fatalf("model=%v", got)
		}
		if got := body["type"]; got != "aigc" {
			t.Fatalf("type=%v", got)
		}
		_, _ = w.Write([]byte(`{"qgroupid":"qgroup-uid001-1716100000000000000","type":"aigc","name":"我的虚拟人像分组","description":"用于存放公司形象虚拟人像素材","model":"bytedance/doubao-seedance-2-0-260128","status":"pending","is_default":true,"created_at":1716100000,"updated_at":1716100000}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.CreateGroup(context.Background(), &seedanceassets.CreateAssetGroupRequest{
		Name:        "我的虚拟人像分组",
		Description: "用于存放公司形象虚拟人像素材",
		Model:       testModel,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "qgroup-uid001-1716100000000000000" {
		t.Fatalf("id=%q", got.ID)
	}
	if got.Status != seedanceassets.AssetStatusPending {
		t.Fatalf("status=%q", got.Status)
	}
	if !got.IsDefault {
		t.Fatal("expected default group")
	}
	if got.Model != testModel {
		t.Fatalf("model=%q", got.Model)
	}
}

func TestCreateAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathCreateAsset {
			t.Fatalf("path=%s", r.URL.Path)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(bodyBytes, []byte(`"type":"image"`)) {
			t.Fatalf("body=%s", string(bodyBytes))
		}
		if !bytes.Contains(bodyBytes, []byte(`"group_id":"qgroup-uid001-1716100000000000000"`)) {
			t.Fatalf("body=%s", string(bodyBytes))
		}
		_, _ = w.Write([]byte(`{"qassetid":"qasset-uid001-1716100100000000000","type":"image","name":"年轻男人","model":"bytedance/doubao-seedance-2-0-260128","status":"pending","group_id":"qgroup-uid001-1716100000000000000","created_at":1716100100,"updated_at":1716100100}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.UploadAsset(context.Background(), &seedanceassets.UploadAssetRequest{
		GroupID:   "qgroup-uid001-1716100000000000000",
		Name:      "年轻男人",
		AssetType: seedanceassets.AssetTypeImage,
		URL:       "https://example.com/portrait.jpg",
		Model:     testModel,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AssetID != "qasset-uid001-1716100100000000000" {
		t.Fatalf("asset_id=%q", got.AssetID)
	}
	if got.AssetType != "image" {
		t.Fatalf("type=%q", got.AssetType)
	}
	if got.Status != seedanceassets.AssetStatusPending {
		t.Fatalf("status=%q", got.Status)
	}
	if got.GroupID != "qgroup-uid001-1716100000000000000" {
		t.Fatalf("group_id=%q", got.GroupID)
	}
}

func TestGetAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != pathGetAsset+"qasset-uid001-1716100100000000000" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"qassetid":"qasset-uid001-1716100100000000000","type":"image","name":"年轻男人","model":"bytedance/doubao-seedance-2-0-260128","status":"approved","group_id":"qgroup-uid001-1716100000000000000","created_at":1716100100,"updated_at":1716100200}`))
	}))
	defer srv.Close()

	b := newBackend(testClient(t, srv.URL))
	got, err := b.GetAsset(context.Background(), "qasset-uid001-1716100100000000000")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "qasset-uid001-1716100100000000000" {
		t.Fatalf("id=%q", got.ID)
	}
	if got.Status != seedanceassets.AssetStatusApproved {
		t.Fatalf("status=%q", got.Status)
	}
	if got.Model != testModel {
		t.Fatalf("model=%q", got.Model)
	}
	if got.UpdatedAt != 1716100200 {
		t.Fatalf("updated_at=%d", got.UpdatedAt)
	}
}
