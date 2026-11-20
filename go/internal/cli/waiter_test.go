package cli

import "testing"

// A shorter status line must overwrite the whole previous one, or leftovers
// remain on screen: "...: ACTIVE (attempt 9/90)0)".
func TestProgressLinePadsShorterLines(t *testing.T) {
	prev := 0
	first := progressLine("Waiting: CREATING-BILLING (attempt 10/90)", &prev)
	if first != "\rWaiting: CREATING-BILLING (attempt 10/90)" {
		t.Errorf("first line = %q, want no padding", first)
	}

	second := progressLine("Waiting: ACTIVE (attempt 11/90)", &prev)
	want := "\rWaiting: ACTIVE (attempt 11/90)" + "          " // 10 spaces
	if second != want {
		t.Errorf("second line = %q, want %q", second, want)
	}
	if prev != len("Waiting: ACTIVE (attempt 11/90)") {
		t.Errorf("prevLen = %d, want the unpadded length of the last line", prev)
	}
}
