package vm_test

import (
	"strings"
	"testing"

	"github.com/malivvan/rumo/vm"
)

// TestFormatWidthDoS verifies that width and precision specifiers in format
// strings are bounded and cannot be used to trigger large memory allocations.
// Prior to the fix, a format string such as "%999999d" would immediately
// allocate ~1 MB of padding for a single integer without raising any error,
// ignoring both MaxAllocs and MaxStringLen limits on the running VM.  An
// untrusted script could therefore exhaust process memory with a trivially
// short format string.
func TestFormatWidthDoS(t *testing.T) {
	// Width specifier far exceeding MaxFormatWidth must return an error.
	_, err := vm.Format("%999999d", &vm.Int{Value: 1})
	if err == nil {
		t.Error("expected error for width 999999 exceeding MaxFormatWidth, got nil")
	}

	// Precision specifier is subject to the same limit.
	_, err = vm.Format("%.999999f", &vm.Float64{Value: 1.0})
	if err == nil {
		t.Error("expected error for precision 999999 exceeding MaxFormatWidth, got nil")
	}

	// Star-width taken from an argument is also bounded.
	_, err = vm.Format("%*d", &vm.Int{Value: 999999}, &vm.Int{Value: 1})
	if err == nil {
		t.Error("expected error for star-width 999999 exceeding MaxFormatWidth, got nil")
	}
}

// TestFormatWidthNormal verifies that small, legitimate width/precision
// specifiers continue to work correctly after the DoS fix.
func TestFormatWidthNormal(t *testing.T) {
	// Right-pad with width
	s, err := vm.Format("%10d", &vm.Int{Value: 42})
	if err != nil {
		t.Fatalf("unexpected error for %%10d: %v", err)
	}
	if !strings.HasSuffix(s, "42") || len(s) != 10 {
		t.Errorf("%%10d: got %q, want 10-char string ending in '42'", s)
	}

	// Precision on float
	s, err = vm.Format("%.4f", &vm.Float64{Value: 3.14159})
	if err != nil {
		t.Fatalf("unexpected error for %%.4f: %v", err)
	}
	if s != "3.1416" {
		t.Errorf("%%.4f: got %q, want \"3.1416\"", s)
	}

	// Exactly at the default MaxFormatWidth limit should be allowed.
	maxW := vm.DefaultConfig.MaxFormatWidth
	if maxW > 0 {
		fmtStr := "%" + strings.Repeat("", 0) + formatWidthString(maxW) + "d"
		_, err = vm.Format(fmtStr, &vm.Int{Value: 1})
		if err != nil {
			t.Errorf("format with width == MaxFormatWidth (%d) should succeed, got: %v", maxW, err)
		}
	}

	// One over the default limit must fail.
	if maxW > 0 {
		fmtStr := "%" + formatWidthString(maxW+1) + "d"
		_, err = vm.Format(fmtStr, &vm.Int{Value: 1})
		if err == nil {
			t.Errorf("format with width == MaxFormatWidth+1 (%d) should fail", maxW+1)
		}
	}
}

// TestFormatWithConfigMaxFormatWidth verifies that FormatWithConfig respects
// a custom per-call MaxFormatWidth limit independent of DefaultConfig.
func TestFormatWithConfigMaxFormatWidth(t *testing.T) {
	cfg := &vm.Config{
		MaxStringLen:   vm.DefaultConfig.MaxStringLen,
		MaxFormatWidth: 5,
	}

	// Width 6 should exceed the custom limit of 5 and return an error.
	_, err := vm.FormatWithConfig("%6d", cfg, &vm.Int{Value: 1})
	if err == nil {
		t.Error("expected error for width 6 with MaxFormatWidth=5, got nil")
	}

	// Width 5 is right at the limit and should succeed.
	s, err := vm.FormatWithConfig("%5d", cfg, &vm.Int{Value: 1})
	if err != nil {
		t.Fatalf("unexpected error for width 5 with MaxFormatWidth=5: %v", err)
	}
	if len(s) != 5 {
		t.Errorf("expected string of length 5, got %q (len %d)", s, len(s))
	}
}

// formatWidthString converts an integer to its decimal string representation
// without importing strconv (to keep the test self-contained).
func formatWidthString(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

