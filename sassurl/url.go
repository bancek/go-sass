// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

// dart-source: sdk/lib/core/uri.dart (Dart 3.13.5) — Uri.parse/resolve,
// normalization, and error messages; file paths follow package:path's
// toUri/fromUri behavior.

// Package sassurl provides a Dart-compatible URI implementation.
//
// Dart's Uri normalizes and resolves URLs differently from net/url in several
// observable ways. This is a port of the subset Sass needs: parsing (including
// relative references), component normalization, resolution, file-path
// conversion (package:path), and data: URIs.
package sassurl

import (
	"hash/fnv"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	percent      = 0x25
	backslash    = 0x5c
	colon        = 0x3a
	dot          = 0x2e
	leftBracket  = 0x5b
	rightBracket = 0x5d
	lowerCaseA   = 0x61
	lowerCaseF   = 0x66
	lowerCaseV   = 0x76
	lowerCaseZ   = 0x7a
	upperCaseA   = 0x41
	upperCaseZ   = 0x5a
	digit0       = 0x30
)

var hexDigits = "0123456789ABCDEF"

const (
	unreservedMask      uint16 = 0x0001
	schemeMask          uint16 = 0x0008
	userInfoMask        uint16 = 0x0010
	regNameMask         uint16 = 0x0020
	pathCharOrSlashMask uint16 = 0x0080
	queryCharMask       uint16 = 0x0100
	genDelimitersMask   uint16 = 0x0400
)

const (
	schemeEndIndex     = 1
	hostStartIndex     = 2
	portStartIndex     = 3
	pathStartIndex     = 4
	queryStartIndex    = 5
	fragmentStartIndex = 6
	notSimpleIndex     = 7
)

const (
	uriStart           = 0
	nonSimpleEndStates = 14
	schemeStart        = 20
)

// DartUriError is a Dart FormatException raised by Uri.parse.
type DartUriError struct {
	Message string
}

func (e *DartUriError) Error() string { return e.Message }

func uriError(message string) error { return &DartUriError{Message: message} }

type uriKind uint8

const (
	simpleURI uriKind = iota
	fullURI
)

// URL is a Dart-compatible URI.
//
// Field names follow the net/url surface used by the rest of the port where
// that is harmless; Dart semantics live in String, Resolve, Equal, and the
// Has* methods.
type URL struct {
	Scheme   string
	UserInfo string
	// Host is Dart's Uri.host getter: IP literals lose their brackets.
	Host string

	Port    int
	HasPort bool

	Path string

	Query       string
	HasQuery    bool
	Fragment    string
	HasFragment bool

	HasAuthority bool

	// hostRaw keeps the serialized host (including IPv6 brackets).
	hostRaw string
	kind    uriKind
}

// Parse parses raw exactly as Dart's Uri.parse does.
func Parse(raw string) (*URL, error) {
	units := toUnits(raw)
	if rest, ok := stripDataScheme(units); ok {
		return parseDataURI(units, rest)
	}
	return parseReference(units)
}

func toUnits(s string) []uint16 { return utf16.Encode([]rune(s)) }

func fromUnits(units []uint16) string { return string(utf16.Decode(units)) }

func (u *URL) make(scheme, userInfo string, host *string, port *int, path string, query, fragment *string) *URL {
	u.Scheme = scheme
	u.UserInfo = userInfo
	u.Path = path
	u.Port = -1
	u.HasPort = false
	if host != nil {
		u.HasAuthority = true
		u.hostRaw = *host
		u.Host = dartHost(*host)
	}
	if port != nil {
		u.HasPort = true
		u.Port = *port
	}
	if query != nil {
		u.HasQuery = true
		u.Query = *query
	}
	if fragment != nil {
		u.HasFragment = true
		u.Fragment = *fragment
	}
	return u
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func dartHost(host string) string {
	if strings.HasPrefix(host, "[") && !strings.HasPrefix(host, "[v") {
		return host[1 : len(host)-1]
	}
	return host
}

func (u *URL) String() string {
	var b strings.Builder
	if u.Scheme != "" {
		b.WriteString(u.Scheme)
		b.WriteByte(':')
	}
	if u.HasAuthority || u.Scheme == "file" {
		b.WriteString("//")
		if u.UserInfo != "" {
			b.WriteString(u.UserInfo)
			b.WriteByte('@')
		}
		if u.hostRaw != "" {
			b.WriteString(u.hostRaw)
		} else {
			b.WriteString(u.Host)
		}
		if u.HasPort {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(u.Port))
		}
	}
	b.WriteString(u.Path)
	if u.HasQuery {
		b.WriteByte('?')
		b.WriteString(u.Query)
	}
	if u.HasFragment {
		b.WriteByte('#')
		b.WriteString(u.Fragment)
	}
	return b.String()
}

func (u *URL) HasScheme() bool       { return u.Scheme != "" }
func (u *URL) IsRelative() bool      { return u.Scheme == "" }
func (u *URL) IsFile() bool          { return u.Scheme == "file" }
func (u *URL) IsFileLike() bool      { return u.IsFile() || u.IsRelative() }
func (u *URL) HasAbsolutePath() bool { return strings.HasPrefix(u.Path, "/") }
func (u *URL) CannotBeABase() bool {
	return !u.HasAuthority && !strings.HasPrefix(u.Path, "/")
}
func (u *URL) PortNumber() int {
	if u.HasPort {
		return u.Port
	}
	return defaultPort(u.Scheme)
}
func (u *URL) IsAbs() bool {
	return u.Scheme != "" && !u.HasFragment
}

func (u *URL) Equal(other *URL) bool {
	if u == nil || other == nil {
		return u == other
	}
	return u.Scheme == other.Scheme &&
		u.HasAuthority == other.HasAuthority &&
		u.UserInfo == other.UserInfo &&
		u.Host == other.Host &&
		u.PortNumber() == other.PortNumber() &&
		u.Path == other.Path &&
		u.HasQuery == other.HasQuery &&
		u.Query == other.Query &&
		u.HasFragment == other.HasFragment &&
		u.Fragment == other.Fragment
}

func (u *URL) HashCode() int {
	h := fnv.New32a()
	writeHash := func(s string) { _, _ = h.Write([]byte(s)) }
	writeHash(u.Scheme)
	if u.HasAuthority {
		writeHash("1")
	} else {
		writeHash("0")
	}
	writeHash(u.UserInfo)
	writeHash(u.Host)
	writeHash(strconv.Itoa(u.PortNumber()))
	writeHash(u.Path)
	if u.HasQuery {
		writeHash("1")
	} else {
		writeHash("0")
	}
	writeHash(u.Query)
	if u.HasFragment {
		writeHash("1")
	} else {
		writeHash("0")
	}
	writeHash(u.Fragment)
	return int(h.Sum32())
}

// Resolve resolves reference using Dart's Uri.resolve.
func (u *URL) Resolve(reference *URL) *URL {
	var targetScheme string
	var targetUserInfo string
	var targetHost *string
	var targetPort *int
	var targetPath string
	var targetQuery *string
	usedSimpleMerge := false

	if reference.Scheme != "" {
		targetScheme = reference.Scheme
		if reference.HasAuthority {
			targetUserInfo = reference.UserInfo
			targetHost = hostPtr(reference)
			targetPort = portPtr(reference)
		}
		targetPath = removeDotSegments(reference.Path)
		if reference.HasQuery {
			targetQuery = strPtr(reference.Query)
		}
	} else {
		targetScheme = u.Scheme
		if reference.HasAuthority {
			targetUserInfo = reference.UserInfo
			targetHost = hostPtr(reference)
			targetPort = makePort(portPtr(reference), targetScheme)
			targetPath = removeDotSegments(reference.Path)
			if reference.HasQuery {
				targetQuery = strPtr(reference.Query)
			}
		} else {
			targetUserInfo = u.UserInfo
			targetHost = hostPtr(u)
			targetPort = portPtr(u)
			if reference.Path == "" {
				targetPath = u.Path
				if reference.HasQuery {
					targetQuery = strPtr(reference.Query)
				} else if u.HasQuery {
					targetQuery = strPtr(u.Query)
				}
			} else if reference.HasAbsolutePath() {
				targetPath = removeDotSegments(reference.Path)
				if reference.HasQuery {
					targetQuery = strPtr(reference.Query)
				}
			} else if u.Path == "" {
				if !u.HasAuthority {
					if u.Scheme == "" {
						targetPath = reference.Path
					} else {
						targetPath = removeDotSegments(reference.Path)
					}
				} else {
					targetPath = removeDotSegments("/" + reference.Path)
				}
				if reference.HasQuery {
					targetQuery = strPtr(reference.Query)
				}
			} else if u.kind == simpleURI && reference.kind == simpleURI &&
				u.Scheme == "" && !u.HasAuthority && !u.HasAbsolutePath() {
				targetPath = simpleMergeRelative(u.Path, reference.Path)
				if reference.HasQuery {
					targetQuery = strPtr(reference.Query)
				}
				usedSimpleMerge = true
			} else {
				merged := mergePaths(u.Path, reference.Path)
				if u.Scheme != "" || u.HasAuthority || u.HasAbsolutePath() {
					targetPath = removeDotSegments(merged)
				} else {
					targetPath = normalizeRelativePath(merged, u.Scheme != "" || u.HasAuthority)
				}
				if reference.HasQuery {
					targetQuery = strPtr(reference.Query)
				}
			}
		}
	}

	var fragment *string
	if reference.HasFragment {
		fragment = strPtr(reference.Fragment)
	}
	url := &URL{}
	url.make(targetScheme, targetUserInfo, targetHost, targetPort, targetPath, targetQuery, fragment)
	if usedSimpleMerge {
		url.kind = simpleURI
	} else {
		url.kind = fullURI
	}
	return url
}

func portPtr(u *URL) *int {
	if u.HasPort {
		return intPtr(u.Port)
	}
	return nil
}

func hostPtr(u *URL) *string {
	if !u.HasAuthority {
		return nil
	}
	if u.hostRaw != "" {
		return &u.hostRaw
	}
	return &u.Host
}

// ResolveString parses and resolves reference.
func (u *URL) ResolveString(reference string) (*URL, error) {
	ref, err := Parse(reference)
	if err != nil {
		return nil, err
	}
	return u.Resolve(ref), nil
}

// ResolveReference is the net/url-compatible spelling.
func (u *URL) ResolveReference(ref *URL) *URL { return u.Resolve(ref) }

// Resolve resolves ref against base.
func Resolve(base, ref *URL) *URL { return base.Resolve(ref) }

// --- Parsing --------------------------------------------------------------

func parseReference(units []uint16) (*URL, error) {
	start := 0
	end := len(units)
	indices := [8]int{0, -1, -1, -1, 0, end, end, -1}
	state := scan(units, start, end, uriStart, &indices)
	if state >= nonSimpleEndStates {
		indices[notSimpleIndex] = end
	}
	schemeEnd := indices[schemeEndIndex]
	if schemeEnd >= start {
		state = scan(units, start, schemeEnd, schemeStart, &indices)
		if state == schemeStart {
			indices[notSimpleIndex] = schemeEnd
		}
	}

	hostStart := indices[hostStartIndex] + 1
	portStart := indices[portStartIndex]
	pathStart := indices[pathStartIndex]
	queryStart := indices[queryStartIndex]
	fragmentStart := indices[fragmentStartIndex]

	if fragmentStart < queryStart {
		queryStart = fragmentStart
	}
	if pathStart < hostStart {
		pathStart = queryStart
	} else if pathStart <= schemeEnd {
		pathStart = schemeEnd + 1
	}
	if portStart < hostStart {
		portStart = pathStart
	}

	isSimple := indices[notSimpleIndex] < start
	if isSimple && (hostStart > schemeEnd+3 ||
		(portStart > start && portStart+1 == pathStart) ||
		startsWithAt(units, pathStart, `\`) ||
		(hostStart > start && (startsWithAt(units, hostStart-1, `\`) || startsWithAt(units, hostStart-2, `\`))) ||
		(queryStart < end && ((queryStart == pathStart+2 && startsWithAt(units, pathStart, "..")) ||
			(queryStart > pathStart+2 && startsWithAt(units, queryStart-3, "/.."))))) {
		isSimple = false
	}

	var scheme string
	if schemeEnd > start {
		var err error
		scheme, err = makeScheme(units, start, schemeEnd)
		if err != nil {
			return nil, err
		}
	} else if schemeEnd == start {
		return nil, uriError("Invalid empty scheme")
	}

	isLowercaseFileScheme := schemeEnd == 4 && startsWithAt(units, 0, "file")

	userInfo := ""
	if hostStart > start {
		userInfoStart := schemeEnd + 3
		if userInfoStart < hostStart {
			var err error
			userInfo, err = makeUserInfo(units, userInfoStart, hostStart-1)
			if err != nil {
				return nil, err
			}
		}
	}

	var host *string
	if hostStart > start {
		h, err := makeHost(units, hostStart, portStart, false)
		if err != nil {
			return nil, err
		}
		host = &h
	}

	var port *int
	if portStart+1 < pathStart {
		n, ok := parseDartInt(fromUnits(units[portStart+1 : pathStart]))
		if !ok {
			return nil, uriError("Invalid port")
		}
		port = makePort(intPtr(n), scheme)
	}

	if isLowercaseFileScheme && host == nil {
		empty := ""
		host = &empty
	}

	path, err := makePath(units, pathStart, queryStart, scheme, host != nil)
	if err != nil {
		return nil, err
	}

	var query *string
	if queryStart < fragmentStart {
		q, err := makeQuery(units, queryStart+1, fragmentStart)
		if err != nil {
			return nil, err
		}
		query = &q
	}

	var fragment *string
	if fragmentStart < end {
		f, err := makeFragment(units, fragmentStart+1, end)
		if err != nil {
			return nil, err
		}
		fragment = &f
	}

	url := &URL{}
	url.make(scheme, userInfo, host, port, path, query, fragment)
	if isSimple {
		url.kind = simpleURI
	} else {
		url.kind = fullURI
	}
	return url, nil
}

func scan(units []uint16, start, end, state int, indices *[8]int) int {
	const stateTableSize = 0x60
	const charXor uint16 = 96
	const xorCharLimit uint16 = 95
	const invalidChar = 0x7f ^ 96

	for i := start; i < end; i++ {
		ch := units[i] ^ charXor
		if ch > xorCharLimit {
			ch = invalidChar
		}
		transition := scannerTables[state*stateTableSize+int(ch)]
		state = int(transition & 31)
		indices[(transition >> 5)] = i
	}
	return state
}

// --- Component normalization ---------------------------------------------

func normalize(units []uint16, start, end int, charMask uint16, escapeDelimiters, replaceBackslash bool) (*string, error) {
	var buffer *strings.Builder
	sectionStart := start
	index := start
	for index < end {
		ch := units[index]
		if ch < 127 && (charTables[ch]&charMask) != 0 {
			index++
			continue
		}
		var replacement string
		sourceLength := 1
		if ch == percent {
			esc := normalizeEscape(units, index, false)
			switch {
			case esc == nil:
				index += 3
				continue
			case *esc == "%":
				replacement = "%25"
				sourceLength = 1
			default:
				replacement = *esc
				sourceLength = 3
			}
		} else if ch == backslash && replaceBackslash {
			replacement = "/"
		} else if !escapeDelimiters && isGeneralDelimiter(ch) {
			return nil, uriError("Invalid character")
		} else {
			codePoint, length := codePointAt(units, index, end)
			replacement = escapeChar(codePoint)
			sourceLength = length
		}
		if buffer == nil {
			buffer = &strings.Builder{}
		}
		buffer.WriteString(fromUnits(units[sectionStart:index]))
		buffer.WriteString(replacement)
		index += sourceLength
		sectionStart = index
	}
	if buffer == nil {
		return nil, nil
	}
	buffer.WriteString(fromUnits(units[sectionStart:end]))
	s := buffer.String()
	return &s, nil
}

func normalizeOrSubstring(units []uint16, start, end int, charMask uint16, escapeDelimiters, replaceBackslash bool) (string, error) {
	normalized, err := normalize(units, start, end, charMask, escapeDelimiters, replaceBackslash)
	if err != nil {
		return "", err
	}
	if normalized != nil {
		return *normalized, nil
	}
	return fromUnits(units[start:end]), nil
}

func normalizeEscape(units []uint16, index int, lowerCase bool) *string {
	if index+2 >= len(units) {
		s := "%"
		return &s
	}
	firstDigit := units[index+1]
	secondDigit := units[index+2]
	firstValue := hexDigitValue(firstDigit)
	secondValue := hexDigitValue(secondDigit)
	if firstValue < 0 || secondValue < 0 {
		s := "%"
		return &s
	}
	value := uint16(firstValue*16 + secondValue)
	if isUnreservedChar(value) {
		if lowerCase && value >= upperCaseA && value <= upperCaseZ {
			value |= 0x20
		}
		s := string(rune(value))
		return &s
	}
	if firstDigit >= lowerCaseA || secondDigit >= lowerCaseA {
		s := strings.ToUpper(fromUnits(units[index : index+3]))
		return &s
	}
	return nil
}

func codePointAt(units []uint16, index, end int) (rune, int) {
	ch := units[index]
	if ch&0xFC00 == 0xD800 && index+1 < end {
		tail := units[index+1]
		if tail&0xFC00 == 0xDC00 {
			return rune(0x10000 + (int(ch&0x3ff) << 10) + int(tail&0x3ff)), 2
		}
	}
	return rune(ch), 1
}

func escapeChar(ch rune) string {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], ch)
	var b strings.Builder
	for _, by := range buf[:n] {
		b.WriteByte('%')
		b.WriteByte(hexDigits[by>>4])
		b.WriteByte(hexDigits[by&0xf])
	}
	return b.String()
}

func hexDigitValue(ch uint16) int {
	switch {
	case ch >= 0x30 && ch <= 0x39:
		return int(ch - 0x30)
	case ch >= 0x61 && ch <= 0x66:
		return int(ch-0x61) + 10
	case ch >= 0x41 && ch <= 0x46:
		return int(ch-0x41) + 10
	default:
		return -1
	}
}

func isAlphabeticCharacter(ch uint16) bool {
	lower := ch | 0x20
	return lower >= lowerCaseA && lower <= lowerCaseZ
}

func isUnreservedChar(ch uint16) bool {
	return ch < 127 && (charTables[ch]&unreservedMask) != 0
}
func isSchemeCharacter(ch uint16) bool {
	return ch < 128 && (charTables[ch]&schemeMask) != 0
}
func isGeneralDelimiter(ch uint16) bool {
	return ch <= rightBracket && (charTables[ch]&genDelimitersMask) != 0
}
func isRegNameChar(ch uint16) bool {
	return ch < 127 && (charTables[ch]&regNameMask) != 0
}
func isZoneIDChar(ch uint16) bool {
	return ch < 127 && (charTables[ch]&unreservedMask) != 0
}
func isIPvFutureAddressChar(ch uint16) bool {
	if ch >= 127 {
		return true
	}
	return (charTables[ch] & userInfoMask) != 0
}

func defaultPort(scheme string) int {
	switch scheme {
	case "http":
		return 80
	case "https":
		return 443
	default:
		return 0
	}
}

func makePort(port *int, scheme string) *int {
	if port != nil && *port == defaultPort(scheme) {
		return nil
	}
	return port
}

func makeScheme(units []uint16, start, end int) (string, error) {
	if start == end {
		return "", nil
	}
	if !isAlphabeticCharacter(units[start]) {
		return "", uriError("Scheme not starting with alphabetic character")
	}
	for _, ch := range units[start:end] {
		if !isSchemeCharacter(ch) {
			return "", uriError("Illegal scheme character")
		}
	}
	return strings.ToLower(fromUnits(units[start:end])), nil
}

func makeUserInfo(units []uint16, start, end int) (string, error) {
	return normalizeOrSubstring(units, start, end, userInfoMask, false, false)
}

func makeHost(units []uint16, start, end int, strictIPv6 bool) (string, error) {
	if start == end {
		return "", nil
	}
	if units[start] == leftBracket {
		if units[end-1] != rightBracket {
			return "", uriError("Missing end `]` to match `[` in host")
		}
		zoneID := ""
		index := end - 1
		if units[start+1] != lowerCaseV {
			index = checkZoneID(units, start+1, end-1)
			if index < end-1 {
				zoneIDStart := index + 1
				if startsWithAt(units, index+1, "25") {
					zoneIDStart = index + 3
				}
				z, err := normalizeZoneID(units, zoneIDStart, end-1)
				if err != nil {
					return "", err
				}
				zoneID = "%25" + z
			}
		}
		isIPv6, err := validateIPAddress(units, start+1, index)
		if err != nil {
			return "", err
		}
		hostChars := fromUnits(units[start+1 : index])
		if isIPv6 {
			hostChars = strings.ToLower(hostChars)
		}
		return "[" + hostChars + zoneID + "]", nil
	}
	if !strictIPv6 {
		for i := start; i < end; i++ {
			if units[i] == colon {
				zoneID := ""
				index := checkZoneID(units, start, end)
				if index < end {
					zoneIDStart := index + 1
					if startsWithAt(units, index+1, "25") {
						zoneIDStart = index + 3
					}
					z, err := normalizeZoneID(units, zoneIDStart, end)
					if err != nil {
						return "", err
					}
					zoneID = "%25" + z
				}
				if err := parseIPv6Address(fromUnits(units[start:index]), 0, index-start); err != nil {
					return "", err
				}
				return "[" + fromUnits(units[start:index]) + "]" + zoneID, nil
			}
		}
	}
	return normalizeRegName(units, start, end)
}

func startsWithAt(units []uint16, index int, needle string) bool {
	n := toUnits(needle)
	if index < 0 || len(units) < index+len(n) {
		return false
	}
	for i, v := range n {
		if units[index+i] != v {
			return false
		}
	}
	return true
}

func checkZoneID(units []uint16, start, end int) int {
	for i := start; i < end; i++ {
		if units[i] == percent {
			return i
		}
	}
	return end
}

func normalizeZoneID(units []uint16, start, end int) (string, error) {
	var out strings.Builder
	index := start
	for index < end {
		ch := units[index]
		if ch == percent {
			replacement := normalizeEscape(units, index, true)
			switch {
			case replacement == nil:
				out.WriteString(fromUnits(units[index : index+3]))
				index += 3
			case *replacement == "%":
				return "", uriError("ZoneID should not contain % anymore")
			default:
				out.WriteString(*replacement)
				index += 3
			}
		} else if isZoneIDChar(ch) {
			out.WriteRune(rune(ch))
			index++
		} else {
			codePoint, length := codePointAt(units, index, end)
			out.WriteString(escapeChar(codePoint))
			index += length
		}
	}
	return out.String(), nil
}

func normalizeRegName(units []uint16, start, end int) (string, error) {
	var out strings.Builder
	index := start
	for index < end {
		ch := units[index]
		if ch == percent {
			replacement := normalizeEscape(units, index, true)
			switch {
			case replacement == nil:
				out.WriteString(fromUnits(units[index : index+3]))
				index += 3
			case *replacement == "%":
				out.WriteString("%25")
				index++
			default:
				out.WriteString(*replacement)
				index += 3
			}
		} else if isRegNameChar(ch) {
			if ch >= upperCaseA && ch <= upperCaseZ {
				out.WriteByte(byte(ch) + ('a' - 'A'))
			} else {
				out.WriteRune(rune(ch))
			}
			index++
		} else if isGeneralDelimiter(ch) {
			return "", uriError("Invalid character")
		} else {
			codePoint, length := codePointAt(units, index, end)
			out.WriteString(escapeChar(codePoint))
			index += length
		}
	}
	return out.String(), nil
}

func makePath(units []uint16, start, end int, scheme string, hasAuthority bool) (string, error) {
	isFile := scheme == "file"
	ensureLeadingSlash := isFile || hasAuthority
	result, err := normalizeOrSubstring(units, start, end, pathCharOrSlashMask, true, true)
	if err != nil {
		return "", err
	}
	if result == "" {
		if isFile {
			result = "/"
		}
	} else if ensureLeadingSlash && !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	return normalizePathString(result, scheme, hasAuthority), nil
}

func makeQuery(units []uint16, start, end int) (string, error) {
	return normalizeOrSubstring(units, start, end, queryCharMask, true, false)
}
func makeFragment(units []uint16, start, end int) (string, error) {
	return normalizeOrSubstring(units, start, end, queryCharMask, true, false)
}

func normalizePathString(path, scheme string, hasAuthority bool) string {
	if scheme == "" && !hasAuthority && !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, `\`) {
		return normalizeRelativePath(path, false)
	}
	return removeDotSegments(path)
}

func mayContainDotSegments(path string) bool {
	return strings.HasPrefix(path, ".") || strings.Contains(path, "/.")
}

func removeDotSegments(path string) string {
	if !mayContainDotSegments(path) {
		return path
	}
	var output []string
	appendSlash := false
	for _, segment := range strings.Split(path, "/") {
		appendSlash = false
		if segment == ".." {
			if len(output) > 0 {
				output = output[:len(output)-1]
				if len(output) == 0 {
					output = append(output, "")
				}
			}
			appendSlash = true
		} else if segment == "." {
			appendSlash = true
		} else {
			output = append(output, segment)
		}
	}
	if appendSlash {
		output = append(output, "")
	}
	return strings.Join(output, "/")
}

func normalizeRelativePath(path string, allowScheme bool) string {
	if !mayContainDotSegments(path) {
		if allowScheme {
			return path
		}
		return escapeScheme(path)
	}
	var output []string
	appendSlash := false
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			appendSlash = true
			if len(output) > 0 && output[len(output)-1] != ".." {
				output = output[:len(output)-1]
			} else {
				output = append(output, "..")
			}
		} else if segment == "." {
			appendSlash = true
		} else {
			appendSlash = false
			if segment == "" && len(output) == 0 {
				segment = "./"
			}
			output = append(output, segment)
		}
	}
	if len(output) == 0 {
		return "./"
	}
	if appendSlash {
		output = append(output, "")
	}
	if !allowScheme {
		output[0] = escapeScheme(output[0])
	}
	return strings.Join(output, "/")
}

func escapeScheme(path string) string {
	bytes := []byte(path)
	if len(bytes) >= 2 && isASCIIAlpha(bytes[0]) {
		for i := 1; i < len(bytes); i++ {
			if bytes[i] == colon {
				return path[:i] + "%3A" + path[i+1:]
			}
			if bytes[i] > 127 || (charTables[bytes[i]]&schemeMask) == 0 {
				break
			}
		}
	}
	return path
}

func isASCIIAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func simpleMergeRelative(base, reference string) string {
	baseStart := 0
	for strings.HasPrefix(base[baseStart:], "../") {
		baseStart += 3
	}
	refStart := 0
	backCount := 0
	for strings.HasPrefix(reference[refStart:], "../") {
		refStart += 3
		backCount++
	}
	insert := ""
	baseEnd := len(base)
	for baseEnd > baseStart {
		baseEnd--
		if base[baseEnd] == '/' {
			insert = "/"
			if backCount == 0 {
				break
			}
			backCount--
		}
	}
	if baseEnd == baseStart {
		insert = ""
		refStart -= backCount * 3
	}
	return base[:baseEnd] + insert + reference[refStart:]
}

func mergePaths(base, reference string) string {
	backCount := 0
	refStart := 0
	for strings.HasPrefix(reference[refStart:], "../") {
		refStart += 3
		backCount++
	}
	baseEnd := strings.LastIndex(base, "/")
	if baseEnd < 0 {
		baseEnd = -1
	}
	for baseEnd > 0 && backCount > 0 {
		newEnd := strings.LastIndex(base[:baseEnd], "/")
		if newEnd < 0 {
			break
		}
		delta := baseEnd - newEnd
		if (delta == 2 || delta == 3) && base[newEnd+1] == '.' &&
			(delta == 2 || base[newEnd+2] == '.') {
			break
		}
		baseEnd = newEnd
		backCount--
	}
	suffixStart := refStart - 3*backCount
	return base[:baseEnd+1] + reference[suffixStart:]
}

func parseDartInt(text string) (int, bool) {
	text = strings.TrimSpace(text)
	negative := false
	if strings.HasPrefix(text, "+") {
		text = text[1:]
	} else if strings.HasPrefix(text, "-") {
		negative = true
		text = text[1:]
	}
	radix := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		radix = 16
		text = text[2:]
	}
	if text == "" {
		return 0, false
	}
	var value uint64
	for _, ch := range text {
		digit := -1
		switch {
		case ch >= '0' && ch <= '9':
			digit = int(ch - '0')
		case ch >= 'a' && ch <= 'f':
			digit = int(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			digit = int(ch-'A') + 10
		}
		if digit < 0 || digit >= radix {
			return 0, false
		}
		value = value*uint64(radix) + uint64(digit)
		if value > uint64(^uint(0)>>1) {
			return 0, false
		}
	}
	if negative {
		return -int(value), true
	}
	return int(value), true
}
