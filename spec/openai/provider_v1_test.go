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
