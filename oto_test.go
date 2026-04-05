// Copyright 2022 The Oto Authors
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

package oto_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/metaisfacil/oto/v3"
)

func newTestContextOptions() *oto.NewContextOptions {
	return &oto.NewContextOptions{
		SampleRate:   48000,
		ChannelCount: 2,
		Format:       oto.FormatFloat32LE,
	}
}

func newTestContext(t *testing.T) *oto.Context {
	t.Helper()

	ctx, ready, err := oto.NewContext(newTestContextOptions())
	if err != nil {
		t.Fatalf("oto.NewContext() failed: %v", err)
	}
	<-ready
	t.Cleanup(func() {
		if err := ctx.Close(); err != nil {
			t.Errorf("ctx.Close() failed: %v", err)
		}
	})
	return ctx
}

func TestEmptyPlayer(t *testing.T) {
	ctx := newTestContext(t)

	bs := bytes.NewReader(make([]byte, 0))
	p := ctx.NewPlayer(bs)
	p.Play()
	for p.IsPlaying() {
		time.Sleep(time.Millisecond)
	}
}

// Issue #258
func TestSetBufferSize(t *testing.T) {
	ctx := newTestContext(t)

	for i := 0; i < 10; i++ {
		bs := bytes.NewReader(make([]byte, 512))
		p := ctx.NewPlayer(bs)
		p.Play()
		p.SetBufferSize(256)
		for p.IsPlaying() {
			time.Sleep(time.Millisecond)
		}
	}
}

func TestContextCloseAllowsRecreate(t *testing.T) {
	ctx := newTestContext(t)

	if _, _, err := oto.NewContext(newTestContextOptions()); err == nil {
		t.Fatal("oto.NewContext() succeeded while another context is still live")
	}

	if err := ctx.Close(); err != nil {
		t.Fatalf("ctx.Close() failed: %v", err)
	}

	ctx2, ready, err := oto.NewContext(newTestContextOptions())
	if err != nil {
		t.Fatalf("oto.NewContext() after Close failed: %v", err)
	}
	<-ready
	if err := ctx2.Close(); err != nil {
		t.Fatalf("ctx2.Close() failed: %v", err)
	}
}
