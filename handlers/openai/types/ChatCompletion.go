package types

type JsonSchema struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Schema      any     `json:"schema,omitempty"`
	Strict      bool    `json:"strict,omitempty"`
}

// ResponseFormat 结构体定义了响应格式
type ResponseFormat struct {
	Type       string      `json:"type"` // 类型，"text"、"json_object"、"json_schema"
	JsonSchema *JsonSchema `json:"json_schema,omitempty"`
}

// Function 结构体定义了一个函数工具
type Function struct {
	Name        string `json:"name"`                  // 函数名称
	Description string `json:"description,omitempty"` // 函数描述
	Parameters  any    `json:"parameters"`            // 函数参数
}

type Grammar struct {
	Definition string `json:"definition"`
	Syntax     string `json:"syntax"`
}

type Format struct {
	Type    string   `json:"type"`
	Grammar *Grammar `json:"grammar,omitempty"`
}

type Custom struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Format      Format `json:"parameters"`
}

// Tool 结构体定义了模型可以使用的工具
type Tool struct {
	Type     string    `json:"type"`               // 工具类型，function 或 custom
	Function *Function `json:"function,omitempty"` // 工具函数
	Custom   *Custom   `json:"custom,omitempty"`
}

type TextContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ImageUrl struct {
	URL     string  `json:"url"`
	Details *string `json:"details,omitempty"`
}

type ImageContentPart struct {
	Type     string   `json:"type"`
	ImageUrl ImageUrl `json:"image_url"`
}

type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type AudioContentPart struct {
	Type       string     `json:"type"` // 始终是 `input_audio`
	InputAudio InputAudio `json:"input_audio"`
}

type File struct {
	FileID   string `json:"file_id"`
	FileName string `json:"filename"`
	FileData string `json:"file_data"`
}

type FileContentPart struct {
	Type string `json:"type"` // 始终是 `file`
	File File   `json:"file"`
}

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string 或上面的各种 Part 的数组
}

// ChatCompletionRequest 结构体定义了聊天补全请求参数
//
// 存在暂未完全支持的参数：
//
//   - audio
//   - metadata
//   - modalities
//   - parallel_tool_calls
//   - prediction
//   - prompt_cache_key
//   - safety_identifier
//   - service_tier
//   - store
//   - stream_options
//   - tool_choice
//   - web_search_options
//
// 处于废弃状态且不打算支持的参数：
//   - function_call
//   - functions
//   - max_tokens
//   - user
//
// 请参阅 https://platform.openai.com/docs/api-reference/chat/create
type ChatCompletionRequest struct {
	// 必要参数
	Model    string    `json:"model"`    // 模型名称
	Messages []Message `json:"messages"` // 消息列表

	// 可选参数
	FrequencyPenalty    float64        `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]int `json:"logit_bias,omitempty"`
	LogProbs            bool           `json:"logprobs,omitempty"`
	MaxCompletionTokens uint           `json:"max_completion_tokens,omitempty"`
	N                   uint           `json:"n,omitempty"`
	PresencePenalty     float64        `json:"presence_penalty,omitempty"`
	ReasoningEffort     string         `json:"reasoning_effort,omitempty"`
	ResponseFormat      ResponseFormat `json:"response_format,omitempty"`
	Seed                int            `json:"seed,omitempty"`
	Stop                any            `json:"stop,omitempty"`   // 停止序列
	Stream              bool           `json:"stream,omitempty"` // 是否使用流式响应
	Temperature         float64        `json:"temperature,omitempty"`
	Tools               []Tool         `json:"tools,omitempty"`
	TopLogProbs         uint           `json:"top_logprobs,omitempty"`
	TopP                float64        `json:"top_p,omitempty"`
	Verbosity           string         `json:"verbosity,omitempty"`
}

type Delta struct {
	Content   *string `json:"content"`
	Refusal   *string `json:"refusal,omitempty"`
	Role      string  `json:"role"`
	ToolCalls []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Index    uint   `json:"index"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
}

// TopLogProb defines top log probability.
type TopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []byte  `json:"bytes"`
}

// TokenLogProb defines log probability for a token.
type TokenLogProb struct {
	Bytes       []byte       `json:"bytes,omitempty"`
	LogProb     float64      `json:"logprob"`
	Token       string       `json:"token"`
	TopLogProbs []TopLogProb `json:"top_logprobs"`
}

type Refusal struct {
	Bytes       []byte       `json:"bytes,omitempty"`
	LogProb     float64      `json:"logprob"`
	Token       string       `json:"token"`
	TopLogProbs []TopLogProb `json:"top_logprobs"`
}

// LogProbs defines log probability information.
type LogProbs struct {
	Content []TokenLogProb `json:"content"`
	Refusal []Refusal      `json:"refusal"`
}

// Choice 结构体定义了聊天补全响应中的选项内容
type Choice struct {
	Index        int       `json:"index"` // 选项索引
	Delta        Delta     `json:"delta"`
	LogProbs     *LogProbs `json:"logprobs,omitempty"`      // 日志概率
	FinishReason *string   `json:"finish_reason,omitempty"` // 结束原因
}

// Usage 结构体定义了 token 使用情况统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入的 token 数量
	CompletionTokens int `json:"completion_tokens"` // 输出的 token 数量
	TotalTokens      int `json:"total_tokens"`      // 总 token 数量
}

// ChatCompletionResponse 结构体定义了聊天补全响应格式
type ChatCompletionResponse struct {
	ID                string   `json:"id"`                     // 响应 ID
	Object            string   `json:"object"`                 // 对象类型，通常是 "chat.completion"
	Created           int64    `json:"created"`                // 创建时间戳
	Model             string   `json:"model"`                  // 使用的模型
	SystemFingerprint string   `json:"system_fingerprint"`     // 系统指纹
	ServiceTier       string   `json:"service_tier,omitempty"` // 服务层
	Choices           []Choice `json:"choices"`                // 结果选项
	Usage             Usage    `json:"usage,omitempty"`        // token 使用情况
}
