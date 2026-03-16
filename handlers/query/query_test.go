package query

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestContext(t *testing.T, rawQuery string) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	c.Request = req
	return c
}

func TestPagination_Missing_UseDefault(t *testing.T) {
	c := newTestContext(t, "")

	page, pageSize, err := Pagination(c)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if page != 1 {
		t.Fatalf("expected page=1, got %d", page)
	}
	if pageSize != 10 {
		t.Fatalf("expected pageSize=10, got %d", pageSize)
	}
}

func TestPagination_Valid_ParseOK(t *testing.T) {
	c := newTestContext(t, "page=2&page_size=20")

	page, pageSize, err := Pagination(c)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if page != 2 {
		t.Fatalf("expected page=2, got %d", page)
	}
	if pageSize != 20 {
		t.Fatalf("expected pageSize=20, got %d", pageSize)
	}
}

func TestPagination_Invalid_ReturnError(t *testing.T) {
	t.Parallel()

	cases := []string{
		"page=abc",
		"page=-1",
		"page_size=foo",
		"page_size=-1",
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			c := newTestContext(t, tc)
			_, _, err := Pagination(c)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestPagination_PageSize_ClampTo100(t *testing.T) {
	c := newTestContext(t, "page_size=200")

	page, pageSize, err := Pagination(c)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if page != 1 {
		t.Fatalf("expected page=1, got %d", page)
	}
	if pageSize != 100 {
		t.Fatalf("expected pageSize=100, got %d", pageSize)
	}
}

func TestOptionalBool_Missing_ReturnNil(t *testing.T) {
	c := newTestContext(t, "")

	v, err := OptionalBool(c, "success")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v != nil {
		t.Fatalf("expected nil, got %v", *v)
	}
}

func TestOptionalBool_Valid_ParseOK(t *testing.T) {
	c := newTestContext(t, "success=true")

	v, err := OptionalBool(c, "success")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v == nil || *v != true {
		t.Fatalf("expected true, got %v", v)
	}
}

func TestOptionalBool_Invalid_ReturnError(t *testing.T) {
	c := newTestContext(t, "success=foo")

	_, err := OptionalBool(c, "success")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOptionalUint_Missing_ReturnNil(t *testing.T) {
	c := newTestContext(t, "")

	v, err := OptionalUint(c, "platform_id")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v != nil {
		t.Fatalf("expected nil, got %v", *v)
	}
}

func TestOptionalUint_Valid_ParseOK(t *testing.T) {
	c := newTestContext(t, "platform_id=12")

	v, err := OptionalUint(c, "platform_id")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v == nil || *v != 12 {
		t.Fatalf("expected 12, got %v", v)
	}
}

func TestOptionalUint_Invalid_ReturnError(t *testing.T) {
	t.Parallel()

	cases := []string{
		"platform_id=abc",
		"platform_id=-1",
		"platform_id=0",
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			c := newTestContext(t, tc)
			_, err := OptionalUint(c, "platform_id")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
