package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func ConvertJsonObjectToStruct(source interface{}, target interface{}) error {
	if source == nil {
		return fmt.Errorf("source is nil")
	}
	if target == nil {
		return fmt.Errorf("target is nil")
	}
	// source must be a map[string]
	// obj must be a pointer to a struct
	_, ok := source.(map[string]interface{})
	if !ok {
		return fmt.Errorf("source must be a map[string]interface{}")
	}
	if !isPointerToStruct(target) {
		return fmt.Errorf("target must be a pointer to a struct")
	}
	// then convert
	bytes, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}
	err = json.Unmarshal(bytes, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}
	return nil
}

func isPointerToStruct(obj interface{}) bool {
	v := reflect.ValueOf(obj)
	return v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct
}
