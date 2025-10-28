package types

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Duration is an extended version of [time.Duration] which adds the ability to include new durations such as
// months, weeks, days and years using the suffixes "mo", "w", "d" and "y" respectively.
//
// A month is defined as 30 days, a week is defined as 7 days, a day is defined as 24 hours and a year is
// defined as 365 days.
//
// It also supports unmarshaling empty strings to an empty [Duration] object.
type Duration time.Duration

func ParseDuration(dur string) (Duration, error) {
	// empty duration
	if dur == "" {
		return Duration(0), nil
	}

	// convert new suffixes to supported [time.Duration] suffix
	var hourPeriod int64
	if strings.HasSuffix(dur, "mo") {
		dur = strings.TrimSuffix(dur, "mo")
		hourPeriod = int64(30 /* days */ * 24 /* hours */)
	} else if strings.HasSuffix(dur, "w") {
		dur = strings.TrimSuffix(dur, "w")
		hourPeriod = int64(7 /* days */ * 24 /* hours */)
	} else if strings.HasSuffix(dur, "d") {
		dur = strings.TrimSuffix(dur, "d")
		hourPeriod = int64(24 /* hours */)
	} else if strings.HasSuffix(dur, "y") {
		dur = strings.TrimSuffix(dur, "y")
		hourPeriod = int64(365 /* days */ * 24 /* hours */)
	}
	if hourPeriod > 0 {
		val, err := strconv.ParseInt(dur, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse integer portion of duration '%s': %w", dur, err)
		}
		if val > math.MaxInt64/hourPeriod {
			return 0, fmt.Errorf("duration '%s' exceeds maximum size of a 64-bit integer", dur)
		}
		dur = fmt.Sprintf("%dh", val*hourPeriod)
	}

	// parse duration by hours
	parsedDuration, err := time.ParseDuration(dur)
	return Duration(parsedDuration), err
}

// MarshalJSON marshals the [Duration] object to JSON.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// MarshalText marshals the [Duration] object to plain text.
func (d Duration) MarshalText() ([]byte, error) {
	return json.Marshal(d.String())
}

// String returns the [Duration] object as a string.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// UnmarshalJSON parses the JSON data into a [Duration] object.
//
// If an empty string is supplied, 0 is stored.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var ival64 int64
	if err := json.Unmarshal(data, &ival64); err == nil {
		*d = Duration(ival64)
		return nil
	}

	var sval string
	if err := json.Unmarshal(data, &sval); err != nil {
		return err
	}
	dur, err := ParseDuration(sval)
	if err != nil {
		return err
	}
	*d = dur
	return nil
}

// UnmarshalText parses the text into a [Duration] object.
//
// If an empty string is supplied, 0 is stored.
func (d *Duration) UnmarshalText(data []byte) error {
	dur, err := ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = dur
	return nil
}
