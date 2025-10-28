package types_test

import (
	"testing"

	"go.innotegrity.dev/types"
)

// TODO: implement additional testing and benchmarks

func TestDuration1(t *testing.T) {
	durations := []string{
		"",
		"3m",
		"4mo",
		"5w",
		"6d",
		"7y",
		"134y",
	}

	for _, str := range durations {
		dur, err := types.ParseDuration(str)
		if err != nil {
			t.Errorf("failed to parse duration: %v", err)
		} else {
			t.Logf("duration: %s", dur)
		}
	}

}
