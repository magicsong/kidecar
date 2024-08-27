package extractor

import (
	"fmt"

	"github.com/tidwall/gjson"
)

func GetDataFromJsonText(json, path string) (interface{}, error) {
	if !gjson.Valid(json) {
		return nil, fmt.Errorf("invalid json")
	}
	value := gjson.Get(json, path)
	if !value.Exists() {
		return nil, fmt.Errorf("path not found")
	}
	return value.Value(), nil
}
