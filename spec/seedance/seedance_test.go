package seedance

import "testing"

func TestIsVideoModel(t *testing.T) {
	if !IsVideoModel(ModelDoubaoSeedance20) {
		t.Fatal("builtin model should be video")
	}
	if !IsVideoModel("doubao-seedance-2-0-999999") {
		t.Fatal("doubao-seedance-* prefix should be accepted")
	}
	if IsVideoModel("") || IsVideoModel("other-model") {
		t.Fatal("unexpected video model")
	}
}
