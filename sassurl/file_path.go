package sassurl

// dart-source: sdk/lib/core/uri.dart (Uri.file / toFilePath) and
// package:path's toUri/fromUri (Dart 3.13.5) — filesystem path conversion.

import (
	"runtime"
	"strings"
)

var windowsPaths = runtime.GOOS == "windows"

// FileURLFromAbsPath converts an absolute filesystem path to a file: URL.
func FileURLFromAbsPath(path string) (*URL, error) {
	return uriFile(preparePath(path, windowsPaths), windowsPaths)
}

// PathToURI converts a filesystem path using package:path's toUri: absolute
// paths become file: URLs, relative paths become relative references.
func PathToURI(path string) (*URL, error) {
	return pathToURIWithOS(path, windowsPaths)
}

func pathToURIWithOS(path string, windows bool) (*URL, error) {
	path = preparePath(path, windows)
	if !isAbsolutePath(path, windows) {
		separator := "/"
		if windows {
			separator = `\`
		}
		segments := strings.Split(path, separator)
		for i, segment := range segments {
			segments[i] = encodeSegment(segment)
		}
		if len(segments) > 0 {
			segments[0] = escapeSchemePath(segments[0])
		}
		u := &URL{}
		u.make("", "", nil, nil, strings.Join(segments, "/"), nil, nil)
		return u, nil
	}
	if windows {
		u := windowsToURI(path)
		if u == nil {
			return nil, uriError("invalid file path")
		}
		return u, nil
	}
	return uriFile(path, false)
}

// windowsToURI is package:path's toUri for an absolute Windows path. Unlike
// Uri.file, toUri does not reject characters Windows forbids: it encodes the
// path segments as written.
func windowsToURI(path string) *URL {
	fileURL := func(host, p string) *URL {
		u := &URL{}
		u.make("file", "", strPtr(host), nil, p, nil, nil)
		return u
	}
	if strings.HasPrefix(path, `\\`) {
		rest := path[2:]
		pathStart := strings.IndexByte(rest, '\\')
		var host, pathPart string
		if pathStart >= 0 {
			host = rest[:pathStart]
			pathPart = rest[pathStart+1:]
		} else {
			host = rest
		}
		var segments []string
		if pathPart != "" {
			segments = strings.Split(pathPart, `\`)
		}
		encoded, _ := encodeSegments(segments)
		if encoded != "" && !strings.HasPrefix(encoded, "/") {
			encoded = "/" + encoded
		}
		return fileURL(host, encoded)
	}
	if len(path) >= 3 && isASCIIAlpha(path[0]) && path[1] == ':' {
		drive := path[:2]
		segments := strings.Split(path[3:], `\`)
		encoded, _ := encodeSegments(segments)
		p := "/" + drive + "/"
		if encoded != "" {
			p += encoded
		}
		return fileURL("", p)
	}
	if strings.HasPrefix(path, `\`) {
		rest := path[1:]
		if len(rest) >= 2 && isASCIIAlpha(rest[0]) && rest[1] == ':' {
			drive := rest[:2]
			tail := strings.TrimPrefix(rest[2:], `\`)
			var segments []string
			if tail != "" {
				segments = strings.Split(tail, `\`)
			}
			encoded, _ := encodeSegments(segments)
			p := "/" + drive + "/"
			if encoded != "" {
				p += encoded
			}
			return fileURL("", p)
		}
		segments := strings.Split(rest, `\`)
		encoded, _ := encodeSegments(segments)
		return fileURL("", "/"+encoded)
	}
	// Unreachable for an absolute Windows path after preparePath; keep the
	// relative form as a safe fallback.
	return nil
}

func preparePath(path string, windows bool) string {
	if windows {
		if strings.HasPrefix(path, `\\?\`) {
			stripped := path[4:]
			if strings.HasPrefix(stripped, `UNC\`) {
				path = `\\` + stripped[4:]
			} else {
				path = stripped
			}
		}
		path = strings.ReplaceAll(path, "/", `\`)
	}
	return normalizeFilePath(path, windows)
}

func normalizeFilePath(path string, windows bool) string {
	separator := "/"
	if windows {
		separator = `\`
	}
	var prefix, rest string
	if windows {
		switch {
		case strings.HasPrefix(path, `\\`):
			parts := strings.SplitN(path[2:], `\`, 3)
			server := ""
			share := ""
			if len(parts) > 0 {
				server = parts[0]
			}
			if len(parts) > 1 {
				share = parts[1]
			}
			if len(parts) > 2 {
				rest = parts[2]
			}
			prefix = `\\` + server + `\` + share
		case len(path) >= 2 && path[1] == ':':
			prefix = path[:2]
			if len(path) > 2 && path[2] == '\\' {
				prefix += `\`
				rest = path[3:]
			} else {
				rest = path[2:]
			}
		case strings.HasPrefix(path, `\`):
			prefix = `\`
			rest = path[1:]
		default:
			rest = path
		}
	} else if strings.HasPrefix(path, "/") {
		prefix = "/"
		rest = path[1:]
	} else {
		rest = path
	}

	trailing := strings.HasSuffix(rest, separator)
	var stack []string
	for _, segment := range strings.Split(rest, separator) {
		if segment == "" || segment == "." {
			continue
		}
		if segment == ".." {
			if len(stack) > 0 && stack[len(stack)-1] != ".." {
				stack = stack[:len(stack)-1]
				continue
			}
			if prefix == "" {
				stack = append(stack, "..")
			}
		} else {
			stack = append(stack, segment)
		}
	}
	result := prefix
	if result != "" && !strings.HasSuffix(result, separator) && len(stack) > 0 {
		result += separator
	}
	result += strings.Join(stack, separator)
	if trailing && len(stack) > 0 && !strings.HasSuffix(result, separator) {
		result += separator
	}
	return result
}

func escapeSchemePath(path string) string {
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		return escapeScheme(path[:idx]) + "/" + path[idx+1:]
	}
	return escapeScheme(path)
}

func isAbsolutePath(path string, windows bool) bool {
	if windows {
		if strings.HasPrefix(path, `\`) || strings.HasPrefix(path, "/") {
			return true
		}
		return len(path) >= 3 && isASCIIAlpha(path[0]) && path[1] == ':' &&
			(path[2] == '\\' || path[2] == '/')
	}
	return strings.HasPrefix(path, "/")
}

func uriFile(path string, windows bool) (*URL, error) {
	if windows {
		return windowsFileURL(path)
	}
	segments := strings.Split(path, "/")
	if strings.HasPrefix(path, "/") {
		encoded, ok := encodeSegments(segments)
		if !ok {
			return nil, uriError("invalid file path")
		}
		u := &URL{}
		u.make("file", "", strPtr(""), nil, encoded, nil, nil)
		return u, nil
	}
	encoded, ok := encodeSegments(segments)
	if !ok {
		return nil, uriError("invalid file path")
	}
	u := &URL{}
	u.make("", "", nil, nil, escapeSchemePath(encoded), nil, nil)
	return u, nil
}

func windowsFileURL(path string) (*URL, error) {
	if strings.HasPrefix(path, `\\?\`) {
		stripped := path[4:]
		if strings.HasPrefix(stripped, `UNC\`) {
			path = `\` + stripped[4:]
		} else {
			path = stripped
			if len(path) < 3 || path[1] != ':' || path[2] != '\\' {
				return nil, uriError("invalid file path")
			}
		}
	}

	if len(path) > 1 && path[1] == ':' {
		if !isASCIIAlpha(path[0]) {
			return nil, uriError("invalid file path")
		}
		if len(path) == 2 || path[2] != '\\' {
			return nil, uriError("invalid file path")
		}
		segments := strings.Split(path, `\`)
		if !checkWindowsSegments(segments[1:]) {
			return nil, uriError("invalid file path")
		}
		encoded, ok := encodeSegments(segments)
		if !ok {
			return nil, uriError("invalid file path")
		}
		if !strings.HasPrefix(encoded, "/") {
			encoded = "/" + encoded
		}
		u := &URL{}
		u.make("file", "", strPtr(""), nil, encoded, nil, nil)
		return u, nil
	}

	if strings.HasPrefix(path, `\`) {
		rest := path[1:]
		if strings.HasPrefix(rest, `\`) {
			unc := rest[1:]
			pathStart := strings.IndexByte(unc, '\\')
			var host, pathPart string
			if pathStart >= 0 {
				host = unc[:pathStart]
				pathPart = unc[pathStart+1:]
			} else {
				host = unc
			}
			var segments []string
			if pathPart != "" {
				segments = strings.Split(pathPart, `\`)
			}
			if !checkWindowsSegments(segments) {
				return nil, uriError("invalid file path")
			}
			encoded, ok := encodeSegments(segments)
			if !ok {
				return nil, uriError("invalid file path")
			}
			if encoded != "" && !strings.HasPrefix(encoded, "/") {
				encoded = "/" + encoded
			}
			u := &URL{}
			u.make("file", "", strPtr(host), nil, encoded, nil, nil)
			return u, nil
		}
		segments := strings.Split(path, `\`)
		if !checkWindowsSegments(segments) {
			return nil, uriError("invalid file path")
		}
		encoded, ok := encodeSegments(segments)
		if !ok {
			return nil, uriError("invalid file path")
		}
		u := &URL{}
		u.make("file", "", strPtr(""), nil, encoded, nil, nil)
		return u, nil
	}

	segments := strings.Split(path, `\`)
	if !checkWindowsSegments(segments) {
		return nil, uriError("invalid file path")
	}
	encoded, ok := encodeSegments(segments)
	if !ok {
		return nil, uriError("invalid file path")
	}
	u := &URL{}
	u.make("", "", nil, nil, escapeSchemePath(encoded), nil, nil)
	return u, nil
}

func checkWindowsSegments(segments []string) bool {
	for _, segment := range segments {
		for i := 0; i < len(segment); i++ {
			switch segment[i] {
			case '"', '*', '/', ':', '<', '>', '?', '\\', '|':
				return false
			}
		}
	}
	return true
}

func encodeSegments(segments []string) (string, bool) {
	var b strings.Builder
	for i, segment := range segments {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(encodeSegment(segment))
	}
	return b.String(), true
}

func encodeSegment(segment string) string {
	var b strings.Builder
	for i := 0; i < len(segment); i++ {
		ch := segment[i]
		if isPathChar(ch) {
			b.WriteByte(ch)
		} else {
			b.WriteByte('%')
			b.WriteByte(hexDigits[ch>>4])
			b.WriteByte(hexDigits[ch&0xf])
		}
	}
	return b.String()
}

func isPathChar(ch byte) bool {
	if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
		return true
	}
	switch ch {
	case '-', '.', '_', '~', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', ':', '@':
		return true
	}
	return false
}

// ToFilePathString converts a file: URL to a filesystem path.
func (u *URL) ToFilePathString() (string, bool) {
	if !u.IsFile() {
		return "", false
	}
	return uriToPath(u.Path, hostPtr(u), windowsPaths)
}

// FSPath returns the filesystem path for a file: URL, falling back to the URL
// path when conversion fails.
func (u *URL) FSPath() string {
	if p, ok := u.ToFilePathString(); ok {
		return p
	}
	return u.Path
}

func uriToPath(path string, host *string, windows bool) (string, bool) {
	decoded, ok := percentDecode(path)
	if !ok {
		return "", false
	}
	if windows {
		if host != nil && *host != "" {
			return `\\` + *host + strings.ReplaceAll(decoded, "/", `\`), true
		}
		if len(decoded) >= 3 && decoded[0] == '/' && decoded[2] == ':' {
			return strings.ReplaceAll(decoded[1:], "/", `\`), true
		}
		return strings.ReplaceAll(decoded, "/", `\`), true
	}
	return decoded, true
}

// DecodeRelativePath decodes percent escapes in a relative path.
func DecodeRelativePath(path string) string {
	if decoded, ok := percentDecode(path); ok {
		return decoded
	}
	return path
}

func percentDecode(path string) (string, bool) {
	bytes := []byte(path)
	out := make([]byte, 0, len(bytes))
	for i := 0; i < len(bytes); {
		if bytes[i] == '%' {
			if i+2 >= len(bytes) {
				return "", false
			}
			hi, ok1 := hexValue(bytes[i+1])
			lo, ok2 := hexValue(bytes[i+2])
			if !ok1 || !ok2 {
				return "", false
			}
			out = append(out, hi<<4|lo)
			i += 3
		} else {
			out = append(out, bytes[i])
			i++
		}
	}
	return string(out), true
}

func hexValue(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}

// ResolveFilePath resolves a file-like URL against a file: base.
func ResolveFilePath(u, base *URL) (*URL, bool) {
	if !u.IsFileLike() || !base.IsFile() {
		return nil, false
	}
	if u.IsRelative() {
		return base.Resolve(u), true
	}
	return u, true
}
