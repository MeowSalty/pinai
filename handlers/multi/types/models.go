package types

// OpenAIModel 结构体定义了 OpenAI 兼容的模型信息
type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelList 结构体定义了模型列表响应格式
type OpenAIModelList struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

// AnthropicModel 结构体定义了 Anthropic 兼容的模型信息
type AnthropicModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// AnthropicModelList 结构体定义了模型列表响应格式
type AnthropicModelList struct {
	Object string           `json:"object"`
	Data   []AnthropicModel `json:"data"`
}

// GeminiModel 结构体定义了 Gemini 兼容的模型信息
type GeminiModel struct {
	Name string `json:"name"`
}

// GeminiModelList 结构体定义了模型列表响应格式
type GeminiModelList struct {
	Models []GeminiModel `json:"models"`
}
