/*
  This file is part of the gwm-router project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package lexer

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type itemType int

const (
	itemError itemType = iota

	itemSep
	itemLacc
	itemRacc
	itemOp
	itemExplode
	itemPrefix
	itemLength
	itemDot
	itemComma
	itemRaw
	itemVar

	itemEOF
)

type item struct {
	typ itemType
	val string
}

func (i item) String() string {
	switch i.typ {
	case itemError:
		return fmt.Sprintf("ERROR %s", i.val)
	case itemSep:
		return "/"
	case itemLacc:
		return "{"
	case itemRacc:
		return "}"
	case itemOp:
		return i.val
	case itemExplode:
		return "*"
	case itemPrefix:
		return ":"
	case itemLength:
		return i.val
	case itemDot:
		return "."
	case itemComma:
		return ","
	case itemRaw:
		return fmt.Sprintf("%q", i.val)
	case itemVar:
		return fmt.Sprintf("'%s'", i.val)
	case itemEOF:
		return "EOF"
	}
	return ""
}

type lexer struct {
	input string
	start int
	pos   int
	items chan item
}

func lex(input string) *lexer {
	l := &lexer{
		input: input,
		start: 0,
		pos:   0,
		items: make(chan item),
	}
	go l.run()
	return l
}

func (l *lexer) eof() bool {
	return l.pos >= len(l.input)
}

func (l *lexer) peek() (byte, bool) {
	if l.eof() {
		return 0, true
	}
	return l.input[l.pos], false
}

func (l *lexer) next() (byte, bool) {
	c, eof := l.peek()
	if !eof {
		l.pos++
	}
	return c, eof
}

func (l *lexer) emit(typ itemType) {
	l.items <- item{typ, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) emitRaw(s string) {
	l.items <- item{itemRaw, s}
	l.start = l.pos
}

func (l *lexer) run() {
	for state := lexPath; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func isVarchar(c byte) bool {
	return false ||
		c >= 'a' && c <= 'z' ||
		c >= 'A' && c <= 'Z' ||
		c >= '0' && c <= '9' ||
		c == '_'
}

type stateFn func(*lexer) stateFn

// lexPath is the entrypoint
func lexPath(l *lexer) stateFn {
	c, eof := l.next()
	if eof {
		l.emit(itemEOF)
		return nil
	}
	switch c {
	case '/':
		l.emit(itemSep)
		return lexPath
	case '%':
		return lexPercent
	case '{':
		l.emit(itemLacc)
		return lexBeginExpr
	default:
		l.pos--
		return lexRaw
	}
}

// lexRaw scans a literal path part.
//
// - l.pos is at the beginning of the part
//
// - l.pos is at index 0 or after any of `}`, `/`, or percent-encoded
//
// - undefined behaviour if l.eof()
func lexRaw(l *lexer) stateFn {
	limit := strings.IndexAny(l.input[l.pos:], "/{%") + l.pos
	if limit < l.pos {
		limit = len(l.input)
	}
	for l.pos < limit {
		c, _ := l.next()
		if c <= ' ' || strings.IndexByte(`"'<>\^|}`+"`", c) != -1 {
			return l.error(errorIllegal(c))
		}
	}
	l.emit(itemRaw)
	return lexPath
}

// lexPercent scans a percent-encoded character.
//
// - l.pos is after the `%` sign
func lexPercent(l *lexer) stateFn {
	l.pos += 2
	if l.pos > len(l.input) {
		return l.error(errorUnfinishedPercent())
	}
	decoded, err := hex.DecodeString(l.input[l.pos-2 : l.pos])
	if err != nil {
		// We checked for hex.ErrLength earlier
		e, _ := err.(hex.InvalidByteError)
		return l.error(errorIllegalPercent(rune(e)))
	}
	l.emitRaw(string(decoded))
	return lexPath
}

// lexBeginExpr scans an identifier, or an operator if present.
//
// - l.pos is after the `{` delimiter
func lexBeginExpr(l *lexer) stateFn {
	c, eof := l.peek()
	switch {
	case eof:
		return l.error(errorUnfinishedExpr())
	case c == '}':
		return l.error(errorEmptyExpr())
	case isVarchar(c):
		return lexInExpr
	case strings.IndexByte("+#./;?&", c) != -1:
		l.pos++
		l.emit(itemOp)
		return lexInExpr
	case strings.IndexByte("=,!@|", c) != -1:
		return l.error(errorReservedOp(c))
	default:
		return l.error(errorUnexpected(c))
	}
}

// lexInExpr scans elements inside an expression until the `}` delimiter.
//
// - l.pos is after the `{` delimiter, or after another expression item
func lexInExpr(l *lexer) stateFn {
	for {
		c, eof := l.next()
		switch {
		case eof:
			return l.error(errorUnfinishedExpr())
		case c == '}':
			l.emit(itemRacc)
			return lexPath
		case c == '.':
			l.emit(itemDot)
		case c == ',':
			l.emit(itemComma)
		case isVarchar(c):
			// l.peek() return (0, false) at l.eof()
			for c, _ := l.peek(); isVarchar(c); c, _ = l.peek() {
				l.pos++
			}
			l.emit(itemVar)
		case c == '*':
			l.emit(itemExplode)
		case c == ':':
			l.emit(itemPrefix)
			return lexLength
		default:
			return l.error(errorUnexpected(c))
		}
	}
}

// lexLength scans at most and 4 ascii digits.
func lexLength(l *lexer) stateFn {
	for {
		// l.peek() return (0, false) at l.eof()
		c, _ := l.peek()
		if c < '0' || c > '9' || l.pos > l.start+3 {
			if l.pos == l.start {
				return l.error(errorExpectedLength())
			}
			l.emit(itemLength)
			return lexInExpr
		}
		l.pos++
	}
}
