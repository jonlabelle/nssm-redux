package support

import (
	"reflect"
	"testing"
)

func TestMergeEnvironment(t *testing.T) {
	t.Parallel()

	base := []string{"A=1", "B=2"}
	override := []string{"X=9"}
	extra := []string{"B=3", "Y=4"}

	got := MergeEnvironment(base, override, extra)
	want := []string{"X=9", "B=3", "Y=4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeEnvironment() = %#v, want %#v", got, want)
	}
}
