package kling_test

import (
	"context"
	"testing"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"
)

type wrappedKlingService struct {
	*kling.Service
}

func (s *wrappedKlingService) KlingService() *kling.Service { return s.Service }

type mockImageExecutor struct{}

func (m *mockImageExecutor) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	_ = ctx
	_ = model
	_ = params
	return &kling.SyncOperationResponse{R: kling.NewOutputImages([]string{"https://example.com/out.png"})}, nil
}

func (m *mockImageExecutor) GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error) {
	_ = ctx
	_ = taskID
	return nil, xai.ErrNotSupported
}

type mockVideoExecutor struct{}

func (m *mockVideoExecutor) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	_ = ctx
	_ = model
	_ = params
	return &kling.SyncOperationResponse{R: kling.NewOutputVideos([]string{"https://example.com/out.mp4"})}, nil
}

func (m *mockVideoExecutor) GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error) {
	_ = ctx
	_ = taskID
	return nil, xai.ErrNotSupported
}

func TestWrappedServiceImageOperation(t *testing.T) {
	svc := &wrappedKlingService{
		Service: kling.NewService(&mockImageExecutor{}, &mockVideoExecutor{}),
	}

	op, err := svc.Operation(xai.Model("kling-v2-1"), xai.GenImage)
	if err != nil {
		t.Fatalf("Operation() error = %v", err)
	}
	op.Params().Set(kling.ParamPrompt, "a cat")

	resp, err := op.Call(context.Background(), svc, &kling.Options{})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if !resp.Done() {
		t.Fatal("expected synchronous response")
	}
	if got := resp.Results().At(0).(*xai.OutputImage).Image.StgUri(); got != "https://example.com/out.png" {
		t.Fatalf("unexpected image result %q", got)
	}
}

func TestWrappedServiceVideoOperation(t *testing.T) {
	svc := &wrappedKlingService{
		Service: kling.NewService(&mockImageExecutor{}, &mockVideoExecutor{}),
	}

	op, err := svc.Operation(xai.Model("kling-v2-5-turbo"), xai.GenVideo)
	if err != nil {
		t.Fatalf("Operation() error = %v", err)
	}
	op.Params().Set(kling.ParamPrompt, "a cat running")

	resp, err := op.Call(context.Background(), svc, &kling.Options{})
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
