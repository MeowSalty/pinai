package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// MapDataPlaneError 对数据面错误进行第一轮统一映射。
func (s *service) MapDataPlaneError(err error, fallbackAction string) DataPlaneError {
	if err == nil {
		mapped := defaultDataPlaneError(fallbackAction)
		mapped.ShouldProxyAsHTTPError = true
		return mapped
	}

	mapped := DataPlaneError{
		StatusCode:             http.StatusInternalServerError,
		Message:                fmt.Sprintf("%s：%v", fallbackAction, err),
		Raw:                    err,
		ShouldProxyAsHTTPError: true,
	}

	extractStructuredDataPlaneError(err, &mapped)
	applyHeuristicDataPlaneError(err, fallbackAction, &mapped)

	if mapped.Raw == nil {
		mapped.Raw = err
	}

	return mapped

}

func extractStructuredDataPlaneError(err error, out *DataPlaneError) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		extractByInterfaces(current, out)
		extractByReflection(current, out)
	}
}

func extractByInterfaces(err error, out *DataPlaneError) {
	type statusCoder interface{ StatusCode() int }
	type statusGetter interface{ GetStatusCode() int }
	type httpStatusCoder interface{ HTTPStatusCode() int }
	type providerGetter interface{ Provider() string }
	type typeGetter interface{ Type() string }
	type codeGetter interface{ Code() string }
	type paramGetter interface{ Param() string }
	type retryableGetter interface{ Retryable() bool }
	type shouldProxyGetter interface{ ShouldProxyAsHTTPError() bool }
	type rawGetter interface{ Raw() any }

	var sc statusCoder
	if errors.As(err, &sc) && validHTTPStatus(sc.StatusCode()) {
		out.StatusCode = sc.StatusCode()
	}

	var sg statusGetter
	if errors.As(err, &sg) && validHTTPStatus(sg.GetStatusCode()) {
		out.StatusCode = sg.GetStatusCode()
	}

	var hc httpStatusCoder
	if errors.As(err, &hc) && validHTTPStatus(hc.HTTPStatusCode()) {
		out.StatusCode = hc.HTTPStatusCode()
	}

	var pg providerGetter
	if errors.As(err, &pg) {
		setIfEmptyString(&out.Provider, pg.Provider())
	}

	var tg typeGetter
	if errors.As(err, &tg) {
		setIfEmptyString(&out.ErrorType, tg.Type())
	}

	var cg codeGetter
	if errors.As(err, &cg) {
		setIfEmptyString(&out.ErrorCode, cg.Code())
	}

	var pa paramGetter
	if errors.As(err, &pa) {
		setIfEmptyString(&out.Param, pa.Param())
	}

	var rg retryableGetter
	if errors.As(err, &rg) {
		out.Retryable = rg.Retryable()
	}

	var sp shouldProxyGetter
	if errors.As(err, &sp) {
		out.ShouldProxyAsHTTPError = sp.ShouldProxyAsHTTPError()
	}

	var raw rawGetter
	if errors.As(err, &raw) {
		if payload := raw.Raw(); payload != nil {
			out.Raw = payload
		}
	}
}

func extractByReflection(err error, out *DataPlaneError) {
	v := reflect.ValueOf(err)
	if !v.IsValid() {
		return
	}

	if status, ok := reflectFirstInt(v, "StatusCode", "HTTPStatus", "Status"); ok && validHTTPStatus(status) {
		out.StatusCode = status
	}

	if msg, ok := reflectFirstString(v, "Message", "Msg"); ok {
		out.Message = msg
	}
	if provider, ok := reflectFirstString(v, "Provider"); ok {
		setIfEmptyString(&out.Provider, provider)
	}
	if errType, ok := reflectFirstString(v, "ErrorType", "Type"); ok {
		setIfEmptyString(&out.ErrorType, errType)
	}
	if errCode, ok := reflectFirstString(v, "ErrorCode", "Code"); ok {
		setIfEmptyString(&out.ErrorCode, errCode)
	}
	if param, ok := reflectFirstString(v, "Param", "Parameter"); ok {
		setIfEmptyString(&out.Param, param)
	}
	if retryable, ok := reflectFirstBool(v, "Retryable", "IsRetryable", "Temporary"); ok {
		out.Retryable = retryable
	}
	if shouldProxy, ok := reflectFirstBool(v, "ShouldProxyAsHTTPError", "ProxyAsHTTPError"); ok {
		out.ShouldProxyAsHTTPError = shouldProxy
	}
	if raw, ok := reflectFirstAny(v, "Raw", "Body", "Payload", "Data", "Detail", "Details", "ResponseBody"); ok {
		out.Raw = raw
	}

	if nestedErr, ok := reflectFieldValue(v, "Error"); ok {
		extractNestedValue(nestedErr, out)
	}
}

func extractNestedValue(v reflect.Value, out *DataPlaneError) {
	if !v.IsValid() {
		return
	}

	if status, ok := reflectFirstInt(v, "StatusCode", "HTTPStatus", "Status"); ok && validHTTPStatus(status) {
		out.StatusCode = status
	}
	if msg, ok := reflectFirstString(v, "Message", "Msg"); ok {
		out.Message = msg
	}
	if errType, ok := reflectFirstString(v, "ErrorType", "Type"); ok {
		setIfEmptyString(&out.ErrorType, errType)
	}
	if errCode, ok := reflectFirstString(v, "ErrorCode", "Code"); ok {
		setIfEmptyString(&out.ErrorCode, errCode)
	}
	if param, ok := reflectFirstString(v, "Param", "Parameter"); ok {
		setIfEmptyString(&out.Param, param)
	}
}

func applyHeuristicDataPlaneError(err error, fallbackAction string, out *DataPlaneError) {
	lowerMsg := strings.ToLower(err.Error())

	switch {
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(lowerMsg, "timeout"), strings.Contains(lowerMsg, "deadline"):
		setStatusAndMessageIfMissing(out, http.StatusGatewayTimeout, "上游请求超时")
		setIfEmptyString(&out.ErrorType, "timeout_error")
		setIfEmptyString(&out.ErrorCode, "upstream_timeout")
		out.Retryable = true
	case errors.Is(err, context.Canceled):
		setStatusAndMessageIfMissing(out, http.StatusRequestTimeout, "请求已取消")
		setIfEmptyString(&out.ErrorType, "request_canceled")
		setIfEmptyString(&out.ErrorCode, "request_canceled")
	case strings.Contains(lowerMsg, "429"), strings.Contains(lowerMsg, "rate limit"), strings.Contains(lowerMsg, "too many requests"), strings.Contains(lowerMsg, "quota"):
		setStatusAndMessageIfMissing(out, http.StatusTooManyRequests, "请求过于频繁，请稍后重试")
		setIfEmptyString(&out.ErrorType, "rate_limit_error")
		setIfEmptyString(&out.ErrorCode, "rate_limit")
		out.Retryable = true
	case strings.Contains(lowerMsg, "401"), strings.Contains(lowerMsg, "403"), strings.Contains(lowerMsg, "unauthorized"), strings.Contains(lowerMsg, "forbidden"), strings.Contains(lowerMsg, "authentication"):
		setStatusAndMessageIfMissing(out, http.StatusUnauthorized, "鉴权失败")
		setIfEmptyString(&out.ErrorType, "authentication_error")
		setIfEmptyString(&out.ErrorCode, "auth_failed")
	case strings.Contains(lowerMsg, "404"), strings.Contains(lowerMsg, "not found"):
		setStatusAndMessageIfMissing(out, http.StatusNotFound, "请求资源不存在")
		setIfEmptyString(&out.ErrorType, "not_found_error")
		setIfEmptyString(&out.ErrorCode, "not_found")
	case strings.Contains(lowerMsg, "400"), strings.Contains(lowerMsg, "bad request"), strings.Contains(lowerMsg, "invalid request"), strings.Contains(lowerMsg, "invalid parameter"):
		setStatusAndMessageIfMissing(out, http.StatusBadRequest, "请求参数错误")
		setIfEmptyString(&out.ErrorType, "invalid_request_error")
		setIfEmptyString(&out.ErrorCode, "invalid_request")
	default:
		if out.Message == "" {
			out.Message = fmt.Sprintf("%s：%v", fallbackAction, err)
		}
		if out.ErrorType == "" && out.StatusCode >= http.StatusInternalServerError {
			out.ErrorType = "internal_error"
		}
	}

	if out.StatusCode == 0 {
		out.StatusCode = http.StatusInternalServerError
	}

	if out.Message == "" {
		out.Message = fmt.Sprintf("%s：%v", fallbackAction, err)
	}

	if !out.Retryable {
		out.Retryable = isRetryableByStatus(out.StatusCode)
	}

	if !out.ShouldProxyAsHTTPError {
		out.ShouldProxyAsHTTPError = validHTTPStatus(out.StatusCode)
	}
}

func setStatusAndMessageIfMissing(out *DataPlaneError, status int, message string) {
	if !validHTTPStatus(out.StatusCode) {
		out.StatusCode = status
	}
	if out.Message == "" {
		out.Message = message
	}
}

func setIfEmptyString(target *string, value string) {
	if *target == "" && strings.TrimSpace(value) != "" {
		*target = strings.TrimSpace(value)
	}
}

func validHTTPStatus(status int) bool {
	return status >= 100 && status <= 599
}

func isRetryableByStatus(status int) bool {
	if status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status == http.StatusGatewayTimeout {
		return true
	}

	return status >= http.StatusInternalServerError
}

func reflectFirstString(v reflect.Value, names ...string) (string, bool) {
	for _, name := range names {
		if fv, ok := reflectFieldValue(v, name); ok {
			if str, ok := valueToString(fv); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					return str, true
				}
			}
		}
	}

	return "", false
}

func reflectFirstInt(v reflect.Value, names ...string) (int, bool) {
	for _, name := range names {
		if fv, ok := reflectFieldValue(v, name); ok {
			if i, ok := valueToInt(fv); ok {
				return i, true
			}
		}
	}

	return 0, false
}

func reflectFirstBool(v reflect.Value, names ...string) (bool, bool) {
	for _, name := range names {
		if fv, ok := reflectFieldValue(v, name); ok {
			if b, ok := valueToBool(fv); ok {
				return b, true
			}
		}
	}

	return false, false
}

func reflectFirstAny(v reflect.Value, names ...string) (any, bool) {
	for _, name := range names {
		if fv, ok := reflectFieldValue(v, name); ok {
			resolved := indirectValue(fv)
			if resolved.IsValid() {
				return resolved.Interface(), true
			}
		}
	}

	return nil, false
}

func reflectFieldValue(v reflect.Value, name string) (reflect.Value, bool) {
	v = indirectValue(v)
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			if strings.EqualFold(field.Name, name) {
				return v.Field(i), true
			}
		}
	}

	if v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String {
		for _, key := range v.MapKeys() {
			if strings.EqualFold(key.String(), name) {
				return v.MapIndex(key), true
			}
		}
	}

	return reflect.Value{}, false
}

func indirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}

	return v
}

func valueToString(v reflect.Value) (string, bool) {
	v = indirectValue(v)
	if !v.IsValid() {
		return "", false
	}

	switch v.Kind() {
	case reflect.String:
		return v.String(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), true
	case reflect.Bool:
		if v.Bool() {
			return "true", true
		}
		return "false", true
	default:
		if v.CanInterface() {
			if s, ok := v.Interface().(fmt.Stringer); ok {
				return s.String(), true
			}
		}
	}

	return "", false
}

func valueToInt(v reflect.Value) (int, bool) {
	v = indirectValue(v)
	if !v.IsValid() {
		return 0, false
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint()), true
	case reflect.String:
		i, err := strconv.Atoi(strings.TrimSpace(v.String()))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func valueToBool(v reflect.Value) (bool, bool) {
	v = indirectValue(v)
	if !v.IsValid() {
		return false, false
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool(), true
	case reflect.String:
		text := strings.ToLower(strings.TrimSpace(v.String()))
		switch text {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}
