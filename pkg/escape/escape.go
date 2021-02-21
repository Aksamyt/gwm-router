/*
  This file is part of the uritemplate project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

// Package escape provides functions to escape strings as instructed by
// RFC6570.
package escape

const upperhex = "0123456789ABCDEF"
const truth = "" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"ADADDADDDDDDDBBDBBBBBBBBBBDDADAD" +
	"DBBBBBBBBBBBBBBBBBBBBBBBBBBDADAB" +
	"ABBBBBBBBBBBBBBBBBBBBBBBBBBAAABA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

const (
	// Disallowed characters from RFC6570, i.e. everything not Reserved
	// nor Unreserved.
	Disallowed byte = 1 << iota
	// Unreserved characters from RFC6570:
	//     ALPHA          =  %x41-5A / %x61-7A   ; A-Z / a-z
	//     DIGIT          =  %x30-39             ; 0-9
	//     unreserved     =  ALPHA / DIGIT / "-" / "." / "_" / "~"
	Unreserved
	// Reserved characters from RFC6570:
	//     reserved       =  gen-delims / sub-delims
	//     gen-delims     =  ":" / "/" / "?" / "#" / "[" / "]" / "@"
	//     sub-delims     =  "!" / "$" / "&" / "'" / "(" / ")"
	//                    /  "*" / "+" / "," / ";" / "="
	Reserved
)

// Escape escapes the string, replacing masked characters with %XX sequences
// as needed.
func Escape(s string, mask byte) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if truth[c]&mask != 0 {
			hexCount++
		}
	}

	if hexCount == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	for i, j := 0, 0; i < len(s); i++ {
		c := s[i]
		if truth[c]&mask != 0 {
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&0xF]
			j += 3
		} else {
			t[j] = c
			j++
		}
	}
	return string(t)
}
