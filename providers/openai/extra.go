package openai

import (
	"encoding/json"

	"github.com/openai/openai-go/packages/respjson"
)

// stringFromExtra 从 OpenAI SDK JSON.ExtraFields 提取字符串值。
// OpenAI SDK v1.12.0 未将 reasoning_content 等字段定义为类型，
// 这些字段落入 JSON.ExtraFields，标记为 status=invalid 但 Raw() 保留原始 JSON。
//
// respjson.Field.Raw() 返回原始 JSON token:
//   - 字符串值: `"hello"` (含双引号)
//   - null: `"null"`
//   - 缺失: `""`
//
// 不检查 Valid()，因为 SDK 将未识别字段标记为 invalid，但数据完整。
func stringFromExtra(extra map[string]respjson.Field, key string) string {
	f, ok := extra[key]
	if !ok {
		return ""
	}
	raw := f.Raw()
	if raw == "" || raw == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return ""
	}
	return s
}
