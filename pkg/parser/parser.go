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
	"math"
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

type stateFn func(*parser) (stateFn, error)

type parser struct {
	ast      Ast
	expr     Expr
	variable Var
	raw      strings.Builder
	item     lexer.Item
}

func (p *parser) pushRawIfAny() {
	if p.raw.Len() > 0 {
		p.ast.Parts = append(p.ast.Parts, p.raw.String())
		p.raw.Reset()
	}
}

func (p *parser) pushSeparator() {
	if len(p.ast.Parts) == 0 || p.ast.Parts[len(p.ast.Parts)-1] != nil {
		p.ast.Parts = append(p.ast.Parts, nil)
	}
}

func (p *parser) appendVariablePart() {
	part := p.item.Val
	if len(p.variable.ID) == 0 {
		p.ast.Vars[part] = struct{}{}
	}
	p.variable.ID = append(p.variable.ID, part)
}

func (p *parser) pushVariable() {
	p.expr.Vars = append(p.expr.Vars, p.variable)
	p.variable = Var{}
}

func (p *parser) pushExpr() {
	p.ast.Parts = append(p.ast.Parts, p.expr)
	p.expr = Expr{}
}

func (p *parser) assignOp() {
	p.expr.Op = p.item.Val[0]
}

func (p *parser) setVariableLength() {
	length, _ := strconv.Atoi(p.item.Val)
	p.variable.Mod = ModPrefix + Mod(length)
}

func (p *parser) setVariableExplode() {
	p.variable.Mod = ModExplode
}

func (p *parser) noModifierOrError() error {
	if p.variable.Mod != 0 {
		return DoubleModError
	}
	return nil
}

func (p *parser) afterVarOrLengthError() error {
	if p.variable.Mod&ModPrefix != 0 {
		firstByte := p.item.Val[0]
		if firstByte >= '0' && firstByte <= '9' {
			p.item.Pos -= int(math.Log10(float64(
				p.variable.Mod^ModPrefix,
			))) + 1
			return LengthOver9999Error
		}
	}
	return AfterVarError
}

// Parse parses an URI template and returns an Ast or an error detailing what
// happened.
func Parse(input string) (*Ast, error) {
	p := parser{
		ast: Ast{Vars: map[string]struct{}{}},
	}
	state, err := pRaw, error(nil)
	for p.item = range lexer.Lex(input) {
		if p.item.Typ == lexer.ItemError {
			return nil, Error{
				Input: input,
				Pos:   p.item.Pos,
				Err:   LexerError{p.item},
			}
		}
		if state, err = state(&p); err != nil {
			return nil, Error{
				Input: input,
				Pos:   p.item.Pos,
				Err:   err,
			}
		}
		if state == nil {
			break
		}
	}
	return &p.ast, nil
}

func pRaw(p *parser) (state stateFn, err error) {
	state = pRaw
	switch p.item.Typ {
	case lexer.ItemRaw:
		p.raw.WriteString(p.item.Val)

	case lexer.ItemSep:
		p.pushRawIfAny()
		p.pushSeparator()

	case lexer.ItemLacc:
		p.pushRawIfAny()
		state = pMaybeOp

	case lexer.ItemEOF:
		p.pushRawIfAny()
		state = nil

	default:
		err = UnimplementedError{p.item, "pRaw"}
	}
	return
}

func pMaybeOp(p *parser) (stateFn, error) {
	if p.item.Typ == lexer.ItemOp {
		p.assignOp()
		return pExpr, nil
	}
	return pExpr(p)
}

func pExpr(p *parser) (state stateFn, err error) {
	state = pExpr
	switch p.item.Typ {
	case lexer.ItemVar:
		p.appendVariablePart()
		state = pAfterVar

	case lexer.ItemComma, lexer.ItemDot, lexer.ItemRacc:
		err = ExpectedVarError

	default:
		err = UnimplementedError{p.item, "pExpr"}
	}
	return
}

func pAfterVar(p *parser) (state stateFn, err error) {
	switch p.item.Typ {
	case lexer.ItemRacc:
		p.pushVariable()
		p.pushExpr()
		state = pRaw

	case lexer.ItemDot:
		state = pExpr

	case lexer.ItemComma:
		p.pushVariable()
		state = pExpr

	case lexer.ItemPrefix:
		err = p.noModifierOrError()
		state = pLength

	case lexer.ItemExplode:
		err = p.noModifierOrError()
		p.setVariableExplode()
		state = pAfterVar

	case lexer.ItemVar:
		err = p.afterVarOrLengthError()

	default:
		err = UnimplementedError{p.item, "pAfterVar"}
	}
	return
}

func pLength(p *parser) (stateFn, error) {
	if p.item.Typ == lexer.ItemLength {
		p.setVariableLength()
		return pAfterVar, nil
	}
	return nil, UnimplementedError{p.item, "pLength"}
}
