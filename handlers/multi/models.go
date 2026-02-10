package multi

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/handlers/anthropic"
	"github.com/MeowSalty/pinai/handlers/openai"
	"github.com/gofiber/fiber/v2"
)

const (
	anthropicVersionHeader = "anthropic-version"
	anthropicAPIKeyHeader  = "x-api-key"
)

// SelectModels 根据请求头返回 OpenAI 或 Anthropic 格式的模型列表。
func SelectModels(apiToken string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if apiToken != "" {
			if err := validateMultiModelsAuth(c, apiToken); err != nil {
				return err
			}
		}

		models, err := query.Q.Model.WithContext(c.Context()).Find()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("无法获取模型列表：%v", err),
			})
		}

		if isAnthropicModelsRequest(c) {
			modelList := anthropic.ModelList{
				Object: "list",
				Data:   make([]anthropic.Model, 0, len(models)),
			}
			for _, model := range models {
				modelID := model.Name
				if model.Alias != "" {
					modelID = model.Alias
				}
				modelList.Data = append(modelList.Data, anthropic.Model{
					ID:     modelID,
					Object: "model",
				})
			}
			return c.JSON(modelList)
		}

		modelList := openai.ModelList{
			Object: "list",
			Data:   make([]openai.Model, 0, len(models)),
		}
		for _, model := range models {
			modelID := model.Name
			if model.Alias != "" {
				modelID = model.Alias
			}
			modelList.Data = append(modelList.Data, openai.Model{
				ID:     modelID,
				Object: "model",
			})
		}

		return c.JSON(modelList)
	}
}

func isAnthropicModelsRequest(c *fiber.Ctx) bool {
	if c.Get(anthropicAPIKeyHeader) == "" {
		return false
	}
	return c.Get(anthropicVersionHeader) != ""
}

func validateMultiModelsAuth(c *fiber.Ctx, apiToken string) error {
	if isAnthropicModelsRequest(c) {
		apiKey := c.Get(anthropicAPIKeyHeader)
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"type": "error",
				"error": fiber.Map{
					"type":    "authentication_error",
					"message": "缺少 x-api-key 头",
				},
			})
		}
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(apiToken)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"type": "error",
				"error": fiber.Map{
					"type":    "authentication_error",
					"message": "无效的 API key",
				},
			})
		}
		return nil
	}

	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "缺少 Authorization 头",
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization 头格式无效，应为：Bearer <token>",
		})
	}
	token := parts[1]

	if subtle.ConstantTimeCompare([]byte(token), []byte(apiToken)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "无效的 API token",
		})
	}

	return nil
}
