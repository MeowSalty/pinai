package openai

// Model 结构体定义了 OpenAI 兼容的模型信息
type Model struct {
	ID      string `json:"id"`       // 模型 ID
	Object  string `json:"object"`   // 对象类型，通常是 "model"
	Created int64  `json:"created"`  // 创建时间戳
	OwnedBy string `json:"owned_by"` // 模型所有者
}

// ModelList 结构体定义了模型列表响应格式
type ModelList struct {
	Object string  `json:"object"` // 对象类型，通常是 "list"
	Data   []Model `json:"data"`   // 模型列表
}
