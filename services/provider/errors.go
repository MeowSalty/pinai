package provider

import "errors"

var (
	ErrResourceNotFound  = errors.New("资源未找到")
	ErrResourceNotBelong = errors.New("资源不属于该平台")
)
