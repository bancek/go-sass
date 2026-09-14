// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/isolate_dispatcher.dart

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/proto"
)

// maxConcurrentCompilations bounds how many isolates (and therefore
// compilations) may be live at once. Dart sizes its pool by pointer width
// (7 on 32-bit, 15 otherwise) because too many isolates in one isolate
// group can deadlock the VM; Go keeps the 64-bit budget unconditionally
// since goroutine isolates have no such group limit but still need a cap.
const maxConcurrentCompilations = 15

// compilerVersion is the compiler release reported to the host in version
// responses. Dart injects it at build time from the environment; Go holds
// the same value as a variable so the lockstep checklist in
// ref/embedded.md can catch version skew.
//
// Matches Dart: const String.fromEnvironment("compiler-version")
var compilerVersion = "1.104.0"

// protocolVersion is the embedded protocol release reported to the host
// in version responses. Dart injects it at build time from the
// environment; Go holds the same value as a variable so the lockstep
// checklist in ref/embedded.md can catch version skew.
//
// Matches Dart: const String.fromEnvironment("protocol-version")
var protocolVersion = "3.2.0"

// IsolateDispatcher routes host packets to per-compilation isolates and
// carries their replies back to the host.
//
// Nonzero compilation IDs each run on a pooled reusable isolate; ID zero
// is reserved for the version handshake answered inline. The pool caps
// concurrent compilations, and finished isolates return to the inactive
// set for reuse.
//
// Matches Dart: class IsolateDispatcher
type IsolateDispatcher struct {
	reader io.Reader
	writer PacketWriter
	// Stderr receives human-readable diagnostics for transport and
	// protocol errors. A nil writer falls back to os.Stderr.
	Stderr io.Writer
	// ExitFn terminates the process on fatal compiler errors. It
	// defaults to os.Exit and is swapped out in tests.
	ExitFn func(int)

	allIsolates      []*ReusableIsolate
	inactiveIsolates []*ReusableIsolate
	activeIsolates   map[uint32]*ReusableIsolate
	isolatePool      chan struct{}
	closed           atomic.Bool
	mu               sync.Mutex
	wg               sync.WaitGroup
}

// NewIsolateDispatcher returns a dispatcher reading length-delimited
// packets from reader and writing replies through writer, which is wrapped
// in a concurrent packet writer so isolates can answer in parallel. A nil
// stderrWriter falls back to os.Stderr and ExitFn defaults to os.Exit;
// both are assignable for tests.
//
// Matches Dart: IsolateDispatcher(StreamChannel)
func NewIsolateDispatcher(reader io.Reader, writer io.Writer, stderrWriter io.Writer) *IsolateDispatcher {
	d := &IsolateDispatcher{
		reader:         reader,
		writer:         newConcurrentPacketWriter(writer),
		Stderr:         stderrWriter,
		ExitFn:         os.Exit,
		activeIsolates: make(map[uint32]*ReusableIsolate),
		isolatePool:    make(chan struct{}, maxConcurrentCompilations),
	}
	return d
}

// Listen reads host packets until the input stream closes or fails,
// dispatching each packet to its compilation's isolate.
//
// Every packet is handled on its own goroutine so a slow compilation never
// stalls later reads; blocking on the isolate pool happens there rather
// than on the read loop. End of input marks the dispatcher closed, kills
// every spawned isolate, and waits for in-flight handlers before
// returning. A read failure is reported once and returned to the caller.
//
// Matches Dart: IsolateDispatcher.listen
func (d *IsolateDispatcher) Listen() error {
	for {
		packet, err := ReadPacket(d.reader)
		if err != nil {
			if err == io.EOF {
				d.closed.Store(true)
				for _, isolate := range d.allIsolates {
					isolate.Kill()
				}
				d.wg.Wait()
				return nil
			}
			d.handleError(err, "")
			return err
		}

		d.wg.Go(func() {
			if err := d.handlePacket(packet); err != nil {
				d.handleError(err, "")
			}
		})
	}
}

// handlePacket routes one inbound packet: nonzero compilation IDs go to
// their isolate (spawning or reusing one as needed, and dropping the
// packet if shutdown has begun), while ID zero must carry a version
// request, which is answered inline with the protocol, compiler, and
// implementation versions. Anything else on ID zero is a parameter error.
//
// Matches Dart: IsolateDispatcher.listen (packet callback)
func (d *IsolateDispatcher) handlePacket(packet []byte) error {
	compilationID, messageBuffer, err := parsePacket(packet)
	if err != nil {
		return err
	}

	if compilationID != 0 {
		isolate, err := d.getIsolate(compilationID)
		if err != nil {
			return err
		}

		if d.closed.Load() {
			return nil
		}

		return isolate.Send(packet)
	}

	message := &InboundMessage{}
	if err := proto.Unmarshal(messageBuffer, message); err != nil {
		return parseError(err.Error())
	}

	if _, ok := message.GetMessage().(*InboundMessage_VersionRequest_); !ok {
		return paramsError("Only VersionRequest may have wire ID 0.")
	}

	request := message.GetVersionRequest()
	response := &OutboundMessage_VersionResponse{
		Id:                    request.GetId(),
		ProtocolVersion:       protocolVersion,
		CompilerVersion:       compilerVersion,
		ImplementationVersion: compilerVersion,
		ImplementationName:    "dart-sass",
	}
	d.send(0, &OutboundMessage{Message: &OutboundMessage_VersionResponse_{VersionResponse: response}})
	return nil
}

// getIsolate returns the isolate running compilationID, starting it on a
// pooled isolate when this is its first packet.
//
// An already-active compilation reuses its isolate without touching the
// pool. Otherwise the caller holds a pool token while an inactive isolate
// is recycled or a fresh goroutine is spawned, then rechecks the active
// map in case another packet for the same compilation won the race and
// releases the token unused. Borrow wires the isolate's replies through a
// category prefix: ordinary traffic is forwarded, a finished compilation
// returns the isolate to the inactive set and releases the pool token, and
// a fatal error is forwarded before the process exits. The prefix is
// consumed here so a host reuse of one compilation ID cannot interleave
// with a stale completion notice.
//
// Matches Dart: _getIsolate
func (d *IsolateDispatcher) getIsolate(compilationID uint32) (*ReusableIsolate, error) {
	d.mu.Lock()

	// If this compilation is already active, return the existing isolate.
	if isolate, ok := d.activeIsolates[compilationID]; ok {
		d.mu.Unlock()
		return isolate, nil
	}

	d.mu.Unlock()

	// Matches Dart: var resource = await _isolatePool.request()
	// In Dart, await suspends the callback, letting the event loop process
	// ReceivePort messages that drain the pool. In Go, handlePacket runs in
	// its own goroutine, so blocking here doesn't stall stdin reads.
	d.isolatePool <- struct{}{}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check: another goroutine may have started the compilation
	// while we were waiting for the pool token.
	if isolate, ok := d.activeIsolates[compilationID]; ok {
		<-d.isolatePool // release resource
		return isolate, nil
	}

	var isolate *ReusableIsolate
	if len(d.inactiveIsolates) > 0 {
		isolate = d.inactiveIsolates[0]
		d.inactiveIsolates = d.inactiveIsolates[1:]
	} else {
		isolate = SpawnReusableIsolate(func(mailbox *Mailbox, sendPort *SendPort) {
			cd := NewCompilationDispatcher(mailbox, sendPort)
			cd.Stderr = d.Stderr
			cd.Listen()
		})
		d.allIsolates = append(d.allIsolates, isolate)
	}

	if err := isolate.Borrow(func(fullBuffer []byte) {
		category := fullBuffer[0]
		packet := fullBuffer[1:]

		switch category {
		case 0:
			_ = d.writer.WritePacket(packet)
		case 1:
			d.mu.Lock()
			delete(d.activeIsolates, compilationID)
			isolate.Release()
			d.inactiveIsolates = append(d.inactiveIsolates, isolate)
			d.mu.Unlock()
			<-d.isolatePool
			_ = d.writer.WritePacket(packet)
		case 2:
			_ = d.writer.WritePacket(packet)
			d.ExitFn(exitCode)
		}
	}); err != nil {
		return nil, fmt.Errorf("getIsolate: Borrow failed: %v", err)
	}

	d.activeIsolates[compilationID] = isolate
	return isolate, nil
}

// send frames message with compilationID and writes one packet to the
// host.
//
// Matches Dart: _send
func (d *IsolateDispatcher) send(compilationID uint32, message proto.Message) {
	_ = d.writer.WritePacket(serializePacket(compilationID, message))
}

// sendError wraps protocolError in the outbound error branch and sends it
// to the host under compilationID.
//
// Matches Dart: sendError
func (d *IsolateDispatcher) sendError(compilationID uint32, protocolError *ProtocolError) {
	d.send(compilationID, &OutboundMessage{Message: &OutboundMessage_Error{Error: protocolError}})
}

// handleError converts a dispatcher-level failure into a protocol error
// packet addressed to the reserved error ID, which carries no compilation
// context. The read loop then returns so the caller can shut the session
// down, matching Dart closing the channel sink after reporting.
//
// Matches Dart: _handleError
func (d *IsolateDispatcher) handleError(err error, stack string) {
	d.sendError(errorId, handleError(err, stack, nil, d.Stderr))
}

// compilationDispatcherEntryPoint runs a fresh compilation dispatcher on a
// pooled isolate. It matches Dart's isolate main, which constructs a
// CompilationDispatcher on the new isolate; Go passes the mailbox and send
// port directly instead of tunneling them through an isolate-spawn
// message.
//
// Matches Dart: _isolateMain
func compilationDispatcherEntryPoint(mailbox *Mailbox, sendPort *SendPort) {
	NewCompilationDispatcher(mailbox, sendPort).Listen()
}
