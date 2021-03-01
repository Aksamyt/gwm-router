/*
  This file is part of the uritemplate project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package parser

import (
	"fmt"

	"github.com/aksamyt/uritemplate/pkg/lexer"
)

// Error represents a parser. Its Error() method provides a visual explanation
// of where the error occured.
type Error struct {
	Err   error
	Input string
	Pos   int
}

func (e Error) Error() string {
	return fmt.Sprintf(
		`error at col %d: %v
%s
% *s`,
		e.Pos+1, e.Err,
		e.Input,
		e.Pos+1, "^",
	)
}

// LexerError wraps a lexer.ItemError.
type LexerError struct {
	Item lexer.Item
}

func (e LexerError) Error() string {
	return e.Item.Val
}

// A SimpleError does not need any context.
type SimpleError int

const (
	// DoubleModError is returned when more than one modifier is parsed.
	DoubleModError SimpleError = iota
	// ExpectedVarError is returned when a variable was expected.
	ExpectedVarError
	// AfterVarError is usually returned when a comma or a dot is missing.
	AfterVarError
	// LengthOver9999Error is returned when at least five digits are given.
	LengthOver9999Error
)

func (e SimpleError) Error() (what string) {
	switch e {
	case DoubleModError:
		what = "only 1 modifier allowed, expected '}' or ','"
	case ExpectedVarError:
		what = "expected variable"
	case AfterVarError:
		what = "expected '}', '.', or ','"
	case LengthOver9999Error:
		what = "length must be between 0 and 9999"
	}
	return
}

// UnimplementedError signals an illegal state in the parser.
//
// Please open an issue at https://github.com/Aksamyt/uritemplate
// if you see one :)
type UnimplementedError struct {
	Item  lexer.Item
	State string
}

func (e UnimplementedError) Error() string {
	return fmt.Sprintf(`undefined state!
current state: %s
current item: %v`,
		e.State,
		e.Item,
	)
}
