package multi

import (
	"fmt"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	multiAuth "github.com/MeowSalty/pinai/handlers/multi/auth"
	multiTypes "github.com/MeowSalty/pinai/handlers/multi/types"
	"github.com/gofiber/fiber/v2"
)

// SelectModels 处理获取模型列表请求，路径为 GET /multi/v1/models。
// 根据请求头中的 provider 字段返回对应格式的模型列表（OpenAI/Anthropic/Gemini）。
// 成功时返回 200 状态码及模型列表，失败时返回 500 状态码。
//
// @Summary      获取模型列表
// @Description  根据请求头中的 provider 字段返回对应格式的模型列表
// @Tags         Models
// @Accept       json
// @Produce      json
// @Success      200  {object}  multiTypes.OpenAIModelList
// @Success      200  {object}  multiTypes.AnthropicModelList
// @Success      200  {object}  multiTypes.GeminiModelList
// @Failure      500  {object}  fiber.Map
// @Router       /multi/v1/models [get]
// @Security     ApiKeyAuth
func (h *Handler) SelectModels() fiber.Handler {
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

// SelectGeminiModels 处理获取 Gemini 模型列表请求，路径为 GET /multi/v1beta/models。
// 返回 Gemini API 格式的模型列表，包含所有可用模型的名称信息。
// 成功时返回 200 状态码及模型列表，失败时返回 500 状态码。
//
// @Summary      获取 Gemini 模型列表
// @Description  返回 Gemini v1beta API 格式的模型列表
// @Tags         Models
// @Accept       json
// @Produce      json
// @Success      200  {object}  multiTypes.GeminiModelList
// @Failure      500  {object}  fiber.Map
// @Router       /multi/v1beta/models [get]
// @Security     ApiKeyAuth
func (h *Handler) SelectGeminiModels() fiber.Handler {
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
