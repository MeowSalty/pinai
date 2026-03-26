package portal

import (
	"fmt"
	"strings"
)

// parseModelMapping 解析模型映射字符串
//
// 将字符串格式的模型映射转换为 map[string]string
//
// 参数：
//   - mappingStr: 模型映射字符串，格式为 "key1:value1,key2:value2"
//
// 返回值：
//   - map[string]string: 解析后的模型映射
//   - error: 解析过程中可能出现的错误
func parseModelMapping(mappingStr string) (map[string]string, error) {
	if mappingStr == "" {
		return make(map[string]string), nil
	}

	result := make(map[string]string)
	pairs := strings.Split(mappingStr, ",")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("无效的模型映射格式: %s，期望格式为 key:value", pair)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		if key == "" || value == "" {
			return nil, fmt.Errorf("模型映射的键和值不能为空: %s", pair)
		}

		result[key] = value
	}

	return result, nil
}
