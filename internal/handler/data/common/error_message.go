package common

import (
	"errors"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
)

const defaultPublicErrorMessage = "请求处理失败"

// composePublicErrorMessage 统一组装“主文案 + 可选内部细节”的错误消息。
func composePublicErrorMessage(publicMessage string, err error, internalDetails ...string) string {
	mainMessage := strings.TrimSpace(publicMessage)
	collectedDetails := make([]string, 0, len(internalDetails)+1)

	for _, detail := range internalDetails {
		detail = strings.TrimSpace(detail)
		if detail != "" {
			collectedDetails = append(collectedDetails, detail)
		}
	}

	if err != nil {
		if detail := strings.TrimSpace(err.Error()); detail != "" {
			collectedDetails = append(collectedDetails, detail)
		}
	}

	if mainMessage == "" {
		if len(collectedDetails) > 0 {
			mainMessage = collectedDetails[0]
			collectedDetails = collectedDetails[1:]
		} else {
			mainMessage = defaultPublicErrorMessage
		}
	}

	uniqDetails := uniqueDiagnosticDetails(mainMessage, collectedDetails)
	if len(uniqDetails) == 0 {
		return mainMessage
	}

	return mainMessage + "（内部细节：" + strings.Join(uniqDetails, "；") + "）"
}

func uniqueDiagnosticDetails(mainMessage string, details []string) []string {
	result := make([]string, 0, len(details))
	seen := map[string]struct{}{}
	mainNorm := normalizeDiagnosticText(mainMessage)

	for _, detail := range details {
		detail = strings.TrimSpace(detail)
		if detail == "" {
			continue
		}

		detailNorm := normalizeDiagnosticText(detail)
		if detailNorm == "" {
			continue
		}
		if detailNorm == mainNorm {
			continue
		}
		if strings.Contains(mainNorm, detailNorm) || strings.Contains(detailNorm, mainNorm) {
			continue
		}

		if _, ok := seen[detailNorm]; ok {
			continue
		}

		seen[detailNorm] = struct{}{}
		result = append(result, detail)
	}

	return result
}

func normalizeDiagnosticText(text string) string {
	replacer := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"：", "",
		":", "",
		"（", "",
		"）", "",
		"(", "",
		")", "",
		"，", "",
		",", "",
		"。", "",
		".", "",
		"；", "",
		";", "",
	)

	return strings.ToLower(replacer.Replace(strings.TrimSpace(text)))
}

func firstDataPlaneError(items ...*gateway.DataPlaneError) *gateway.DataPlaneError {
	for _, item := range items {
		if item != nil {
			return item
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		if text := strings.TrimSpace(item); text != "" {
			return text
		}
	}

	return ""
}

func unwrapErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	for current := err; current != nil; current = errors.Unwrap(current) {
		if text := strings.TrimSpace(current.Error()); text != "" {
			return text
		}
	}

	return ""
}
