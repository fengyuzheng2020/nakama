package utils

import "encoding/json"

// 解析json参数 支持泛型
func ParseJsonParam[T any](data string, result *T) error {
	return json.Unmarshal([]byte(data), result)
}
