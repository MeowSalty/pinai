package query

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

// Pagination 解析分页参数。缺失用默认值，非法返回 error。
//
// page 默认 1，page_size 默认 10，上限 100。
func Pagination(c *gin.Context) (page, pageSize int, err error) {
	page = defaultPage
	pageSize = defaultPageSize

	if pageStr := c.Query("page"); pageStr != "" {
		p, convErr := strconv.Atoi(pageStr)
		if convErr != nil || p <= 0 {
			return 0, 0, fmt.Errorf("page 参数无效，应为正整数")
		}
		page = p
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		ps, convErr := strconv.Atoi(pageSizeStr)
		if convErr != nil || ps <= 0 {
			return 0, 0, fmt.Errorf("page_size 参数无效，应为正整数")
		}
		if ps > maxPageSize {
			ps = maxPageSize
		}
		pageSize = ps
	}

	return page, pageSize, nil
}

// OptionalBool 解析可选布尔参数。缺失返回 nil，非法返回 error。
func OptionalBool(c *gin.Context, key string) (*bool, error) {
	valStr := c.Query(key)
	if valStr == "" {
		return nil, nil
	}

	v, err := strconv.ParseBool(valStr)
	if err != nil {
		return nil, fmt.Errorf("%s 参数无效，应为布尔值", key)
	}

	return &v, nil
}

// OptionalUint 解析可选正整数参数。缺失返回 nil，非法返回 error。
func OptionalUint(c *gin.Context, key string) (*uint, error) {
	valStr := c.Query(key)
	if valStr == "" {
		return nil, nil
	}

	v, err := strconv.ParseUint(valStr, 10, 0)
	if err != nil || v == 0 {
		return nil, fmt.Errorf("%s 参数无效，应为正整数", key)
	}

	u := uint(v)
	return &u, nil
}
