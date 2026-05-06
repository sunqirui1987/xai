package openai

import (
	"encoding/json"
	"testing"

	xai "github.com/goplus/xai/spec"
	openai "github.com/openai/openai-go/v3"
)

func mustUnmarshalChunk(t *testing.T, s string) openai.ChatCompletionChunk {
	t.Helper()
	var c openai.ChatCompletionChunk
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		t.Fatalf("unmarshal chunk: %v", err)
	}
	return c
}

func TestBuildV1StreamFinalResponseWithToolCalls(t *testing.T) {
	var acc openai.ChatCompletionAccumulator
	chunks := []openai.ChatCompletionChunk{
		mustUnmarshalChunk(t, `{
			"id":"chatcmpl-stream-tool",
			"choices":[{
				"index":0,
				"delta":{
					"role":"assistant",
					"content":"Checking ",
					"tool_calls":[{
						"index":-1,
						"id":"call_weather_1",
						"type":"function",
						"function":{
							"name":"get_weather",
							"arguments":"{\"city\":"
						}
					}]
				}
			}]
		}`),
		mustUnmarshalChunk(t, `{
			"id":"chatcmpl-stream-tool",
			"choices":[{
				"index":0,
				"delta":{
					"content":"tool",
					"tool_calls":[{
						"index":-1,
						"function":{
							"arguments":"\"Shanghai\"}"
						}
					}]
				},
				"finish_reason":"tool_calls"
			}]
		}`),
	}
	for _, chunk := range chunks {
		if !acc.AddChunk(chunk) {
			t.Fatalf("failed to accumulate chunk: %s", chunk.ID)
		}
	}

	resp := buildV1StreamFinalResponse(&acc.ChatCompletion)
	if resp == nil {
		t.Fatalf("expected final stream response")
	}
	if got, want := resp.StopReason(), xai.PauseTurn; got != want {
		t.Fatalf("StopReason got=%s want=%s", got, want)
	}
	if got, want := resp.Parts(), 2; got != want {
		t.Fatalf("Parts got=%d want=%d", got, want)
	}
	if got, want := resp.Part(0).Text(), "Checking tool"; got != want {
		t.Fatalf("text got=%q want=%q", got, want)
	}
	tool, ok := resp.Part(1).AsToolUse()
	if !ok {
		t.Fatalf("expected tool use part")
	}
	if tool.ID != "call_weather_1" || tool.Name != "get_weather" {
		t.Fatalf("unexpected tool call: %+v", tool)
	}
	if raw, ok := tool.Input.(json.RawMessage); !ok || string(raw) != `{"city":"Shanghai"}` {
		t.Fatalf("unexpected tool input: %#v", tool.Input)
	}
}

func TestBuildV1StreamFinalResponseDefaultsMetadata(t *testing.T) {
	resp := buildV1StreamFinalResponse(&openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{{
					ID:   "call_manual",
					Type: "function",
					Function: openai.ChatCompletionMessageFunctionToolCallFunction{
						Name:      "manual",
						Arguments: `{}`,
					},
				}},
			},
		}},
	})
	if resp == nil {
		t.Fatalf("expected response")
	}
	if got, want := resp.msg.Choices[0].FinishReason, "tool_calls"; got != want {
		t.Fatalf("FinishReason got=%q want=%q", got, want)
	}
	if got, want := string(resp.msg.Choices[0].Message.Role), "assistant"; got != want {
		t.Fatalf("Role got=%q want=%q", got, want)
	}
}

func TestApplyExplicitOptionsToJSONBodyThinkingDisabled(t *testing.T) {
	body := []byte(`{"model":"deepseek/deepseek-v3.2-251201","messages":[{"role":"user","content":"hello"}]}`)
	opts := &options{thinkingSet: true, thinkingEnabled: false}
	got, err := applyExplicitOptionsToJSONBody(body, "deepseek/deepseek-v3.2-251201", opts)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	thinking, ok := payload["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("thinking missing: %s", string(got))
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type=%v body=%s", thinking["type"], string(got))
	}
}

func TestApplyExplicitOptionsToJSONBodyThinkingEnabled(t *testing.T) {
	body := []byte(`{"model":"deepseek/deepseek-v3.2-251201"}`)
	opts := &options{thinkingSet: true, thinkingEnabled: true}
	got, err := applyExplicitOptionsToJSONBody(body, "deepseek/deepseek-v3.2-251201", opts)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	thinking, ok := payload["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("thinking missing: %s", string(got))
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("thinking.type=%v body=%s", thinking["type"], string(got))
	}
}

func TestApplyExplicitOptionsToJSONBodySuppressesThinkingForGemini3(t *testing.T) {
	body := []byte(`{"model":"gemini-3.0-pro"}`)
	opts := &options{thinkingSet: true, thinkingEnabled: true}
	got, err := applyExplicitOptionsToJSONBody(body, "gemini-3.0-pro", opts)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["thinking"]; ok {
		t.Fatalf("thinking should be omitted for gemini-3 models: %s", string(got))
	}
}

func TestSuppressThinkingForModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "gemini-3.0-pro", want: true},
		{model: "qiniu/gemini-3.1-flash", want: true},
		{model: "gemini-2.5-flash", want: false},
		{model: "deepseek/deepseek-v3.2-251201", want: false},
	}
	for _, tt := range tests {
		if got := suppressThinkingForModel(tt.model); got != tt.want {
			t.Fatalf("suppressThinkingForModel(%q)=%v want=%v", tt.model, got, tt.want)
		}
	}
}
