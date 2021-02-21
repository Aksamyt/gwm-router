package escape

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

// You never know
func TestTruthIsCorrect(t *testing.T) {
	if len(truth) != 0x100 {
		t.Error("escape.truth must be 256 bytes long")
	}

	allUnreserved := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-._~"
	for i := range allUnreserved {
		c := allUnreserved[i]
		if truth[c]&Unreserved == 0 {
			t.Errorf("byte %#U must be Unreserved", c)
		}
	}

	allReserved := ":/?#[]@!$&'()*+,;="
	for i := range allReserved {
		c := allReserved[i]
		if truth[c]&Reserved == 0 {
			t.Errorf("byte %#U must be Reserved", c)
		}
	}

	for i := 0; i < 0x100; i++ {
		c := byte(i)
		if strings.IndexByte(allReserved, c) > -1 ||
			strings.IndexByte(allUnreserved, c) > -1 {
			continue
		}
		if truth[c]&(Unreserved|Reserved) != 0 {
			t.Errorf("byte %#U must not be Reserved nor Unreserved", c)
		}
	}
}

func TestNoEscape(t *testing.T) {
	unescaped := "Hello.world~"
	escaped := Escape(unescaped, 0)

	if unescaped != escaped {
		t.Errorf("%q should not be transformed", unescaped)
	}

	getStringbuf := func(s *string) uintptr {
		buf := *(*[]byte)(unsafe.Pointer(s))
		return reflect.ValueOf(buf).Pointer()
	}

	if getStringbuf(&unescaped) != getStringbuf(&escaped) {
		t.Errorf("Escape should not allocate a new string")
	}
}

func TestEscape(t *testing.T) {
	for _, tt := range []struct {
		unescaped string
		mask      byte
		expected  string
	}{
		{"Hello World!", Disallowed | Reserved, "Hello%20World%21"},
		{"Hello World!", Disallowed, "Hello%20World!"},
		{
			"Did you ever hear the tragedy of Darth Plagueis The Wise? I thought not. It’s not a story the Jedi would tell you.",
			Disallowed | Reserved,
			"Did%20you%20ever%20hear%20the%20tragedy%20of%20Darth%20Plagueis%20The%20Wise%3F%20I%20thought%20not.%20It%E2%80%99s%20not%20a%20story%20the%20Jedi%20would%20tell%20you.",
		},
	} {
		got := Escape(tt.unescaped, tt.mask)
		if got != tt.expected {
			t.Errorf("got:\n\t%q\nexpected:\n\t%q\ninput:\n\t%q", got, tt.expected, tt.unescaped)
		}
	}
}
