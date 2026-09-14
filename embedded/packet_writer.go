// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package embedded

import (
	"io"
	"sync"
)

// PacketWriter is the interface for writing length-delimited packets.
type PacketWriter interface {
	WritePacket(packet []byte) error
}

// concurrentPacketWriter serializes writes to an underlying io.Writer.
type concurrentPacketWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// newConcurrentPacketWriter wraps w for concurrent-safe writes.
func newConcurrentPacketWriter(w io.Writer) *concurrentPacketWriter {
	return &concurrentPacketWriter{w: w}
}

func (cw *concurrentPacketWriter) WritePacket(packet []byte) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return WritePacket(cw.w, packet)
}
