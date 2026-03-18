package gemini

import (
	"testing"

	xai "github.com/goplus/xai/spec"
)

func TestVideoSchemaForVeoDurationRestriction(t *testing.T) {
	tests := []struct {
		model string
		want  []string
	}{
		{model: "veo-3.1-generate-preview", want: []string{"4", "6", "8"}},
		{model: "veo-2.0-generate-preview", want: []string{"5", "6", "7", "8"}},
	}

	for _, tc := range tests {
		schema := VideoSchemaFor(tc.model)
		if schema == nil {
			t.Fatalf("%s: VideoSchemaFor returned nil", tc.model)
		}
		r := schema.Restrict(ParamDurationSeconds)
		if r == nil {
			t.Fatalf("%s: Restrict(DurationSeconds) returned nil", tc.model)
		}
		got := r.AllowedValues()
		if len(got) != len(tc.want) {
			t.Fatalf("%s: duration AllowedValues = %v, want %v", tc.model, got, tc.want)
		}
		for i, v := range tc.want {
			if got[i] != v {
				t.Fatalf("%s: duration AllowedValues = %v, want %v", tc.model, got, tc.want)
			}
		}
	}
}

func TestVideoSchemaForVeoNumberOfVideosRestriction(t *testing.T) {
	schema := VideoSchemaFor("veo-3.0-generate-preview")
	if schema == nil {
		t.Fatal("VideoSchemaFor returned nil")
	}
	r := schema.Restrict(ParamNumberOfVideos)
	if r == nil {
		t.Fatal("Restrict(NumberOfVideos) returned nil")
	}
	if got := r.AllowedValues(); len(got) != 4 || got[0] != "1" || got[3] != "4" {
		t.Fatalf("number_of_videos AllowedValues = %v, want [1 2 3 4]", got)
	}
	if err := r.ValidateInt(ParamNumberOfVideos, 3); err != nil {
		t.Fatalf("ValidateInt(valid) err = %v", err)
	}
	if err := r.ValidateInt(ParamNumberOfVideos, 5); err == nil {
		t.Fatal("ValidateInt(invalid) err = nil, want error")
	}
}

func TestVideoSchemaForVeoDurationRestrictionType(t *testing.T) {
	schema := VideoSchemaFor("veo-3.0-generate-preview")
	r := schema.Restrict(ParamDurationSeconds)
	if _, ok := r.Limit.(*xai.IntEnum); !ok {
		t.Fatalf("DurationSeconds limit type = %T, want *xai.IntEnum", r.Limit)
	}
}
