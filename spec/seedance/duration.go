/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package seedance

import (
	"fmt"
	"strings"
)

// ValidateVideoDuration checks the create-task "duration" field (integer seconds) against the
// official Volc Ark matrix for recognized doubao-seedance model ids. See
// https://www.volcengine.com/docs/82379/1520757?lang=zh
//
// Omitting duration lets the API apply its default (currently 5 seconds per doc).
func ValidateVideoDuration(model string, duration int) error {
	if duration == 0 {
		return fmt.Errorf("seedance: duration must not be 0 (omit duration for API default 5s, or use a valid value)")
	}
	m := strings.ToLower(strings.TrimSpace(model))

	switch {
	case strings.Contains(m, "seedance-2-0"):
		// Seedance 2.0 & 2.0 fast: [4, 15] or -1 (smart length).
		if duration == -1 {
			return nil
		}
		if duration < 4 || duration > 15 {
			return fmt.Errorf("seedance: for Seedance 2.0 / 2.0 fast, duration must be in [4, 15] or -1 (smart), got %d", duration)
		}
		return nil

	case strings.Contains(m, "seedance-1-5"):
		// Seedance 1.5 pro: [4, 12] or -1.
		if duration == -1 {
			return nil
		}
		if duration < 4 || duration > 12 {
			return fmt.Errorf("seedance: for Seedance 1.5 pro, duration must be in [4, 12] or -1 (smart), got %d", duration)
		}
		return nil

	case strings.Contains(m, "seedance-1-0"):
		// Seedance 1.0 pro, 1.0 pro fast, 1.0 lite: [2, 12]; -1 not documented.
		if duration == -1 {
			return fmt.Errorf("seedance: for Seedance 1.0 series, duration -1 (smart) is not supported; use [2, 12] seconds")
		}
		if duration < 2 || duration > 12 {
			return fmt.Errorf("seedance: for Seedance 1.0 pro / pro fast / lite, duration must be in [2, 12], got %d", duration)
		}
		return nil

	default:
		// Custom endpoint ids or future model strings: avoid false rejects; API remains authoritative.
		if duration == -1 {
			return nil
		}
		if duration < 1 {
			return fmt.Errorf("seedance: duration must be >= 1 or -1, got %d", duration)
		}
		return nil
	}
}
