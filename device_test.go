// Copyright 2026 The Oto Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oto

import "testing"

func TestNewOutputDeviceSelection(t *testing.T) {
	tests := []struct {
		name        string
		options     NewContextOptions
		want        outputDeviceSelection
		wantErr     bool
	}{
		{
			name: "auto",
			options: NewContextOptions{},
			want: outputDeviceSelection{},
		},
		{
			name: "backend only",
			options: NewContextOptions{
				OutputDeviceBackend: DeviceBackendWASAPI,
			},
			want: outputDeviceSelection{backend: DeviceBackendWASAPI},
		},
		{
			name: "full device id",
			options: NewContextOptions{
				OutputDeviceID: outputDeviceID(DeviceBackendPulseAudio, "sink-1"),
			},
			want: outputDeviceSelection{backend: DeviceBackendPulseAudio, deviceID: "sink-1", explicit: true},
		},
		{
			name: "native id with explicit backend",
			options: NewContextOptions{
				OutputDeviceID:      "native-id",
				OutputDeviceBackend: DeviceBackendCoreAudio,
			},
			want: outputDeviceSelection{backend: DeviceBackendCoreAudio, deviceID: "native-id", explicit: true},
		},
		{
			name: "mismatched backend",
			options: NewContextOptions{
				OutputDeviceID:      outputDeviceID(DeviceBackendWASAPI, "device"),
				OutputDeviceBackend: DeviceBackendWinMM,
			},
			wantErr: true,
		},
		{
			name: "invalid device id without backend",
			options: NewContextOptions{
				OutputDeviceID: "device",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newOutputDeviceSelection(&tt.options)
			if tt.wantErr {
				if err == nil {
					t.Fatal("newOutputDeviceSelection() succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("newOutputDeviceSelection() failed: %v", err)
			}
			if got != tt.want {
				t.Fatalf("newOutputDeviceSelection() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEnsureDefaultOnlyOutputSelection(t *testing.T) {
	selection := outputDeviceSelection{
		backend:  DeviceBackendWebAudio,
		deviceID: "default",
		explicit: true,
	}
	if err := ensureDefaultOnlyOutputSelection(selection, DeviceBackendWebAudio); err != nil {
		t.Fatalf("ensureDefaultOnlyOutputSelection() failed for default device: %v", err)
	}
}