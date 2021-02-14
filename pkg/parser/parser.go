/*
  This file is part of the gwm-router project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

// Package parser provides the Parse function and the AST type.
package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"uritemplate/pkg/lexer"
)

// Mod is a tag representing the modifier to apply, if any.
type Mod int

// Modifier tags
const (
	// No modifier
	ModNone Mod = 0
	// Prefix modifier ':' (real value from (1<<14)+0 to (1<<14)+9999)
	ModPrefix Mod = 1 << 14
	// Explode modifier '*'
	ModExplode Mod = 1 << 15
)

// Var represents a variable with its modifier.
type Var struct {
	ID  string
	Mod Mod
}

// Expr represents an expression with a variable list and an operator.
// If no operator was parsed, Op is '\0'.
type Expr struct {
	Op   byte
	Vars []Var
}

func (e Expr) String() string {
	var s strings.Builder
	s.WriteByte('{')
	if e.Op > 0 {
		s.WriteByte(e.Op)
	}
	for i, v := range e.Vars {
		if i > 0 {
			s.WriteByte(',')
		}
		s.WriteString(v.ID)
		if v.Mod&ModPrefix != 0 {
			s.WriteByte(':')
			s.WriteString(strconv.Itoa(int(v.Mod ^ ModPrefix)))
		}
		if v.Mod&ModExplode != 0 {
			s.WriteByte('*')
		}
	}
	s.WriteByte('}')
	return s.String()
}

// Ast represents the parsed result of an uritemplate.
//
// Variables are listed in a separate slice for easy analysis.
//
// Parts are stored as a slice of interfaces. Path separators '/' are stored
// as nil elements, raw parts as strings, and expressions as Expr.
type Ast struct {
	// Variable names used in the parts.
	Vars map[string]struct{}
	// nil, string, or Expr.
	Parts []interface{}
}

func (t Ast) String() string {
	vars := []string(nil)
	for v := range t.Vars {
		vars = append(vars, v)
	}
	sort.Strings(vars)
	var parts []string
	for _, p := range t.Parts {
		switch p := p.(type) {
		case nil:
			parts = append(parts, "/")
		case string:
			parts = append(parts, fmt.Sprintf("%q", p))
		case Expr:
			parts = append(parts, fmt.Sprintf("%v", p))
		}
	}
	return fmt.Sprintf("VARS: %v\n%v", vars, parts)
}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// Parser states
const (
	sBeginRaw = iota
	sRaw
	sAfterLacc
	sAfterVar
	sExpectVar
	sExpectLength
	sAfterMod
	sVar

	sMax
)

// Parse an URI template.
func Parse(input string) (t Ast, err error) {
	t.Vars = map[string]struct{}{}
	var expr Expr
	var v Var
	var raw strings.Builder
	state := sBeginRaw

loop:
	for item := range lexer.Lex(input) {
		switch int(item.Typ)*sMax + state {
		case sRaw + sMax*int(lexer.ItemEOF):
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemEOF):
			if raw.Len() > 0 {
				t.Parts = append(t.Parts, raw.String())
			}
			break loop

		case sRaw + sMax*int(lexer.ItemRaw):
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemRaw):
			raw.WriteString(item.Val)
			state = sRaw

		case sRaw + sMax*int(lexer.ItemSep):
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemSep):
			if raw.Len() > 0 {
				t.Parts = append(t.Parts, raw.String())
				raw.Reset()
			}
			if len(t.Parts) > 0 && t.Parts[len(t.Parts)-1] != nil {
				t.Parts = append(t.Parts, nil)
			}

		case sRaw + sMax*int(lexer.ItemLacc):
			if raw.Len() > 0 {
				t.Parts = append(t.Parts, raw.String())
				raw.Reset()
			}
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemLacc):
			expr = Expr{}
			v = Var{}
			state = sAfterLacc

		case sAfterLacc + sMax*int(lexer.ItemOp):
			expr.Op = item.Val[0]
			state = sExpectVar

		case sExpectVar + sMax*int(lexer.ItemVar):
			fallthrough
		case sAfterLacc + sMax*int(lexer.ItemVar):
			t.Vars[item.Val] = struct{}{}
			v.ID = item.Val
			state = sAfterVar

		case sAfterMod + sMax*int(lexer.ItemComma):
			fallthrough
		case sAfterVar + sMax*int(lexer.ItemComma):
			expr.Vars = append(expr.Vars, v)
			v = Var{}
			state = sExpectVar

		case sAfterVar + sMax*int(lexer.ItemExplode):
			v.Mod = ModExplode
			state = sAfterMod

		case sAfterVar + sMax*int(lexer.ItemPrefix):
			v.Mod = ModPrefix
			state = sExpectLength

		case sExpectLength + sMax*int(lexer.ItemLength):
			v.Mod += Mod(mustAtoi(item.Val))
			state = sAfterMod

		case sAfterMod + sMax*int(lexer.ItemRacc):
			fallthrough
		case sAfterVar + sMax*int(lexer.ItemRacc):
			expr.Vars = append(expr.Vars, v)
			t.Parts = append(t.Parts, expr)
			state = sBeginRaw
		}
	}
	return
}
