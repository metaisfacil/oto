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

import (
	"fmt"
	"strings"
)

// DeviceBackend identifies the platform audio backend associated with an output device.
type DeviceBackend string

const (
	// DeviceBackendAuto lets Oto choose the platform-default backend.
	DeviceBackendAuto DeviceBackend = ""

	// DeviceBackendWASAPI represents the Windows Audio Session API backend.
	DeviceBackendWASAPI DeviceBackend = "wasapi"

	// DeviceBackendWinMM represents the Windows waveOut/WinMM backend.
	DeviceBackendWinMM DeviceBackend = "winmm"

	// DeviceBackendPulseAudio represents the PulseAudio backend.
	DeviceBackendPulseAudio DeviceBackend = "pulseaudio"

	// DeviceBackendCoreAudio represents the Core Audio output device layer on macOS.
	DeviceBackendCoreAudio DeviceBackend = "coreaudio"

	// DeviceBackendWebAudio represents the Web Audio backend.
	DeviceBackendWebAudio DeviceBackend = "webaudio"

	// DeviceBackendOboe represents the Oboe backend on Android.
	DeviceBackendOboe DeviceBackend = "oboe"

	// DeviceBackendConsole represents console backends that expose only a default output device.
	DeviceBackendConsole DeviceBackend = "console"
)

// OutputDevice describes an audio output device that can be used when creating a Context.
//
// OutputDevice.ID is stable only within the backend that produced it. To reuse a device later,
// store the full ID and pass it back through NewContextOptions.OutputDeviceID.
type OutputDevice struct {
	ID        string
	Name      string
	Backend   DeviceBackend
	IsDefault bool
}

type outputDeviceSelection struct {
	backend  DeviceBackend
	deviceID string
	explicit bool
}

// OutputDevices returns the output devices that the current platform backend can enumerate.
//
// On platforms that do not expose concrete device enumeration, this returns a single default
// device entry.
func OutputDevices() ([]OutputDevice, error) {
	return outputDevices()
}

func newOutputDeviceSelection(options *NewContextOptions) (outputDeviceSelection, error) {
	selection := outputDeviceSelection{
		backend: options.OutputDeviceBackend,
	}

	if selection.backend != DeviceBackendAuto && !isKnownDeviceBackend(selection.backend) {
		return outputDeviceSelection{}, fmt.Errorf("oto: unknown output device backend: %q", selection.backend)
	}

	if options.OutputDeviceID == "" {
		return selection, nil
	}

	backend, nativeID, err := parseOutputDeviceID(options.OutputDeviceID)
	if err != nil {
		if selection.backend == DeviceBackendAuto {
			return outputDeviceSelection{}, err
		}
		backend = selection.backend
		nativeID = options.OutputDeviceID
	}

	if selection.backend != DeviceBackendAuto && selection.backend != backend {
		return outputDeviceSelection{}, fmt.Errorf("oto: output device backend %q does not match output device ID %q", selection.backend, options.OutputDeviceID)
	}

	return outputDeviceSelection{
		backend:  backend,
		deviceID: nativeID,
		explicit: true,
	}, nil
}

func parseOutputDeviceID(deviceID string) (DeviceBackend, string, error) {
	backend, nativeID, ok := strings.Cut(deviceID, ":")
	if !ok || nativeID == "" {
		return DeviceBackendAuto, "", fmt.Errorf("oto: invalid output device ID: %q", deviceID)
	}

	b := DeviceBackend(backend)
	if !isKnownDeviceBackend(b) || b == DeviceBackendAuto {
		return DeviceBackendAuto, "", fmt.Errorf("oto: invalid output device backend in ID: %q", deviceID)
	}

	return b, nativeID, nil
}

func isKnownDeviceBackend(backend DeviceBackend) bool {
	switch backend {
	case DeviceBackendAuto,
		DeviceBackendWASAPI,
		DeviceBackendWinMM,
		DeviceBackendPulseAudio,
		DeviceBackendCoreAudio,
		DeviceBackendWebAudio,
		DeviceBackendOboe,
		DeviceBackendConsole:
		return true
	default:
		return false
	}
}

func outputDeviceID(backend DeviceBackend, nativeID string) string {
	return string(backend) + ":" + nativeID
}

func defaultOutputDevices(backend DeviceBackend, name string) []OutputDevice {
	return []OutputDevice{{
		ID:        outputDeviceID(backend, "default"),
		Name:      name,
		Backend:   backend,
		IsDefault: true,
	}}
}

func ensureDefaultOnlyOutputSelection(selection outputDeviceSelection, backend DeviceBackend) error {
	if selection.backend != DeviceBackendAuto && selection.backend != backend {
		return fmt.Errorf("oto: output device backend %q is not supported on this platform", selection.backend)
	}
	if selection.explicit && selection.deviceID != "default" {
		return fmt.Errorf("oto: selecting a specific output device is not supported on backend %q", backend)
	}
	return nil
}