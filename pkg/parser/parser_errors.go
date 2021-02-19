/*
  This file is part of the gwm-router project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package parser

import (
	"fmt"
	"uritemplate/pkg/lexer"
)

type Error struct {
	Err   error
	Input string
	Pos   int
}

func (e Error) Unwrap() error {
	return e.Err
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
