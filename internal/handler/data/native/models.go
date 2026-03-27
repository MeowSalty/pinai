package native

import (
	"net/http"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	multiAuth "github.com/MeowSalty/pinai/internal/handler/data/auth"
	"github.com/MeowSalty/pinai/internal/handler/data/common"
	multiTypes "github.com/MeowSalty/pinai/internal/handler/data/types"
	"github.com/gin-gonic/gin"
)

// SelectModels 处理模型列表请求，根据请求头中的提供者信息返回对应格式的模型列表。
// 支持 OpenAI、Anthropic 和 Gemini 三种提供者的模型列表格式。
func SelectModels() gin.HandlerFunc {
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

// SelectGeminiModels 专门返回 Gemini API 兼容的 v1beta/models 格式的模型列表。
// 该函数用于处理针对 Gemini 服务的模型列表请求。
func SelectGeminiModels() gin.HandlerFunc {
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
