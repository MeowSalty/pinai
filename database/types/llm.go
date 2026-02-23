package types

// 限流配置
type RateLimitConfig struct {
	RPM int `json:"rpm"` // 每分钟请求数限制
	TPM int `json:"tpm"` // 每分钟 Token 数限制
}

// Endpoint 表示平台端点配置。
// 端点用于存储不同平台的各种服务端点的路径和配置信息。
type Endpoint struct {
	ID              uint              `gorm:"primaryKey" json:"id"`                                                                                                    // 端点 ID
	PlatformID      uint              `gorm:"index:idx_endpoints_platform_default,priority:1;index:idx_endpoints_platform_type_variant,priority:1" json:"platform_id"` // 平台 ID（外键）
	EndpointType    string            `gorm:"index:idx_endpoints_platform_type_variant,priority:2" json:"endpoint_type"`                                               // 端点类型
	EndpointVariant string            `gorm:"index:idx_endpoints_platform_type_variant,priority:3" json:"endpoint_variant"`                                            // 端点变体
	Path            string            `json:"path"`                                                                                                                    // 端点路径
	CustomHeaders   map[string]string `gorm:"serializer:json" json:"custom_headers"`                                                                                   // 自定义请求头
	IsDefault       bool              `gorm:"index:idx_endpoints_platform_default,priority:2" json:"is_default"`                                                       // 是否为默认端点
}

// 平台表 (platforms)
type Platform struct {
	ID        uint            `gorm:"primaryKey" json:"id"`              // 平台 ID
	Name      string          `gorm:"index" json:"name"`                 // 平台名称
	BaseURL   string          `json:"base_url"`                          // 基础 URL
	RateLimit RateLimitConfig `gorm:"serializer:json" json:"rate_limit"` // 限流配置
	Endpoints []Endpoint      `json:"endpoints,omitempty"`               // 平台端点列表
}

// 模型表 (models)
type Model struct {
	ID         uint     `gorm:"primaryKey" json:"id"`     // 模型 ID
	PlatformID uint     `gorm:"index" json:"platform_id"` // 平台 ID（外键）
	Name       string   `gorm:"index" json:"name"`        // 模型名称（平台中的模型标识）
	Alias      string   `gorm:"index" json:"alias"`       // 模型别名（可选）
	Platform   Platform `json:"-"`
	APIKeys    []APIKey `gorm:"many2many:api_key_models;" json:"api_keys,omitempty"` // Many-to-Many 关系
}

// 密钥表 (api_keys)
type APIKey struct {
	ID         uint     `gorm:"primaryKey" json:"id"`     // 密钥 ID
	PlatformID uint     `gorm:"index" json:"platform_id"` // 平台 ID（外键）
	Value      string   `json:"value"`                    // 密钥值
	Platform   Platform `json:"-"`
	Models     []Model  `gorm:"many2many:api_key_models;" json:"models,omitempty"`
}
