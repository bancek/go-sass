// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/reusable_isolate.dart

import (
	"errors"
	"sync"
)

// IsolateEntryPoint is the function a pooled isolate runs, receiving
// inbound packets from mailbox and emitting replies through sendPort.
//
// Dart requires a static global entry point because isolates cannot capture
// state and the mailbox crosses the isolate boundary as a sendable; Go
// goroutines close over freely, so any function with this shape qualifies.
// A reply sent before Borrow subscribes has no listener, matching Dart
// raising while unborrowed.
//
// Matches Dart: ReusableIsolateEntryPoint typedef
type IsolateEntryPoint func(mailbox *Mailbox, sendPort *SendPort)

// Mailbox is a rendezvous channel carrying packets to one isolate. Sends
// block until the isolate takes the packet, which serializes each
// compilation's inbound flow to a single in-flight message.
//
// Matches Dart: package:native_synchronization Mailbox
type Mailbox struct {
	ch chan []byte
}

// NewMailbox returns an empty Mailbox ready to pair a sender with one
// isolate goroutine.
//
// Matches Dart: Mailbox()
func NewMailbox() *Mailbox {
	return &Mailbox{ch: make(chan []byte)}
}

// Put hands message to the isolate, blocking until Take receives it. The
// rendezvous means a second send waits for the isolate to take the first,
// which is how the single in-flight message discipline is enforced.
//
// Matches Dart: Mailbox.put
func (m *Mailbox) Put(message []byte) {
	m.ch <- message
}

// Take blocks for the next message and returns it. A nil return means the
// mailbox was closed, which the dispatch loops treat as a clean shutdown
// rather than an error; Dart raises there instead and its receiver exits
// the isolate from the handler.
//
// Returns nil if the mailbox has been closed.
//
// Matches Dart: Mailbox.take
func (m *Mailbox) Take() []byte {
	msg, ok := <-m.ch
	if !ok {
		return nil
	}
	return msg
}

// Close shuts the mailbox, unblocking any pending Take with a nil return.
// Dart additionally kills the isolate immediately and closes the receive
// port; Go needs neither, since the goroutine returns once Take reports
// the closure.
//
// Matches Dart: Mailbox.close
func (m *Mailbox) Close() {
	close(m.ch)
}

// SendPort delivers packets to its paired ReceivePort over a shared
// channel. Sends may block until the listener drains, matching Dart's
// SendPort backpressure.
//
// Matches Dart: dart:isolate SendPort
type SendPort struct {
	ch chan []byte
}

// NewSendPort returns a SendPort over a fresh channel. Spawned isolates
// share one channel between their send and receive ports instead, so a
// standalone port from this constructor only pairs with a ReceivePort
// built over the same channel.
func NewSendPort() *SendPort {
	return &SendPort{ch: make(chan []byte)}
}

// Send hands message to the paired ReceivePort, blocking until the drain
// loop takes it.
//
// Matches Dart: SendPort.send
func (s *SendPort) Send(message []byte) {
	s.ch <- message
}

// ReceivePort drains packets from its paired SendPort and fans them out
// to the subscribed handler.
//
// Matches Dart: dart:isolate ReceivePort
type ReceivePort struct {
	ch      chan []byte
	onData  func([]byte)
	mu      sync.Mutex
	started chan struct{}
}

// NewReceivePort returns a ReceivePort over a fresh channel that starts
// draining on the first Listen call. Like NewSendPort, it only pairs with
// a SendPort built over the same channel.
func NewReceivePort() *ReceivePort {
	return &ReceivePort{
		ch:      make(chan []byte),
		started: make(chan struct{}),
	}
}

// Listen subscribes onData to drained packets. The first call starts the
// single drain goroutine; later calls only swap the handler, so Borrow can
// subscribe each compilation and Release can stand down without ever
// starting a second listener. A nil onData silences delivery while keeping
// the drain running.
//
// Matches Dart: ReceivePort.listen
func (r *ReceivePort) Listen(onData func([]byte)) {
	r.mu.Lock()
	alreadyStarted := r.onData != nil
	r.onData = func(msg []byte) {
		if onData != nil {
			onData(msg)
		}
	}
	r.mu.Unlock()
	if !alreadyStarted {
		close(r.started)
		go func() {
			for msg := range r.ch {
				r.mu.Lock()
				fn := r.onData
				r.mu.Unlock()
				if fn != nil {
					fn(msg)
				}
			}
		}()
	}
}

// ReusableIsolate wraps one long-lived goroutine that serves many
// compilations in sequence, standing in for Dart's reusable Isolate.
//
// The goroutine outlives any single compilation: Borrow subscribes one
// compilation's reply handler, Send feeds it inbound packets, and Release
// frees it back to the dispatcher's inactive set. At most one compilation
// borrows an isolate at a time; Kill shuts the goroutine down at session
// end.
//
// Matches Dart: class ReusableIsolate
type ReusableIsolate struct {
	mailbox     *Mailbox
	sendPort    *SendPort
	receivePort *ReceivePort
	borrowed    bool
}

// SpawnReusableIsolate launches entryPoint on a new goroutine and returns
// its handle. The send and receive ports share one channel so replies from
// the goroutine reach whoever borrows it, and the mailbox starts
// unborrowed, matching Dart spawning the isolate before any compilation
// claims it. Dart ferries the entry point plus a sendable mailbox through
// Isolate.spawn, which Go's closure-capturing goroutine makes unnecessary.
//
// Matches Dart: ReusableIsolate.spawn
func SpawnReusableIsolate(entryPoint IsolateEntryPoint) *ReusableIsolate {
	mailbox := NewMailbox()
	ch := make(chan []byte)
	sendPort := &SendPort{ch: ch}
	receivePort := &ReceivePort{ch: ch, started: make(chan struct{})}

	go entryPoint(mailbox, sendPort)

	return &ReusableIsolate{
		mailbox:     mailbox,
		sendPort:    sendPort,
		receivePort: receivePort,
	}
}

// Borrow subscribes onData to the isolate's replies for one compilation.
// It reports an error when the isolate is already borrowed, matching
// Dart throwing StateError on a double borrow.
//
// Matches Dart: ReusableIsolate.borrow
func (r *ReusableIsolate) Borrow(onData func([]byte)) error {
	if r.borrowed {
		return errors.New("ReusableIsolate has already been borrowed.")
	}
	r.borrowed = true
	r.receivePort.Listen(onData)
	return nil
}

// Release frees a borrowed isolate back to the dispatcher's inactive set.
// It reports an error when the isolate is not borrowed, matching Dart
// throwing StateError on a spare release. The channel stays open for the
// next Borrow; Dart additionally resets its subscription to a handler that
// throws while unborrowed.
//
// Matches Dart: ReusableIsolate.release
func (r *ReusableIsolate) Release() error {
	if !r.borrowed {
		return errors.New("ReusableIsolate has not been borrowed.")
	}
	r.borrowed = false
	return nil
}

// Send feeds one inbound packet to the isolate's mailbox, blocking until
// the isolate takes it. It reports an error when the isolate is not
// borrowed, matching Dart throwing StateError on an early send; the
// rendezvous mailbox additionally serializes sends the way Dart's
// single-message mailbox does.
//
// Matches Dart: ReusableIsolate.send
func (r *ReusableIsolate) Send(message []byte) error {
	if !r.borrowed {
		return errors.New("Cannot send a message before being borrowed.")
	}
	r.mailbox.Put(message)
	return nil
}

// Kill shuts the isolate down by closing its mailbox, so a Take blocked
// in receive wakes with nil and the goroutine returns. Dart additionally
// kills the isolate immediately and closes the receive port, which
// goroutine teardown does not need.
//
// Matches Dart: ReusableIsolate.kill
func (r *ReusableIsolate) Kill() {
	r.mailbox.Close()
}
