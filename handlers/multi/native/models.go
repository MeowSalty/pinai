package native

import (
	"fmt"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	multiAuth "github.com/MeowSalty/pinai/handlers/multi/auth"
	multiTypes "github.com/MeowSalty/pinai/handlers/multi/types"
	"github.com/gofiber/fiber/v2"
)

// SelectModels 处理模型列表请求，根据请求头中的提供者信息返回对应格式的模型列表。
// 支持 OpenAI、Anthropic 和 Gemini 三种提供者的模型列表格式。
func SelectModels() fiber.Handler {
	return func(c *fiber.Ctx) error {
		models, err := query.Q.Model.WithContext(c.Context()).Find()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("无法获取模型列表：%v", err),
			})
		}

		provider := strings.ToLower(multiAuth.ProviderFromContext(c))
		if provider == multiAuth.ProviderGemini {
			modelList := multiTypes.GeminiModelList{
				Models: make([]multiTypes.GeminiModel, 0, len(models)),
			}
			for _, model := range models {
				modelID := model.Name
				if model.Alias != "" {
					modelID = model.Alias
				}
				modelList.Models = append(modelList.Models, multiTypes.GeminiModel{
					Name: modelID,
				})
			}
			return c.JSON(modelList)
		}

		if provider == multiAuth.ProviderAnthropic {
			modelList := multiTypes.AnthropicModelList{
				Object: "list",
				Data:   make([]multiTypes.AnthropicModel, 0, len(models)),
			}
			for _, model := range models {
				modelID := model.Name
				if model.Alias != "" {
					modelID = model.Alias
				}
				modelList.Data = append(modelList.Data, multiTypes.AnthropicModel{
					ID:     modelID,
					Object: "model",
				})
			}
			return c.JSON(modelList)
		}

		modelList := multiTypes.OpenAIModelList{
			Object: "list",
			Data:   make([]multiTypes.OpenAIModel, 0, len(models)),
		}
		for _, model := range models {
			modelID := model.Name
			if model.Alias != "" {
				modelID = model.Alias
			}
			modelList.Data = append(modelList.Data, multiTypes.OpenAIModel{
				ID:     modelID,
				Object: "model",
			})
		}

		return c.JSON(modelList)
	}
}

// SelectGeminiModels 专门返回 Gemini API 兼容的 v1beta/models 格式的模型列表。
// 该函数用于处理针对 Gemini 服务的模型列表请求。
func SelectGeminiModels() fiber.Handler {
	return func(c *fiber.Ctx) error {
		models, err := query.Q.Model.WithContext(c.Context()).Find()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("无法获取模型列表：%v", err),
			})
		}

		modelList := multiTypes.GeminiModelList{
			Models: make([]multiTypes.GeminiModel, 0, len(models)),
		}
		for _, model := range models {
			modelID := model.Name
			if model.Alias != "" {
				modelID = model.Alias
			}
			modelList.Models = append(modelList.Models, multiTypes.GeminiModel{
				Name: modelID,
			})
		}

		return c.JSON(modelList)
	}
}
