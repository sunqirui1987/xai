package xai

import "testing"

func TestRestrictionAllowedValues(t *testing.T) {
	tests := []struct {
		name string
		in   *Restriction
		want []string
	}{
		{
			name: "string enum",
			in:   &Restriction{Limit: &StringEnum{Values: []string{"a", "b"}}},
			want: []string{"a", "b"},
		},
		{
			name: "int enum",
			in:   &Restriction{Limit: &IntEnum{Values: []int64{4, 6, 8}}},
			want: []string{"4", "6", "8"},
		},
	}

	for _, tc := range tests {
		got := tc.in.AllowedValues()
		if len(got) != len(tc.want) {
			t.Fatalf("%s: AllowedValues() = %v, want %v", tc.name, got, tc.want)
		}
		for i, v := range tc.want {
			if got[i] != v {
				t.Fatalf("%s: AllowedValues() = %v, want %v", tc.name, got, tc.want)
			}
		}
	}
}

func TestRestrictionValidateInt(t *testing.T) {
	r := &Restriction{Limit: &IntEnum{Values: []int64{1, 2, 3}}}
	if err := r.ValidateInt("count", 2); err != nil {
		t.Fatalf("ValidateInt(valid) err = %v", err)
	}
	if err := r.ValidateInt("count", 4); err == nil {
		t.Fatal("ValidateInt(invalid) err = nil, want error")
	}
}
