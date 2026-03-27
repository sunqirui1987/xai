package openai

import (
	"encoding/json"
	"testing"

	xai "github.com/goplus/xai/spec"
)

func mustNormalizeJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal normalized json: %v", err)
	}
	out, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("remarshal normalized json: %v", err)
	}
	return string(out)
}

func defineToolWithSchema(t *testing.T, svc *Service, name string, schema json.RawMessage) {
	t.Helper()
	type withParams interface {
		Parameters(json.RawMessage) xai.Tool
	}
	ref, ok := svc.ToolDef(name).Description("tool").(withParams)
	if !ok {
		t.Fatalf("tool %q does not support Parameters()", name)
	}
	ref.Parameters(schema)
}

func TestToolParametersPropagateToV1BuildParams(t *testing.T) {
	svc := newService(newV1Provider(nil, "", ""), nil)
	schema := json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`)
	defineToolWithSchema(t, svc, "get_weather", schema)

	req := buildParams(svc.Params().
		Model(xai.Model("gpt-test")).
		Messages(svc.UserMsg().Text("weather")).
		Tools(svc.Tool("get_weather")))
	params := svc.provider.(*v1Provider).buildParams(req)

	got := params.Tools[0].OfFunction.Function.Parameters
	if mustNormalizeJSON(t, got) != mustNormalizeJSON(t, schema) {
		t.Fatalf("v1 parameters = %s want %s", mustNormalizeJSON(t, got), mustNormalizeJSON(t, schema))
	}
}

func TestToolParametersPropagateToV3BuildParams(t *testing.T) {
	svc := newService(newV3Provider(nil), nil)
	schema := json.RawMessage(`{"type":"object","properties":{"n":{"type":"number"}},"required":["n"]}`)
	defineToolWithSchema(t, svc, "sum", schema)

	req := buildParams(svc.Params().
		Model(xai.Model("gpt-test")).
		Messages(svc.UserMsg().Text("sum")).
		Tools(svc.Tool("sum")))
	params := svc.provider.(*v3Provider).buildParams(req)

	got := params.Tools[0].OfFunction.Parameters
	if mustNormalizeJSON(t, got) != mustNormalizeJSON(t, schema) {
		t.Fatalf("v3 parameters = %s want %s", mustNormalizeJSON(t, got), mustNormalizeJSON(t, schema))
	}
}

func TestToolParametersFallbackToDefaultSchemaOnInvalidJSON(t *testing.T) {
	got := toolParametersOrDefault(json.RawMessage(`{"type"`))
	want := map[string]any{"type": "object", "properties": map[string]any{}}
	if mustNormalizeJSON(t, got) != mustNormalizeJSON(t, want) {
		t.Fatalf("fallback parameters = %s want %s", mustNormalizeJSON(t, got), mustNormalizeJSON(t, want))
	}
}
