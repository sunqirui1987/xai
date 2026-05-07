package seedanceassets

import (
	"context"
	"testing"
	"time"
)

type stubBackend struct {
	createGroupFn func(context.Context, *CreateAssetGroupRequest) (*AssetGroup, error)
	uploadAssetFn func(context.Context, *UploadAssetRequest) (*AssetUploadResult, error)
	getAssetFn    func(context.Context, string) (*Asset, error)
}

func (b *stubBackend) CreateGroup(ctx context.Context, req *CreateAssetGroupRequest) (*AssetGroup, error) {
	return b.createGroupFn(ctx, req)
}

func (b *stubBackend) UploadAsset(ctx context.Context, req *UploadAssetRequest) (*AssetUploadResult, error) {
	return b.uploadAssetFn(ctx, req)
}

func (b *stubBackend) GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	return b.getAssetFn(ctx, assetID)
}

func TestAwaitAsset(t *testing.T) {
	var calls int
	svc := NewWithBackend(&stubBackend{
		createGroupFn: func(context.Context, *CreateAssetGroupRequest) (*AssetGroup, error) { return nil, nil },
		uploadAssetFn: func(context.Context, *UploadAssetRequest) (*AssetUploadResult, error) { return nil, nil },
		getAssetFn: func(context.Context, string) (*Asset, error) {
			calls++
			if calls < 3 {
				return &Asset{ID: "asset-1", Status: AssetStatusProcessing}, nil
			}
			return &Asset{ID: "asset-1", Status: AssetStatusActive, URL: "https://example.com/a.png"}, nil
		},
	})

	got, err := svc.AwaitAsset(context.Background(), "asset-1", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != AssetStatusActive {
		t.Fatalf("status=%q", got.Status)
	}
	if calls != 3 {
		t.Fatalf("calls=%d", calls)
	}
}
