package types

// 限流配置
type RateLimitConfig struct {
	RPM int `json:"rpm"` // 每分钟请求数限制
	TPM int `json:"tpm"` // 每分钟 Token 数限制
}

// 平台表 (platforms)
type Platform struct {
	ID        uint            `gorm:"primaryKey" json:"id"`              // 平台 ID
	Name      string          `gorm:"index" json:"name"`                 // 平台名称
	Format    string          `gorm:"index" json:"format"`               // API 格式（例如：openai, anthropic, azure 等）
	BaseURL   string          `json:"base_url"`                          // 基础 URL（可选）
	RateLimit RateLimitConfig `gorm:"serializer:json" json:"rate_limit"` // 限流配置
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
