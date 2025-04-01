// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package wsproxy

import (
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"io"
	"time"
)

//go:generate mockery --name Reader --filename reader_mock.go --structname MockReader --srcpkg=io
//go:generate mockery --name Writer --filename writer_mock.go --structname MockWriter --srcpkg=io
func CopyBufferWithIdleTimeout(dst io.Writer, src io.Reader, buf []byte, idleTimeout time.Duration) (written int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}

	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}

	for {
		rCh := make(chan int)
		errCh := make(chan error)

		go func() {
			nr, goErr := src.Read(buf)
			if goErr != nil {
				errCh <- goErr
			}
			rCh <- nr
		}()

		select {
		case nr := <-rCh:
			if nr > 0 {
				nw, ew := dst.Write(buf[0:nr])
				if nw < 0 || nr < nw {
					nw = 0
					if ew == nil {
						ew = errors.NewInvalid("invalid write result")
					}
				}
				written += int64(nw)
				if ew != nil {
					err = ew
					return written, err
				}
				if nr != nw {
					err = errors.NewUnavailable("short buffer error")
					return written, err
				}
			}
		case er := <-errCh:
			if er != io.EOF {
				err = er
			}
			return written, err
		case <-time.After(idleTimeout):
			err = errors.NewTimeout("idle timeout")
			return written, err
		}
	}
}
