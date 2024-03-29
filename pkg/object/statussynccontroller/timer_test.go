/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
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

package statussynccontroller

import (
	"reflect"
	"testing"
	"time"
)

func TestNextDuration(t *testing.T) {
	type parameters struct {
		t             time.Time
		paceInSeconds int
	}

	tests := []struct {
		name  string
		want  time.Duration
		input parameters
	}{
		{
			name: "test-1",
			want: 5*time.Second - 555*time.Nanosecond,
			input: parameters{
				t:             time.Unix(0, 555),
				paceInSeconds: 5,
			},
		},
		{
			name: "test-2",
			want: 778 * time.Millisecond,
			input: parameters{
				t:             time.Unix(199994, 222000000),
				paceInSeconds: 5,
			},
		},
		{
			name: "test-3",
			want: 5*time.Second - 333*time.Nanosecond,
			input: parameters{
				t:             time.Unix(38383835, 333),
				paceInSeconds: 5,
			},
		},
		{
			name: "test-4",
			want: 9*time.Second - 888*time.Nanosecond,
			input: parameters{
				t:             time.Unix(99999991, 888),
				paceInSeconds: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextDuration(tt.input.t, tt.input.paceInSeconds)
			if !reflect.DeepEqual(tt.want, got) {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
