/*
  This file is part of the uritemplate project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package lexer

import "fmt"

func (l *lexer) error(msg string) stateFn {
	l.items <- Item{ItemError, msg, l.pos}
	return nil
}

func ErrorIllegal(c byte) string {
	return fmt.Sprintf("found illegal character «%c»", c)
}

func ErrorUnfinishedPercent() string {
	return "expected two hex digits"
}

func ErrorIllegalPercent(r rune) string {
	return fmt.Sprintf("%s, got %#U", ErrorUnfinishedPercent(), r)
}

func ErrorUnfinishedExpr() string {
	return "expected '}', got EOF"
}

func ErrorEmptyExpr() string {
	return "empty expression"
}

func ErrorUnexpected(c byte) string {
	return fmt.Sprintf("unexpected %#U", c)
}

func ErrorReservedOp(c byte) string {
	return fmt.Sprintf("unexpected reserved operator %#U", c)
}

func ErrorExpectedLength() string {
	return "expected length"
}
