package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Size is an extended version of a float64 which allows abbreviating sizes by adding a suffix.
//
// A value without a suffix must be an integer and will be treated as a size in bytes. You may also add one of
// the following suffixes after the value to indicate a different measurement:
//
//	b | bytes = size is in bytes
//	k | kb = size is in kilobytes (where 1k = 1000^1 bytes)
//	kib = size is in kibibytes (where 1kib = 1024^1 bytes)
//	m | mb = size is in megabytes (where 1m = 1000^2 bytes)
//	mib = size is in mebibytes (where 1mib = 1024^2 bytes)
//	g | gb = size is in gigabytes (where 1g = 1000^3 bytes)
//	gib = size is in gibibytes (where 1gib = 1024^3 bytes)
//	t | tb = size is in terabytes (where 1g = 1000^4 bytes)
//	tib = size is in tebibytes (where 1gib = 1024^4 bytes)
//	p | pb = size is in petabytes (where 1p = 1000^5 bytes)
//	pib = size is in pebibytes (where 1pib = 1024^5 bytes)
type Size float64

// ParseSize parses the given string into a [Size] object.
//
// If an empty string is supplied, 0 is returned.
func ParseSize(size string) (Size, error) {
	// empty size
	if size == "" {
		return 0, nil
	}

	var parsedSize Size
	sizePattern := regexp.MustCompile(
		`^(\d*\.\d+|\d+\.\d*|\d+)(\s*?)(?i)(b|bytes|k|kb|kib|m|mb|mib|g|gb|gib|t|tb|tib|p|pb|pib)$`)
	matches := sizePattern.FindStringSubmatch(size)
	if matches == nil {
		ival64, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse size '%s': %w", size, err)
		}
		parsedSize = Size(ival64)
		return parsedSize, nil
	}

	// NOTE: we parse as a float since the value may contain a decimal point - any decimal place digits
	// left over will be discarded
	fval64, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size '%s': %w", size, err)
	}
	switch strings.ToLower(matches[3]) {
	case "k", "kb":
		if fval64 > (math.MaxFloat64 / 1000) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1000)
	case "kib":
		if fval64 > (math.MaxFloat64 / 1024) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1024)
	case "m", "mb":
		if fval64 > (math.MaxFloat64 / 1000000) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1000000)
	case "mib":
		if fval64 > (math.MaxFloat64 / 1048576) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1048576)
	case "g", "gb":
		if fval64 > (math.MaxFloat64 / 1000000000) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1000000000)
	case "gib":
		if fval64 > (math.MaxFloat64 / 1073741824) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1073741824)
	case "t", "tb":
		if fval64 > (math.MaxFloat64 / 1000000000000) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1000000000000)
	case "tib":
		if fval64 > (math.MaxFloat64 / 1099511627776) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1099511627776)
	case "p", "pb":
		if fval64 > (math.MaxFloat64 / 1000000000000000) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1000000000000000)
	case "pib":
		if fval64 > (math.MaxFloat64 / 1125899906842624) {
			return 0, errors.New("size exceeds maximum size of a 64-bit integer")
		}
		parsedSize = Size(fval64 * 1125899906842624)
	default:
		parsedSize = Size(fval64)
	}
	return parsedSize, nil
}

// MarshalJSON marshals the [Size] object to JSON.
func (s Size) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// MarshalText marshals the [Size] object to plain text.
func (s Size) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// String returns the [Size] object as a string.
func (s Size) String() string {
	if s < 1000 {
		return fmt.Sprintf("%g bytes", s)
	}
	if s < 1000000 {
		return fmt.Sprintf("%gKB", float64(s)/float64(1000))
	}
	if s < 1000000000 {
		return fmt.Sprintf("%gMB", float64(s)/float64(1000000))
	}
	if s < 1000000000000 {
		return fmt.Sprintf("%gGB", float64(s)/float64(1000000000))
	}
	if s < 1000000000000000 {
		return fmt.Sprintf("%gTB", float64(s)/float64(1000000000000))
	}
	return fmt.Sprintf("%gPB", float64(s)/float64(1000000000000000))
}

// UnmarshalJSON parses the JSON data into a [Size] object.
//
// If an empty string is supplied, 0 is stored.
func (s *Size) UnmarshalJSON(data []byte) error {
	var fval64 float64
	if err := json.Unmarshal(data, &fval64); err == nil {
		*s = Size(fval64)
		return nil
	}

	var sval string
	if err := json.Unmarshal(data, &sval); err != nil {
		return err
	}
	size, err := ParseSize(sval)
	if err != nil {
		return err
	}
	*s = size
	return nil
}

// UnmarshalText parses the text into a [Size] object.
//
// If an empty string is supplied, 0 is stored.
func (s *Size) UnmarshalText(data []byte) error {
	size, err := ParseSize(string(data))
	if err != nil {
		return err
	}
	*s = size
	return nil
}
