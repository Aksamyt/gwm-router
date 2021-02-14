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
	"errors"
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

// Var represents a (possibly qualified) variable with its modifier.
type Var struct {
	ID  []string
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
		s.WriteString(strings.Join(v.ID, "."))
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

type Parser struct {
	t Ast
	e Expr
	v Var
	r strings.Builder
	s int
	i lexer.Item
}

func New() *Parser {
	return &Parser{
		t: Ast{Vars: map[string]struct{}{}},
		s: sBeginRaw,
	}
}

// Parse an URI template.
func (p *Parser) Parse(input string) (*Ast, error) {
	for p.i = range lexer.Lex(input) {
		if p.i.Typ == lexer.ItemError {
			return nil, Error{
				Err:   errors.New(p.i.Val),
				Input: input,
				Pos:   p.i.Pos,
			}
		}
		switch int(p.i.Typ)*sMax + p.s {
		case sRaw + sMax*int(lexer.ItemEOF):
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemEOF):
			if p.r.Len() > 0 {
				p.t.Parts = append(p.t.Parts, p.r.String())
			}
			return &p.t, nil

		case sRaw + sMax*int(lexer.ItemRaw):
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemRaw):
			p.r.WriteString(p.i.Val)
			p.s = sRaw

		case sRaw + sMax*int(lexer.ItemSep):
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemSep):
			if p.r.Len() > 0 {
				p.t.Parts = append(p.t.Parts, p.r.String())
				p.r.Reset()
			}
			if len(p.t.Parts) > 0 && p.t.Parts[len(p.t.Parts)-1] != nil {
				p.t.Parts = append(p.t.Parts, nil)
			}

		case sRaw + sMax*int(lexer.ItemLacc):
			if p.r.Len() > 0 {
				p.t.Parts = append(p.t.Parts, p.r.String())
				p.r.Reset()
			}
			fallthrough
		case sBeginRaw + sMax*int(lexer.ItemLacc):
			p.e = Expr{}
			p.v = Var{}
			p.s = sAfterLacc

		case sAfterLacc + sMax*int(lexer.ItemOp):
			p.e.Op = p.i.Val[0]
			p.s = sExpectVar

		case sExpectVar + sMax*int(lexer.ItemVar):
			fallthrough
		case sAfterLacc + sMax*int(lexer.ItemVar):
			if len(p.v.ID) == 0 {
				p.t.Vars[p.i.Val] = struct{}{}
			}
			p.v.ID = append(p.v.ID, p.i.Val)
			p.s = sAfterVar

		case sAfterVar + sMax*int(lexer.ItemDot):
			p.s = sExpectVar

		case sAfterMod + sMax*int(lexer.ItemComma):
			fallthrough
		case sAfterVar + sMax*int(lexer.ItemComma):
			p.e.Vars = append(p.e.Vars, p.v)
			p.v = Var{}
			p.s = sExpectVar

		case sAfterVar + sMax*int(lexer.ItemExplode):
			p.v.Mod = ModExplode
			p.s = sAfterMod

		case sAfterVar + sMax*int(lexer.ItemPrefix):
			p.v.Mod = ModPrefix
			p.s = sExpectLength

		case sExpectLength + sMax*int(lexer.ItemLength):
			p.v.Mod += Mod(mustAtoi(p.i.Val))
			p.s = sAfterMod

		case sAfterMod + sMax*int(lexer.ItemRacc):
			fallthrough
		case sAfterVar + sMax*int(lexer.ItemRacc):
			p.e.Vars = append(p.e.Vars, p.v)
			p.t.Parts = append(p.t.Parts, p.e)
			p.s = sBeginRaw

		default:
			return nil, Error{
				Input: input,
				Pos:   p.i.Pos,
				Err: &UnimplementedError{
					State: p.s,
					Item:  p.i,
				},
			}
		}
	}
	return nil, nil
}
