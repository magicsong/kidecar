package template

import (
	"fmt"
	"reflect"
)

func expressionReplaceValue(value string) (string, error) {
	// 这里添加你自己的模板解析逻辑
	return value, nil
}

// ParseConfig 递归地解析配置结构体中的字段
func ParseConfig(config interface{}) error {
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanInterface() {
			continue
		}

		// 检查是否有 `parse:"true"` 标签
		if tagValue, ok := fieldType.Tag.Lookup("parse"); ok && tagValue == "true" {
			// 解析字段值
			if field.Kind() == reflect.String {
				parsedValue, err := expressionReplaceValue(field.String())
				if err != nil {
					return fmt.Errorf("failed to parse field %s: %w", fieldType.Name, err)
				}
				field.SetString(parsedValue)
			}
		}

		// 如果是结构体或指向结构体的指针，递归处理
		if field.Kind() == reflect.Struct {
			if err := ParseConfig(field.Addr().Interface()); err != nil {
				return err
			}
		} else if field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct {
			if err := ParseConfig(field.Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}
