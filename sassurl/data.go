package sassurl

// dart-source: sdk/lib/core/uri.dart (UriData) and sdk/lib/convert/base64.dart
// (Dart 3.13.5) — data: URI parsing and normalization.

import "strings"

const (
	commaChar     = 0x2c
	slashChar     = 0x2f
	semicolonChar = 0x3b
	equalsChar    = 0x3d
)

func stripDataScheme(units []uint16) (int, bool) {
	if len(units) < 5 {
		return 0, false
	}
	prefix := []uint16{0x64, 0x61, 0x74, 0x61, 0x3a}
	for i, expected := range prefix {
		if units[i]|0x20 != expected {
			return 0, false
		}
	}
	return 5, true
}

func parseDataURI(units []uint16, start int) (*URL, error) {
	text := fromUnits(units[start:])
	path, err := parseURIData(text)
	if err != nil {
		return nil, err
	}
	var query *string
	if queryIndex := strings.IndexByte(path, '?'); queryIndex >= 0 {
		q, err := normalizeOrSubstring(toUnits(path), queryIndex+1, len(path), queryCharMask, false, false)
		if err != nil {
			return nil, err
		}
		query = &q
		path = path[:queryIndex]
	}
	pathUnits := toUnits(path)
	path, err = normalizeOrSubstring(pathUnits, 0, len(pathUnits), pathCharOrSlashMask, false, false)
	if err != nil {
		return nil, err
	}
	u := &URL{}
	u.make("data", "", nil, nil, path, query, nil)
	return u, nil
}

func parseURIData(text string) (string, error) {
	units := toUnits(text)
	length := len(units)
	separators := []int{-1}
	slashIndex := -1
	i := 0
	var ch uint16
	for i < length {
		ch = units[i]
		if ch == commaChar || ch == semicolonChar {
			break
		}
		if ch == slashChar {
			if slashIndex < 0 {
				slashIndex = i
				i++
				continue
			}
			return "", uriError("Invalid MIME type")
		}
		i++
	}
	if slashIndex < 0 && i > 0 {
		return "", uriError("Invalid MIME type")
	}
	for ch != commaChar {
		separators = append(separators, i)
		i++
		equalsIndex := -1
		for i < length {
			ch = units[i]
			if ch == equalsChar {
				if equalsIndex < 0 {
					equalsIndex = i
				}
			} else if ch == semicolonChar || ch == commaChar {
				break
			}
			i++
		}
		if equalsIndex >= 0 {
			separators = append(separators, equalsIndex)
		} else {
			lastSeparator := separators[len(separators)-1]
			if ch != commaChar || i != lastSeparator+7 || !startsWithAt(units, lastSeparator+1, "base64") {
				return "", uriError("Expecting '='")
			}
			break
		}
	}
	separators = append(separators, i)
	isBase64 := len(separators)%2 == 1
	prefix := fromUnits(units[:i+1])
	if isBase64 {
		tail, err := normalizeBase64(units, i+1, length)
		if err != nil {
			return "", err
		}
		return prefix + tail, nil
	}
	data, err := normalize(units, i+1, length, queryCharMask, true, false)
	if err != nil {
		return "", err
	}
	if data == nil {
		return text, nil
	}
	return prefix + *data, nil
}

func base64Value(ch uint16) (byte, bool) {
	switch {
	case ch >= 0x41 && ch <= 0x5a:
		return byte(ch - 0x41), true
	case ch >= 0x61 && ch <= 0x7a:
		return byte(ch-0x61) + 26, true
	case ch >= 0x30 && ch <= 0x39:
		return byte(ch-0x30) + 52, true
	case ch == 0x2b:
		return 62, true
	case ch == 0x2f:
		return 63, true
	default:
		return 0, false
	}
}

func normalizeBase64(units []uint16, start, end int) (string, error) {
	var out strings.Builder
	padding := 0
	sawPadding := false
	i := start
	for i < end {
		ch := units[i]
		i++
		if ch == percent {
			if i+2 > end {
				return "", uriError("Invalid base64 data")
			}
			hi := hexDigitValue(units[i])
			lo := hexDigitValue(units[i+1])
			if hi < 0 || lo < 0 {
				return "", uriError("Invalid base64 data")
			}
			i += 2
			ch = uint16(hi*16 + lo)
		}
		if ch == 0x2d {
			ch = 0x2b
		} else if ch == 0x5f {
			ch = 0x2f
		}
		if _, ok := base64Value(ch); ok {
			if sawPadding {
				return "", uriError("Invalid base64 data")
			}
			out.WriteByte(byte(ch))
		} else if ch == equalsChar {
			sawPadding = true
			padding++
			out.WriteByte('=')
		} else {
			return "", uriError("Invalid base64 data")
		}
	}
	if sawPadding {
		if out.Len()%4 != 0 || padding > 2 {
			return "", uriError("Invalid base64 data")
		}
		if padding > 0 && !strings.HasSuffix(out.String(), "=") {
			return "", uriError("Invalid base64 data")
		}
	} else {
		for out.Len()%4 != 0 {
			out.WriteByte('=')
		}
	}
	return out.String(), nil
}

// DataURLFromString builds the data: URL Dart uses for an
// ImporterResult.sourceMapUrl when none is supplied:
// Uri.dataFromString(contents, encoding: utf8).
func DataURLFromString(contents string) string {
	var b strings.Builder
	b.WriteString("data:;charset=utf-8,")
	for i := 0; i < len(contents); i++ {
		ch := contents[i]
		if isDataSafe(ch) {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexDigits[ch>>4])
		b.WriteByte(hexDigits[ch&0xf])
	}
	return b.String()
}

func isDataSafe(ch byte) bool {
	if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' {
		return true
	}
	switch ch {
	case '-', '.', '_', '~', '!', '$', '&', '\'', '(', ')', '*', '+', ',', '/', ':', ';', '=', '?', '@':
		return true
	}
	return false
}
