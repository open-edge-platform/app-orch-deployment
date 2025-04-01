// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package wsproxy

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/atomix/atomix/api/errors"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/wsproxy/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	testBuffer = []byte("test forwarding")
)

func TestCopyBufferWithIdleTimeout(t *testing.T) {
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer pipeInWriter.Close()
		select {
		case <-ctx.Done():
			return
		default:
			_, err := pipeInWriter.Write(testBuffer)
			if err != nil {
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, len(testBuffer))
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := pipeOutReader.Read(buf)
				if err != nil {
					return
				}
			}
		}
	}()

	_, err := CopyBufferWithIdleTimeout(pipeOutWriter, pipeInReader, nil, time.Minute)
	assert.NoError(t, err)

	t.Cleanup(func() {
		pipeInReader.Close()
		pipeOutReader.Close()
		pipeOutWriter.Close()
	})
}

func TestCopyBufferWithIdleTimeout_IdleTimeoutRaised(t *testing.T) {
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()

	_, err := CopyBufferWithIdleTimeout(pipeOutWriter, pipeInReader, nil, 5*time.Second)
	assert.Error(t, err)

	t.Cleanup(func() {
		pipeInReader.Close()
		pipeInWriter.Close()
		pipeOutReader.Close()
		pipeOutWriter.Close()
	})
}

// dummyReader is the dummy struct to make io.ReadFrom fail
type dummyReader struct{}

func (r *dummyReader) Read(_ []byte) (n int, err error) {
	return 0, nil
}
func (r *dummyReader) WriteTo(_ io.Writer) (n int64, err error) {
	return 0, errors.NewNotSupported("testing")
}

func TestCopyBufferWithIdleTimeout_FailedWriteTo(t *testing.T) {
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()
	d := &dummyReader{}

	_, err := CopyBufferWithIdleTimeout(pipeOutWriter, d, nil, time.Minute)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "testing")

	t.Cleanup(func() {
		pipeInReader.Close()
		pipeInWriter.Close()
		pipeOutReader.Close()
		pipeOutWriter.Close()
	})
}

// dummyWriter is the dummy struct to make io.WriteTo fail
type dummyWriter struct{}

func (w *dummyWriter) Write(_ []byte) (n int, err error) {
	return 0, nil
}

func (w *dummyWriter) ReadFrom(_ io.Reader) (n int64, err error) {
	return 0, errors.NewNotSupported("testing")
}

func TestCopyBufferWithIdleTimeout_FailedReadFrom(t *testing.T) {
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()
	d := &dummyWriter{}

	_, err := CopyBufferWithIdleTimeout(d, pipeInReader, nil, time.Minute)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "testing")

	t.Cleanup(func() {
		pipeInReader.Close()
		pipeInWriter.Close()
		pipeOutReader.Close()
		pipeOutWriter.Close()
	})
}

func TestCopyBufferWithIdleTimeout_LimitedReaderCase_N0(t *testing.T) {
	pipeInReader, pipeInWriter := io.Pipe()
	pipeOutReader, pipeOutWriter := io.Pipe()
	d := &io.LimitedReader{
		R: pipeInReader,
		N: int64(0),
	}

	_, err := CopyBufferWithIdleTimeout(pipeOutWriter, d, nil, time.Minute)
	assert.NoError(t, err)

	t.Cleanup(func() {
		pipeInReader.Close()
		pipeInWriter.Close()
		pipeOutReader.Close()
		pipeOutWriter.Close()
	})
}

func TestCopyBufferWithIdleTimeout_LimitedReaderCase_N1(t *testing.T) {
	mockReader := mocks.NewMockReader(t)
	mockReader.On("Read", mock.Anything).Return(1, nil)
	mockWriter := mocks.NewMockWriter(t)
	mockWriter.On("Write", mock.Anything).Return(1, nil)

	d := &io.LimitedReader{
		R: mockReader,
		N: int64(1),
	}

	_, err := CopyBufferWithIdleTimeout(mockWriter, d, nil, time.Minute)
	assert.NoError(t, err)

	t.Cleanup(func() {
		n, err := mockReader.Read(nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		n, err = mockWriter.Write(nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
	})
}

func TestCopyBufferWithIdleTimeout_InvalidWriteResult(t *testing.T) {
	mockReader := mocks.NewMockReader(t)
	mockReader.On("Read", mock.Anything).Return(1, nil)

	mockWriter := mocks.NewMockWriter(t)
	mockWriter.On("Write", mock.Anything).Return(-1, nil)

	_, err := CopyBufferWithIdleTimeout(mockWriter, mockReader, nil, time.Minute)
	assert.Error(t, err)

	t.Cleanup(func() {
		n, err := mockReader.Read(nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		n, err = mockWriter.Write(nil)
		assert.NoError(t, err)
		assert.Equal(t, -1, n)
	})
}

func TestCopyBufferWithIdleTimeout_ShortBufferError(t *testing.T) {
	mockReader := mocks.NewMockReader(t)
	mockReader.On("Read", mock.Anything).Return(2, nil)

	mockWriter := mocks.NewMockWriter(t)
	mockWriter.On("Write", mock.Anything).Return(1, nil)

	_, err := CopyBufferWithIdleTimeout(mockWriter, mockReader, nil, time.Minute)
	assert.Error(t, err)

	t.Cleanup(func() {
		n, err := mockReader.Read(nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		n, err = mockWriter.Write(nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
	})
}

func TestCopyBufferWithIdleTimeout_ReadFailed(t *testing.T) {
	mockReader := mocks.NewMockReader(t)
	mockReader.On("Read", mock.Anything).Return(0, errors.NewNotSupported("testing"))

	mockWriter := mocks.NewMockWriter(t)
	mockWriter.On("Write", mock.Anything).Return(1, nil)

	_, err := CopyBufferWithIdleTimeout(mockWriter, mockReader, nil, time.Minute)
	assert.Error(t, err)

	t.Cleanup(func() {
		n, err := mockReader.Read(nil)
		assert.Error(t, err)
		assert.Equal(t, "testing", err.Error())
		assert.Equal(t, 0, n)
		n, err = mockWriter.Write(nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
	})
}
