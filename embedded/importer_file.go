// Copyright 2021 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/importer/file.dart

import (
	"fmt"
	"net/url"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/bancek/go-sass/eval"
)

// FileImporter is an importer that asks the host to resolve imports in a
// simplified, file-system-centric way.
//
// Unlike HostImporter it only round-trips non-file URLs; file URLs and all
// loads stay on the local filesystem, so host involvement is limited to
// mapping a URL to a file: location.
//
// Matches Dart: final class FileImporter extends ImporterBase in
// importer/file.dart.
type FileImporter struct {
	// base carries the dispatcher requests are sent through.
	base ImporterBase
	// importerID is the host-provided ID of the importer to invoke.
	importerID uint32
}

// NewFileImporter creates a FileImporter that sends file-import requests for
// importerID through dispatcher.
//
// Matches Dart: FileImporter constructor in importer/file.dart.
func NewFileImporter(dispatcher *CompilationDispatcher, importerID uint32) *FileImporter {
	return &FileImporter{
		base:       ImporterBase{Dispatcher: dispatcher},
		importerID: importerID,
	}
}

// Canonicalize resolves URL to a canonical file: URL.
//
// file: URLs never reach the host and go straight to the working-directory
// filesystem importer. Other schemes send a FileImportRequest; the host's
// file URL must itself use the file: scheme and is then resolved locally the
// same way. Containing-URL access tracking, error propagation, and empty
// (declined) results behave exactly as in HostImporter.Canonicalize.
//
// Matches Dart: FileImporter.canonicalize in importer/file.dart.
func (f *FileImporter) Canonicalize(u *url.URL, ctx *eval.CanonicalizeContext) (*url.URL, error) {
	if u.Scheme == "file" {
		return eval.NewFilesystemImporterCwd(f.base.Dispatcher.IO).Canonicalize(u, ctx)
	}

	request := &OutboundMessage_FileImportRequest{
		Id:         outboundRequestID,
		ImporterId: f.importerID,
		Url:        u.String(),
		FromImport: ctx.FromImport,
	}
	if containingURL := ctx.ContainingURLWithoutMarking(); containingURL != nil {
		request.ContainingUrl = proto.String(containingURL.String())
	}

	response, err := f.base.Dispatcher.sendFileImportRequest(request)
	if err != nil {
		return nil, err
	}

	if !response.GetContainingUrlUnused() {
		ctx.ContainingURL() // mark as accessed
	}

	switch response.GetResult().(type) {
	case *InboundMessage_FileImportResponse_FileUrl:
		fileURL, err := f.base.ParseAbsoluteURL("The file importer", response.GetFileUrl())
		if err != nil {
			return nil, err
		}
		if fileURL.Scheme != "file" {
			return nil, fmt.Errorf(
				"The file importer must return a file: URL, was \"%s\"", fileURL.String())
		}
		return eval.NewFilesystemImporterCwd(f.base.Dispatcher.IO).Canonicalize(fileURL, ctx)
	case *InboundMessage_FileImportResponse_Error:
		return nil, fmt.Errorf("%s", response.GetError())
	default:
		return nil, nil
	}
}

// Load loads the stylesheet at canonicalURL from the local filesystem.
//
// Canonicalize already forced the URL through the filesystem importer, so by
// the time Load runs the host is out of the loop entirely.
//
// Matches Dart: FileImporter.load (delegates to FilesystemImporter.cwd.load)
// in importer/file.dart.
func (f *FileImporter) Load(canonicalURL *url.URL) (*eval.ImporterResult, error) {
	return eval.NewFilesystemImporterCwd(f.base.Dispatcher.IO).Load(canonicalURL)
}

// CouldCanonicalize always returns false: like the host importer, the file
// importer cannot pre-check without either local resolution or a host
// round-trip, so callers always go through Canonicalize.
func (f *FileImporter) CouldCanonicalize(u, canonicalURL *url.URL) bool {
	return false
}

// IsNonCanonicalScheme returns true for every scheme except file: only file:
// URLs are ever canonical for this importer, so everything else counts as
// non-canonical.
//
// Matches Dart: FileImporter.isNonCanonicalScheme in importer/file.dart.
func (f *FileImporter) IsNonCanonicalScheme(scheme string) bool {
	return scheme != "file"
}

// ModificationTime returns the current time: file-importer results resolve
// through the host, whose freshness the compiler cannot observe, so they are
// treated as always fresh just like host-importer results.
func (f *FileImporter) ModificationTime(canonicalURL *url.URL) (time.Time, error) {
	return time.Now(), nil
}
