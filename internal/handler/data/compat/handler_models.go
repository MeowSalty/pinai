package multi

import (
	"net/http"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	multiAuth "github.com/MeowSalty/pinai/internal/handler/data/auth"
	"github.com/MeowSalty/pinai/internal/handler/data/common"
	multiTypes "github.com/MeowSalty/pinai/internal/handler/data/types"
	"github.com/gin-gonic/gin"
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
// @Failure      500  {object}  gin.H
// @Router       /multi/v1/models [get]
// @Security     ApiKeyAuth
func (h *Handler) SelectModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := strings.ToLower(multiAuth.ProviderFromContext(c))

		models, err := query.Q.Model.WithContext(c.Request.Context()).Find()
		if err != nil {
			switch provider {
			case multiAuth.ProviderGemini:
				c.JSON(http.StatusInternalServerError, common.NewGeminiErrorResponse("无法获取模型列表", http.StatusInternalServerError, err))
			case multiAuth.ProviderAnthropic:
				c.JSON(http.StatusInternalServerError, common.NewAnthropicErrorResponse("无法获取模型列表", http.StatusInternalServerError, err))
			default:
				c.JSON(http.StatusInternalServerError, common.NewOpenAIHTTPErrorResponse("无法获取模型列表", http.StatusInternalServerError, err))
			}
			return
		}

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
			c.JSON(http.StatusOK, modelList)
			return
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
			c.JSON(http.StatusOK, modelList)
			return
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

		c.JSON(http.StatusOK, modelList)
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
// @Failure      500  {object}  gin.H
// @Router       /multi/v1beta/models [get]
// @Security     ApiKeyAuth
func (h *Handler) SelectGeminiModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		models, err := query.Q.Model.WithContext(c.Request.Context()).Find()
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.NewGeminiErrorResponse("无法获取模型列表", http.StatusInternalServerError, err))
			return
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

		c.JSON(http.StatusOK, modelList)
	}
}
