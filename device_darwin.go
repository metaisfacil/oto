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

//go:build darwin && !ios

package oto

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	kAudioObjectSystemObject                 = _AudioObjectID(1)
	kAudioHardwarePropertyDevices            = darwinFourCC("dev#")
	kAudioHardwarePropertyDefaultOutputDevice = darwinFourCC("dOut")
	kAudioDevicePropertyDeviceUID            = darwinFourCC("uid ")
	kAudioDevicePropertyStreams              = darwinFourCC("stm#")
	kAudioObjectPropertyName                 = darwinFourCC("lnam")
	kAudioObjectPropertyScopeGlobal          = darwinFourCC("glob")
	kAudioObjectPropertyScopeOutput          = darwinFourCC("outp")
	kAudioQueuePropertyCurrentDevice         = darwinFourCC("aqcd")
)

const kAudioObjectPropertyElementMain = 0

func outputDevices() ([]OutputDevice, error) {
	if err := initializeAPI(); err != nil {
		return nil, err
	}
	if err := initializeDarwinDeviceAPI(); err != nil {
		return nil, err
	}

	defaultDeviceID, err := defaultDarwinOutputDeviceID()
	if err != nil {
		return nil, err
	}

	address := _AudioObjectPropertyAddress{
		mSelector: kAudioHardwarePropertyDevices,
		mScope:    kAudioObjectPropertyScopeGlobal,
		mElement:  kAudioObjectPropertyElementMain,
	}

	deviceCount, deviceIDs, err := darwinObjectIDs(kAudioObjectSystemObject, address)
	if err != nil {
		return nil, err
	}

	devices := make([]OutputDevice, 0, deviceCount)
	for _, deviceID := range deviceIDs {
		hasOutput, err := darwinDeviceHasOutputStreams(deviceID)
		if err != nil {
			return nil, err
		}
		if !hasOutput {
			continue
		}

		uid, err := darwinStringProperty(deviceID, kAudioDevicePropertyDeviceUID, kAudioObjectPropertyScopeGlobal)
		if err != nil {
			return nil, err
		}
		name, err := darwinStringProperty(deviceID, kAudioObjectPropertyName, kAudioObjectPropertyScopeGlobal)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = uid
		}

		devices = append(devices, OutputDevice{
			ID:        outputDeviceID(DeviceBackendCoreAudio, uid),
			Name:      name,
			Backend:   DeviceBackendCoreAudio,
			IsDefault: deviceID == defaultDeviceID,
		})
	}

	return devices, nil
}

func validateDarwinOutputSelection(selection outputDeviceSelection) error {
	if selection.backend != DeviceBackendAuto && selection.backend != DeviceBackendCoreAudio {
		return fmt.Errorf("oto: output device backend %q is not supported on this platform", selection.backend)
	}
	return nil
}

func setAudioQueueOutputDevice(audioQueue _AudioQueueRef, selection outputDeviceSelection) error {
	if !selection.explicit {
		return nil
	}
	if err := initializeDarwinDeviceAPI(); err != nil {
		return err
	}

	uid, err := darwinCFString(selection.deviceID)
	if err != nil {
		return err
	}
	defer _CFRelease(uintptr(uid))

	if osstatus := _AudioQueueSetProperty(audioQueue, kAudioQueuePropertyCurrentDevice, unsafe.Pointer(&uid), uint32(unsafe.Sizeof(uid))); osstatus != noErr {
		return fmt.Errorf("oto: AudioQueueSetProperty(CurrentDevice) failed: %d", osstatus)
	}
	return nil
}

func defaultDarwinOutputDeviceID() (_AudioObjectID, error) {
	address := _AudioObjectPropertyAddress{
		mSelector: kAudioHardwarePropertyDefaultOutputDevice,
		mScope:    kAudioObjectPropertyScopeGlobal,
		mElement:  kAudioObjectPropertyElementMain,
	}
	var deviceID _AudioObjectID
	if err := darwinGetPropertyData(kAudioObjectSystemObject, address, unsafe.Pointer(&deviceID), uint32(unsafe.Sizeof(deviceID))); err != nil {
		return 0, err
	}
	return deviceID, nil
}

func darwinObjectIDs(objectID _AudioObjectID, address _AudioObjectPropertyAddress) (int, []_AudioObjectID, error) {
	size, err := darwinGetPropertyDataSize(objectID, address)
	if err != nil {
		return 0, nil, err
	}
	if size == 0 {
		return 0, nil, nil
	}

	ids := make([]_AudioObjectID, int(size/uint32(unsafe.Sizeof(_AudioObjectID(0)))))
	if err := darwinGetPropertyData(objectID, address, unsafe.Pointer(&ids[0]), size); err != nil {
		return 0, nil, err
	}
	return len(ids), ids, nil
}

func darwinDeviceHasOutputStreams(deviceID _AudioObjectID) (bool, error) {
	address := _AudioObjectPropertyAddress{
		mSelector: kAudioDevicePropertyStreams,
		mScope:    kAudioObjectPropertyScopeOutput,
		mElement:  kAudioObjectPropertyElementMain,
	}
	size, err := darwinGetPropertyDataSize(deviceID, address)
	if err != nil {
		return false, err
	}
	return size > 0, nil
}

func darwinStringProperty(objectID _AudioObjectID, selector uint32, scope uint32) (string, error) {
	address := _AudioObjectPropertyAddress{
		mSelector: selector,
		mScope:    scope,
		mElement:  kAudioObjectPropertyElementMain,
	}
	var value _CFStringRef
	if err := darwinGetPropertyData(objectID, address, unsafe.Pointer(&value), uint32(unsafe.Sizeof(value))); err != nil {
		return "", err
	}
	if value == 0 {
		return "", nil
	}
	defer _CFRelease(uintptr(value))
	return darwinCFStringToString(value)
}

func darwinGetPropertyDataSize(objectID _AudioObjectID, address _AudioObjectPropertyAddress) (uint32, error) {
	var size uint32
	if osstatus := _AudioObjectGetPropertyDataSize(objectID, &address, 0, nil, &size); osstatus != noErr {
		return 0, fmt.Errorf("oto: AudioObjectGetPropertyDataSize failed: %d", osstatus)
	}
	return size, nil
}

func darwinGetPropertyData(objectID _AudioObjectID, address _AudioObjectPropertyAddress, outData unsafe.Pointer, dataSize uint32) error {
	if dataSize == 0 {
		return nil
	}
	ioDataSize := dataSize
	if osstatus := _AudioObjectGetPropertyData(objectID, &address, 0, nil, &ioDataSize, outData); osstatus != noErr {
		return fmt.Errorf("oto: AudioObjectGetPropertyData failed: %d", osstatus)
	}
	return nil
}

func darwinCFString(s string) (_CFStringRef, error) {
	buf := append([]byte(s), 0)
	value := _CFStringCreateWithCString(0, &buf[0], kCFStringEncodingUTF8)
	if value == 0 {
		return 0, fmt.Errorf("oto: CFStringCreateWithCString failed")
	}
	return value, nil
}

func darwinCFStringToString(value _CFStringRef) (string, error) {
	length := _CFStringGetLength(value)
	if length == 0 {
		return "", nil
	}
	maxSize := _CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8)
	buf := make([]byte, int(maxSize)+1)
	if ok := _CFStringGetCString(value, &buf[0], _CFIndex(len(buf)), kCFStringEncodingUTF8); ok == 0 {
		return "", fmt.Errorf("oto: CFStringGetCString failed")
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}

func darwinFourCC(s string) uint32 {
	if len(s) != 4 {
		panic("oto: invalid fourCC")
	}
	return uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3])
}

func initializeDarwinDeviceAPI() error {
	coreAudio, err := purego.Dlopen("/System/Library/Frameworks/CoreAudio.framework/CoreAudio", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	coreFoundation, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&_AudioObjectGetPropertyDataSize, coreAudio, "AudioObjectGetPropertyDataSize")
	purego.RegisterLibFunc(&_AudioObjectGetPropertyData, coreAudio, "AudioObjectGetPropertyData")
	purego.RegisterLibFunc(&_CFStringCreateWithCString, coreFoundation, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&_CFStringGetLength, coreFoundation, "CFStringGetLength")
	purego.RegisterLibFunc(&_CFStringGetMaximumSizeForEncoding, coreFoundation, "CFStringGetMaximumSizeForEncoding")
	purego.RegisterLibFunc(&_CFStringGetCString, coreFoundation, "CFStringGetCString")
	purego.RegisterLibFunc(&_CFRelease, coreFoundation, "CFRelease")
	return nil
}