package types_test

import (
	"testing"

	"go.innotegrity.dev/types"
)

// TODO: implement additional testing and benchmarks

func TestSet1(t *testing.T) {
	requiredLangs := types.NewSet("go", "javascript", "C#")
	t.Logf("required languages: %s", requiredLangs)
	knownLangs := types.NewSet("java", "C++", "go")
	t.Logf("known languages: %s", knownLangs)
	knownLangs.Add("javascript", "python", "shell")
	t.Logf("known languages now: %s", knownLangs)

	t.Logf("matching languages: %s", requiredLangs.Intersection(knownLangs))
	t.Logf("is Python known: %t", knownLangs.Contains("python"))
}
