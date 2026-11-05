package imgresize

import (
	"testing"
)

func TestResolveSizePercent(t *testing.T) {
	w, h, err := resolveSize("percent", 0, 0, 50, true, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 400 || h != 300 {
		t.Fatalf("50%% of 800x600 = %dx%d, want 400x300", w, h)
	}

	// Rounding: 33% of 100x33 = 33x10 (33*33/100 = 10.89 → 11? no: floor via int math)
	w, h, err = resolveSize("percent", 0, 0, 33, true, 100, 33)
	if err != nil {
		t.Fatal(err)
	}
	if w != 33 || h != 10 {
		t.Fatalf("33%% of 100x33 = %dx%d, want 33x10", w, h)
	}

	if _, _, err = resolveSize("percent", 0, 0, 0, true, 800, 600); err == nil {
		t.Fatal("0% accepted")
	}
	if _, _, err = resolveSize("percent", 0, 0, 401, true, 800, 600); err == nil {
		t.Fatal("401% accepted")
	}
}

func TestResolveSizeExact(t *testing.T) {
	// Both axes given: exact.
	w, h, err := resolveSize("exact", 320, 240, 0, false, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 320 || h != 240 {
		t.Fatalf("exact = %dx%d, want 320x240", w, h)
	}

	// Locked with one axis: derive the other from the source ratio.
	w, h, err = resolveSize("exact", 400, 0, 0, true, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 400 || h != 300 {
		t.Fatalf("locked width 400 = %dx%d, want 400x300", w, h)
	}

	w, h, err = resolveSize("exact", 0, 150, 0, true, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 200 || h != 150 {
		t.Fatalf("locked height 150 = %dx%d, want 200x150", w, h)
	}

	// Rounding on the derived axis: 901 wide of 300x200 → 601 (rounded, not 600).
	w, h, err = resolveSize("exact", 901, 0, 0, true, 300, 200)
	if err != nil {
		t.Fatal(err)
	}
	if w != 901 || h != 601 {
		t.Fatalf("locked width 901 = %dx%d, want 901x601", w, h)
	}

	// Unlocked with one axis: the other stays 0 → resolved to keep-size.
	w, h, err = resolveSize("exact", 400, 0, 0, false, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 0 || h != 0 {
		t.Fatalf("unlocked partial = %dx%d, want 0x0 (keep source)", w, h)
	}

	// Negative input rejected.
	if _, _, err = resolveSize("exact", -1, 0, 0, true, 800, 600); err == nil {
		t.Fatal("negative width accepted")
	}
}

func TestResolveSizeKeepSource(t *testing.T) {
	// All-zero exact request means keep the source size (0x0 → keep).
	w, h, err := resolveSize("exact", 0, 0, 0, true, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 0 || h != 0 {
		t.Fatalf("keep-size = %dx%d, want 0x0", w, h)
	}
}

func TestRoundRatio(t *testing.T) {
	cases := []struct {
		base, factor, value, want int
	}{
		{600, 800, 400, 300},
		{200, 300, 901, 601},
		{33, 100, 33, 11},
		{0, 0, 5, 0},
	}
	for _, c := range cases {
		if got := roundRatio(c.base, c.factor, c.value); got != c.want {
			t.Fatalf("roundRatio(%d,%d,%d) = %d, want %d", c.base, c.factor, c.value, got, c.want)
		}
	}
}
