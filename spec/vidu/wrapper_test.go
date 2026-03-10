package vidu_test

import (
	"context"
	"testing"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/vidu"
)

type wrappedViduService struct {
	*vidu.Service
}

func (s *wrappedViduService) ViduService() *vidu.Service { return s.Service }

type mockBackend struct{}

func (m *mockBackend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	_ = ctx
	_ = model
	_ = params
	return &vidu.SyncOperationResponse{R: vidu.NewOutputVideos([]string{"https://example.com/out.mp4"})}, nil
}

func (m *mockBackend) GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error) {
	_ = ctx
	_ = taskID
	return nil, xai.ErrNotSupported
}

func TestWrappedServiceVideoOperation(t *testing.T) {
	svc := &wrappedViduService{
		Service: vidu.NewService(&mockBackend{}),
	}

	op, err := svc.Operation(xai.Model(vidu.ModelViduQ1), xai.GenVideo)
	if err != nil {
		t.Fatalf("Operation() error = %v", err)
	}
	op.Params().Set(vidu.ParamPrompt, "a cat running")

	resp, err := op.Call(context.Background(), svc, &vidu.Options{})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if !resp.Done() {
		t.Fatal("expected synchronous response")
	}
	if got := resp.Results().At(0).(*xai.OutputVideo).Video.StgUri(); got != "https://example.com/out.mp4" {
		t.Fatalf("unexpected video result %q", got)
	}
}
