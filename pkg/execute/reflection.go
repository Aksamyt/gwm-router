/*
  This file is part of the uritemplate project.
  Copyright (C) 2021 Alexandre Szymocha (@Aksamyt).

  This Source Code Form is subject to the terms of the Mozilla Public
  License, v. 2.0. If a copy of the MPL was not distributed with this
  file, You can obtain one at http://mozilla.org/MPL/2.0/.
*/

package execute

import (
	"reflect"

	"github.com/aksamyt/uritemplate/pkg/parser"
)

func dereference(v *reflect.Value) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		*v = v.Elem()
	}
}

// getByTag assumes s is a Struct value.
func getByTag(s reflect.Value, key string) reflect.Value {
	for i, t := 0, s.Type(); i < t.NumField(); i++ {
		field := t.Field(i)
		if tag, ok := field.Tag.Lookup("uri"); ok && tag == key {
			return s.Field(i)
		}
	}
	return reflect.Value{}
}

func getByKey(data reflect.Value, key string) (value reflect.Value) {
	switch data.Kind() {
	case reflect.Map:
		keyValue := reflect.ValueOf(key)
		if keyValue.Type().AssignableTo(data.Type().Key()) {
			value = data.MapIndex(keyValue)
		}
	case reflect.Struct:
		if value = data.FieldByName(key); !value.IsValid() {
			value = getByTag(data, key)
		}
	}
	dereference(&value)
	return
}

func findVariableValue(data reflect.Value, v *parser.Var) reflect.Value {
	value := data
	for _, part := range v.ID {
		value = getByKey(value, part)
	}
	return value
}
