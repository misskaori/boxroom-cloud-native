package util_func

import (
	"errors"
	"fmt"
	"reflect"
)

type Duplicator struct {
}

func NewDuplicator() *Duplicator {
	return &Duplicator{}
}

func (d *Duplicator) CopyFieldsByReflect(target interface{}, source interface{}, fields ...string) error {
	targetType := reflect.TypeOf(target)
	targetValues := reflect.ValueOf(target)
	sourceType := reflect.TypeOf(source)
	sourceValues := reflect.ValueOf(source)

	if targetType.Kind() != reflect.Ptr {
		e := fmt.Sprintf("target must be a ptr")
		return errors.New(e)
	}
	if sourceType.Kind() != reflect.Struct {
		e := fmt.Sprintf("target must be a struct")
		return errors.New(e)
	}
	targetValues = reflect.ValueOf(targetValues.Interface())
	_fields := make([]string, 0)
	if len(fields) > 0 {
		_fields = fields
	} else {
		for i := 0; i < sourceValues.NumField(); i++ {
			_fields = append(_fields, sourceType.Field(i).Name)
		}
	}
	if len(_fields) == 0 {
		return nil
	}
	for i := 0; i < len(_fields); i++ {
		name := _fields[i]
		f := targetValues.Elem().FieldByName(name)
		sourceValue := sourceValues.FieldByName(name)
		if f.IsValid() && f.Kind() == sourceValue.Kind() {
			f.Set(sourceValue)
		} else {
			e := fmt.Sprintf("there is no such field in target, fieldName: %s\n", name)
			return errors.New(e)
		}
	}
	return nil
}
