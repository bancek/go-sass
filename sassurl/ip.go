package sassurl

import "fmt"

// dart-source: sdk/lib/core/uri.dart (IPv4/IPv6/IPvFuture validation; Dart 3.13.5)

func validateIPAddress(units []uint16, start, end int) (bool, error) {
	if start == end {
		return false, uriError("Empty IP address")
	}
	if units[start] == lowerCaseV {
		if err := validateIPvFuture(units, start, end); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := parseIPv6Address(fromUnits(units[start:end]), 0, end-start); err != nil {
		return false, err
	}
	return true, nil
}

func parseIPv6Address(host string, start, end int) error {
	units := toUnits(host)
	fail := func(message string) error {
		return uriError(fmt.Sprintf("Illegal IPv6 address, %s", message))
	}
	if end-start < 2 {
		return fail("address is too short")
	}
	partStart := start
	wildcardAt := -1
	partCount := 0
	decValue := true
	cursor := start
	if units[cursor] == colon {
		if units[cursor+1] == colon {
			wildcardAt = 0
			partCount = 1
			cursor += 2
			partStart = cursor
		} else {
			return fail("invalid start colon")
		}
	}
	for {
		var ch uint16
		if cursor >= end {
			ch = 0
		} else {
			ch = units[cursor]
		}
		var digit int
		if ch >= digit0 && ch <= digit0+9 {
			digit = int(ch - digit0)
		} else {
			lower := ch | 0x20
			if lower >= lowerCaseA && lower <= lowerCaseF {
				decValue = false
				digit = int(lower-lowerCaseA) + 10
			} else {
				digit = -1
			}
		}
		if digit >= 0 {
			if cursor < partStart+4 {
				cursor++
				continue
			}
			return fail("an IPv6 part can contain a maximum of 4 hex digits")
		}
		if cursor > partStart {
			if ch == dot {
				if decValue {
					if partCount <= 6 {
						if err := parseIPv4Address(host, partStart, end); err != nil {
							return err
						}
						cursor = end
						partCount += 2
						break
					}
					return fail("an address must contain at most 8 parts")
				}
				break
			}
			partCount++
			if ch == colon {
				if partCount < 8 {
					decValue = true
					cursor++
					partStart = cursor
					continue
				}
				return fail("an address must contain at most 8 parts")
			}
			break
		}
		if ch == colon {
			if wildcardAt < 0 {
				wildcardAt = partCount
				partCount++
				cursor++
				partStart = cursor
				continue
			}
			return fail("only one wildcard `::` is allowed")
		}
		if wildcardAt != partCount-1 {
			return fail("missing part")
		}
		break
	}
	if cursor < end {
		return fail("invalid character")
	}
	if partCount < 8 && wildcardAt < 0 {
		return fail("an address without a wildcard must contain exactly 8 parts")
	}
	return nil
}

func parseIPv4Address(host string, start, end int) error {
	units := toUnits(host)
	fail := func(message string) error {
		return uriError(fmt.Sprintf("Illegal IPv4 address, %s", message))
	}
	cursor := start
	octetStart := cursor
	octetIndex := 0
	octetValue := 0
	for {
		var ch uint16
		if cursor >= end {
			ch = 0
		} else {
			ch = units[cursor]
		}
		digit := int(ch) - digit0
		if digit >= 0 && digit <= 9 {
			if octetValue != 0 || cursor == octetStart {
				octetValue = octetValue*10 + digit
				if octetValue <= 255 {
					cursor++
					continue
				}
				return fail("each part must be in the range 0..255")
			}
			return fail("parts must not have leading zeros")
		}
		if cursor == octetStart {
			if cursor == end {
				break
			}
			return fail("invalid character")
		}
		octetIndex++
		if ch == dot {
			if octetIndex < 4 {
				octetValue = 0
				cursor++
				octetStart = cursor
				continue
			}
			break
		}
		if cursor == end {
			if octetIndex == 4 {
				return nil
			}
			break
		}
		return fail("invalid character")
	}
	return fail("IPv4 address should contain exactly 4 parts")
}

func validateIPvFuture(units []uint16, start, end int) error {
	hexStart := start + 1
	cursor := hexStart
	for {
		if cursor < end {
			ch := units[cursor]
			cursor++
			if hexDigitValue(ch) >= 0 {
				continue
			}
			if ch == dot {
				if cursor-1 == hexStart {
					return uriError("Missing hex-digit in IPvFuture address")
				}
				break
			}
			return uriError("Unexpected character")
		}
		if cursor-1 == hexStart {
			return uriError("Missing hex-digit in IPvFuture address")
		}
		return uriError("Missing '.' in IPvFuture address")
	}
	if cursor == end {
		return uriError("Missing address in IPvFuture address, host, cursor")
	}
	for _, ch := range units[cursor:end] {
		if !isIPvFutureAddressChar(ch) {
			return uriError("Invalid IPvFuture address character")
		}
	}
	return nil
}
