// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/util/length_delimited_transformer.dart

import (
	"io"
)

// ReadPacket reads one length-delimited packet from r: an unsigned varint
// length prefix (at most 53 bits) followed by that many payload bytes.
//
// Dart implements this as a chunked stream transformer that buffers partial
// lengths and payloads across arbitrary chunk boundaries (a chunk may hold a
// length plus a message, several messages, or half a message). Go instead
// serves the blocking stdio server loop, so it reads one byte at a time
// until the length varint completes and then reads the payload in full;
// framing errors surface as plain errors for the dispatcher to wrap.
//
// Matches Dart: lengthDelimitedDecoder in
// util/length_delimited_transformer.dart.
func ReadPacket(r io.Reader) ([]byte, error) {
	packetLengthBuilder := NewVarintBuilder(53, "packet length")
	defer packetLengthBuilder.Reset()

	var packetLength int
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		length, done, err := packetLengthBuilder.Add(buf[0])
		if err != nil {
			return nil, err
		}
		if done {
			packetLength = length
			break
		}
	}

	packet := make([]byte, packetLength)
	if _, err := io.ReadFull(r, packet); err != nil {
		return nil, err
	}
	return packet, nil
}

// WritePacket writes packet to w with an unsigned varint length prefix so it
// can travel over a medium like stdio that preserves no packet boundaries.
//
// Dart's encoder special-cases the empty message; Go needs no branch because
// EncodeVarint(0) already yields a single zero byte.
//
// Matches Dart: lengthDelimitedEncoder in
// util/length_delimited_transformer.dart.
func WritePacket(w io.Writer, packet []byte) error {
	lengthVarint := EncodeVarint(uint32(len(packet)))
	if _, err := w.Write(lengthVarint); err != nil {
		return err
	}
	_, err := w.Write(packet)
	return err
}

// EncodeVarint encodes value as an unsigned varint: 7 payload bits per byte
// with the high bit marking continuation, sized at ceil(bitLength/7) bytes.
//
// Dart's serializeVarint rejects negatives explicitly; Go takes a uint32 so
// negativity is unrepresentable by type. Zero encodes as a single zero byte.
//
// Matches Dart: serializeVarint in embedded/utils.dart.
func EncodeVarint(value uint32) []byte {
	if value == 0 {
		return []byte{0}
	}

	// Compute byte count: ceil(bitLength / 7)
	var bitLength int
	v := value
	for v > 0 {
		bitLength++
		v >>= 1
	}
	lengthInBytes := (bitLength + 6) / 7

	result := make([]byte, lengthInBytes)
	for i := 0; i < lengthInBytes; i++ {
		// The highest-order bit indicates whether more bytes are necessary.
		b := byte(value & 0x7f)
		if value > 0x7f {
			b |= 0x80
		}
		result[i] = b
		value >>= 7
	}
	return result
}
