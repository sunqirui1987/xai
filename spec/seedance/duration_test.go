/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package seedance

import "testing"

func TestValidateVideoDuration20(t *testing.T) {
	m := ModelDoubaoSeedance20
	if err := ValidateVideoDuration(m, -1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, 4); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, 15); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, 3); err == nil {
		t.Fatal("expected error for duration 3 on 2.0")
	}
	if err := ValidateVideoDuration(m, 16); err == nil {
		t.Fatal("expected error for duration 16 on 2.0")
	}
}

func TestValidateVideoDuration15(t *testing.T) {
	m := "doubao-seedance-1-5-pro-000"
	if err := ValidateVideoDuration(m, -1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, 12); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, 3); err == nil {
		t.Fatal("expected error for duration 3 on 1.5")
	}
}

func TestValidateVideoDuration10(t *testing.T) {
	m := "doubao-seedance-1-0-pro-000"
	if err := ValidateVideoDuration(m, 2); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, 12); err != nil {
		t.Fatal(err)
	}
	if err := ValidateVideoDuration(m, -1); err == nil {
		t.Fatal("expected error for -1 on 1.0")
	}
	if err := ValidateVideoDuration(m, 1); err == nil {
		t.Fatal("expected error for duration 1 on 1.0")
	}
}

func TestValidateVideoDurationZero(t *testing.T) {
	if err := ValidateVideoDuration(ModelDoubaoSeedance20, 0); err == nil {
		t.Fatal("expected error for 0")
	}
}
