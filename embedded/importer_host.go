// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/importer/host.dart

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/bancek/go-sass/eval"
)

// HostImporter is an importer that asks the host to resolve imports over the
// embedded protocol.
//
// Each compilation holds one per host-provided importer ID; canonicalize and
// load become synchronous round-trips the dispatcher answers from the host.
//
// Matches Dart: final class HostImporter extends ImporterBase in
// importer/host.dart.
type HostImporter struct {
	// base carries the dispatcher requests are sent through.
	base ImporterBase
	// importerID is the host-provided ID of the importer to invoke.
	importerID uint32
	// nonCanonicalSchemes holds URL schemes this importer promises never to
	// return from Canonicalize. It corresponds to Dart's _nonCanonicalSchemes
	// set.
	nonCanonicalSchemes map[string]struct{}
}

// NewHostImporter creates a HostImporter that sends requests for importerID
// through dispatcher, treating nonCanonicalSchemes as never canonical.
//
// Every scheme is validated up front; an invalid one returns an error where
// Dart throws a SassException with a bogus span, since the embedded setup
// path has no span to attach.
//
// Matches Dart: HostImporter constructor in importer/host.dart.
func NewHostImporter(dispatcher *CompilationDispatcher, importerID uint32, nonCanonicalSchemes []string) (*HostImporter, error) {
	schemes := make(map[string]struct{}, len(nonCanonicalSchemes))
	for _, s := range nonCanonicalSchemes {
		schemes[s] = struct{}{}
	}
	for _, scheme := range nonCanonicalSchemes {
		if !isValidURLScheme(scheme) {
			return nil, fmt.Errorf("%q isn't a valid URL scheme (for example \"file\").", scheme)
		}
	}
	return &HostImporter{
		base:                ImporterBase{Dispatcher: dispatcher},
		importerID:          importerID,
		nonCanonicalSchemes: schemes,
	}, nil
}

// Canonicalize asks the host to canonicalize URL via a CanonicalizeRequest.
//
// The containing URL travels along only for access tracking and is marked
// accessed when the host reports it used the value, which feeds the import
// cache's containing-URL dependency logic. A URL result must be absolute, an
// error result fails the load, and an empty result means "unresolved" (nil,
// nil) so the next importer may try.
//
// Matches Dart: HostImporter.canonicalize in importer/host.dart.
func (h *HostImporter) Canonicalize(u *url.URL, ctx *eval.CanonicalizeContext) (*url.URL, error) {
	request := &OutboundMessage_CanonicalizeRequest{
		Id:         outboundRequestID,
		ImporterId: h.importerID,
		Url:        u.String(),
		FromImport: ctx.FromImport,
	}
	if containingURL := ctx.ContainingURLWithoutMarking(); containingURL != nil {
		request.ContainingUrl = proto.String(containingURL.String())
	}

	response, err := h.base.Dispatcher.sendCanonicalizeRequest(request)
	if err != nil {
		return nil, err
	}

	if !response.GetContainingUrlUnused() {
		ctx.ContainingURL() // mark as accessed
	}

	switch response.GetResult().(type) {
	case *InboundMessage_CanonicalizeResponse_Url:
		return h.base.ParseAbsoluteURL("The importer", response.GetUrl())
	case *InboundMessage_CanonicalizeResponse_Error:
		return nil, fmt.Errorf("%s", response.GetError())
	default:
		return nil, nil
	}
}

// Load asks the host to load the stylesheet at canonicalURL via an
// ImportRequest.
//
// A success result becomes an ImporterResult with the host's contents,
// syntax, and optional source-map URL (empty means absent); an error result
// fails the load and an empty result declines it so import fallthrough can
// continue.
//
// Matches Dart: HostImporter.load in importer/host.dart.
func (h *HostImporter) Load(canonicalURL *url.URL) (*eval.ImporterResult, error) {
	response, err := h.base.Dispatcher.sendImportRequest(
		&OutboundMessage_ImportRequest{
			Id:         outboundRequestID,
			ImporterId: h.importerID,
			Url:        canonicalURL.String(),
		})
	if err != nil {
		return nil, err
	}

	switch response.GetResult().(type) {
	case *InboundMessage_ImportResponse_Success:
		success := response.GetSuccess()
		var sourceMapURL *url.URL
		if smURL := success.GetSourceMapUrl(); smURL != "" {
			var parseErr error
			sourceMapURL, parseErr = h.base.ParseAbsoluteURL("The importer", smURL)
			if parseErr != nil {
				return nil, parseErr
			}
		}
		syntax, err := syntaxToSyntax(success.GetSyntax())
		if err != nil {
			return nil, err
		}
		return eval.NewImporterResult(success.GetContents(), syntax, sourceMapURL)
	case *InboundMessage_ImportResponse_Error:
		return nil, fmt.Errorf("%s", response.GetError())
	default:
		return nil, nil
	}
}

// CouldCanonicalize always returns false: a host importer cannot answer
// whether it could canonicalize a URL without a host round-trip, so the
// import cache never pre-checks it and always calls Canonicalize.
func (h *HostImporter) CouldCanonicalize(u, canonicalURL *url.URL) bool {
	return false
}

// IsNonCanonicalScheme returns whether scheme is one this importer promised
// never to return from Canonicalize, consulting the set validated by
// NewHostImporter.
//
// Matches Dart: HostImporter.isNonCanonicalScheme in importer/host.dart.
func (h *HostImporter) IsNonCanonicalScheme(scheme string) bool {
	_, ok := h.nonCanonicalSchemes[scheme]
	return ok
}

// ModificationTime returns the current time: host-provided stylesheets have
// no observable filesystem mtime, so callers treat them as always fresh and
// never cache on modification time.
func (h *HostImporter) ModificationTime(canonicalURL *url.URL) (time.Time, error) {
	return time.Now(), nil
}

// urlSchemeRegExp matches valid URL schemes: lowercase alphanumerics plus
// "+", "-", and ".", mirroring Dart's importer/utils.dart pattern. Case is
// significant — schemes are lowercased before matching upstream.
//
// Matches Dart: _urlSchemeRegExp in importer/utils.dart.
var urlSchemeRegExp = regexp.MustCompile(`^[a-z0-9+.-]+$`)

// isValidURLScheme returns whether scheme is a valid URL scheme per
// urlSchemeRegExp. It guards NewHostImporter so a typo surfaces at setup
// rather than as a mysterious canonicalize failure later.
//
// Matches Dart: isValidUrlScheme in importer/utils.dart.
func isValidURLScheme(scheme string) bool {
	return urlSchemeRegExp.MatchString(scheme)
}
