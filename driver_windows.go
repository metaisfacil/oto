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

package oto

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/metaisfacil/oto/v3/internal/mux"
)

var errDeviceNotFound = errors.New("oto: device not found")

type context struct {
	sampleRate   int
	channelCount int

	mux *mux.Mux

	wasapiContext *wasapiContext
	winmmContext  *winmmContext
	nullContext   *nullContext

	ready chan struct{}
	err   atomicError
}

func newContext(sampleRate int, channelCount int, format mux.Format, bufferSizeInBytes int, _ string, selection outputDeviceSelection) (*context, chan struct{}, error) {
	ctx := &context{
		sampleRate:   sampleRate,
		channelCount: channelCount,
		mux:          mux.New(sampleRate, channelCount, format),
		ready:        make(chan struct{}),
	}

	// Initializing drivers might take some time. Do this asynchronously.
	go func() {
		defer close(ctx.ready)

		var err0 error
		var err1 error

		switch selection.backend {
		case DeviceBackendAuto, DeviceBackendWASAPI:
			var xc *wasapiContext
			xc, err0 = newWASAPIContext(sampleRate, channelCount, ctx.mux, bufferSizeInBytes, selection)
			if err0 == nil {
				ctx.wasapiContext = xc
				return
			}
			if selection.backend != DeviceBackendAuto {
				ctx.err.TryStore(err0)
				return
			}
		default:
			err0 = fmt.Errorf("oto: output device backend %q is not supported on Windows", selection.backend)
		}

		switch selection.backend {
		case DeviceBackendAuto, DeviceBackendWinMM:
			var wc *winmmContext
			wc, err1 = newWinMMContext(sampleRate, channelCount, ctx.mux, bufferSizeInBytes, selection)
			if err1 == nil {
				ctx.winmmContext = wc
				return
			}
			if selection.backend != DeviceBackendAuto {
				ctx.err.TryStore(err1)
				return
			}
		default:
			err1 = fmt.Errorf("oto: output device backend %q is not supported on Windows", selection.backend)
		}

		if errors.Is(err0, errDeviceNotFound) && errors.Is(err1, errDeviceNotFound) {
			ctx.nullContext = newNullContext(sampleRate, channelCount, ctx.mux)
			return
		}

		ctx.err.TryStore(fmt.Errorf("oto: initialization failed: WASAPI: %v, WinMM: %v", err0, err1))
	}()

	return ctx, ctx.ready, nil
}

func (c *context) Suspend() error {
	if err := c.err.Load(); err != nil {
		return err
	}
	<-c.ready
	if c.wasapiContext != nil {
		return c.wasapiContext.Suspend()
	}
	if c.winmmContext != nil {
		return c.winmmContext.Suspend()
	}
	if c.nullContext != nil {
		return c.nullContext.Suspend()
	}
	return nil
}

func (c *context) Resume() error {
	if err := c.err.Load(); err != nil {
		return err
	}
	<-c.ready
	if c.wasapiContext != nil {
		return c.wasapiContext.Resume()
	}
	if c.winmmContext != nil {
		return c.winmmContext.Resume()
	}
	if c.nullContext != nil {
		return c.nullContext.Resume()
	}
	return nil
}

func (c *context) Err() error {
	if err := c.err.Load(); err != nil {
		return err
	}

	select {
	case <-c.ready:
	default:
		return nil
	}

	if c.wasapiContext != nil {
		return c.wasapiContext.Err()
	}
	if c.winmmContext != nil {
		return c.winmmContext.Err()
	}
	if c.nullContext != nil {
		return c.nullContext.Err()
	}
	return nil
}

func (c *context) Close() error {
	c.err.TryStore(errContextClosed)
	c.mux.Close()

	<-c.ready

	var errs []error
	if c.wasapiContext != nil {
		if err := c.wasapiContext.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.winmmContext != nil {
		if err := c.winmmContext.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.nullContext != nil {
		if err := c.nullContext.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type nullContext struct {
	m         sync.Mutex
	suspended bool
	closed    bool
}

func newNullContext(sampleRate int, channelCount int, mux *mux.Mux) *nullContext {
	c := &nullContext{}
	go c.loop(sampleRate, channelCount, mux)
	return c
}

func (c *nullContext) loop(sampleRate int, channelCount int, mux *mux.Mux) {
	var buf32 [4096]float32
	sleep := time.Duration(float64(time.Second) * float64(len(buf32)) / float64(channelCount) / float64(sampleRate))
	for {
		c.m.Lock()
		suspended := c.suspended
		closed := c.closed
		c.m.Unlock()
		if closed {
			return
		}

		if suspended {
			time.Sleep(time.Second)
			continue
		}

		mux.ReadFloat32s(buf32[:])
		time.Sleep(sleep)
	}
}

func (c *nullContext) Suspend() error {
	c.m.Lock()
	c.suspended = true
	c.m.Unlock()
	return nil
}

func (c *nullContext) Resume() error {
	c.m.Lock()
	c.suspended = false
	c.m.Unlock()
	return nil
}

func (c *nullContext) Close() error {
	c.m.Lock()
	c.closed = true
	c.m.Unlock()
	return nil
}

func (*nullContext) Err() error {
	return nil
}
