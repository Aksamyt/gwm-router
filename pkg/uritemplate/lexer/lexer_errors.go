/*
  This file is part of the gwm-router project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package uritemplatelexer

import "fmt"

func (l *lexer) error(msg string) stateFn {
	l.items <- item{itemError, msg}
	return nil
}

func errorIllegal(c byte) string {
	return fmt.Sprintf("found illegal character «%c»", c)
}

func errorUnfinishedPercent() string {
	return "expected two hex digits"
}

func errorIllegalPercent(r rune) string {
	return fmt.Sprintf("%s, got %#U", errorUnfinishedPercent(), r)
}

func errorUnfinishedExpr() string {
	return "expected '}', got EOF"
}

func errorEmptyExpr() string {
	return "empty expression"
}

func errorUnexpected(c byte) string {
	return fmt.Sprintf("unexpected %#U", c)
}

func errorReservedOp(c byte) string {
	return fmt.Sprintf("unexpected reserved operator %#U", c)
}

func errorExpectedLength() string {
	return "expected length"
}
