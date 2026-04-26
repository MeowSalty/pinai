package provider

import "errors"

var (
	ErrResourceNotFound  = errors.New("资源未找到")
	ErrResourceNotBelong = errors.New("资源不属于该平台")
	ErrInvalidArgument   = errors.New("请求参数不合法")
	ErrDefaultConflict   = errors.New("默认端点冲突")
	ErrTaskNotFound      = errors.New("任务未找到")
)
