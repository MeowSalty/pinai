package multi

import (
	"bufio"
	"encoding/json"
	"fmt"
)

// sendStreamError 发送流式错误响应
func sendStreamError(w *bufio.Writer, errType, message, code string) {
	errorEvent := map[string]any{
		"error": map[string]any{
			"type":    errType,
			"message": message,
			"code":    code,
		},
	}
	if jsonBytes, err := json.Marshal(errorEvent); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
		w.Flush()
	}
}
