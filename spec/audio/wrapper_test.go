package audio_test

import (
	"context"
	"testing"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/audio"
)

type wrappedAudioService struct {
	*audio.Service
}

func (s *wrappedAudioService) AudioService() *audio.Service { return s.Service }

type mockASRExecutor struct{}

func (m *mockASRExecutor) Transcribe(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	_ = ctx
	_ = model
	_ = params
	return &audio.SyncOperationResponse{R: audio.NewOutputText("wrapped-transcribe", nil)}, nil
}

type mockTTSExecutor struct{}

func (m *mockTTSExecutor) Synthesize(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	_ = ctx
	_ = model
	_ = params
	return &audio.SyncOperationResponse{R: audio.NewOutputAudio("https://example.com/audio.mp3", "mp3", "1.0")}, nil
}

func TestWrappedServiceTranscribeOperation(t *testing.T) {
	svc := &wrappedAudioService{
		Service: audio.NewService(&mockASRExecutor{}, &mockTTSExecutor{}),
	}

	op, err := svc.Operation(xai.Model(audio.ModelASR), xai.Transcribe)
	if err != nil {
		t.Fatalf("Operation() error = %v", err)
	}
	op.Params().Set(audio.ParamAudio, "https://example.com/in.mp3")

	resp, err := op.Call(context.Background(), svc, nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if !resp.Done() {
		t.Fatal("expected synchronous response")
	}
	if got := resp.Results().At(0).(*xai.OutputText).Text; got != "wrapped-transcribe" {
		t.Fatalf("unexpected text result %q", got)
	}
}

func TestWrappedServiceSynthesizeOperation(t *testing.T) {
	svc := &wrappedAudioService{
		Service: audio.NewService(&mockASRExecutor{}, &mockTTSExecutor{}),
	}

	op, err := svc.Operation(xai.Model(audio.ModelTTSV1), xai.Synthesize)
	if err != nil {
		t.Fatalf("Operation() error = %v", err)
	}
	op.Params().Set(audio.ParamInput, "hello")

	resp, err := op.Call(context.Background(), svc, nil)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if !resp.Done() {
		t.Fatal("expected synchronous response")
	}
	if got := resp.Results().At(0).(*xai.OutputAudio).Audio; got != "https://example.com/audio.mp3" {
		t.Fatalf("unexpected audio result %q", got)
	}
}
