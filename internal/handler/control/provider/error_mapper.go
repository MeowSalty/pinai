package provider

import (
	"errors"

	serviceprovider "github.com/MeowSalty/pinai/internal/app/provider"
	"github.com/MeowSalty/pinai/internal/handler/response"

	"github.com/gin-gonic/gin"
)

func respondProviderServiceError(c *gin.Context, err error, notFoundMessage, internalMessage string) {
	if errors.Is(err, serviceprovider.ErrResourceNotFound) {
		response.NotFound(c, notFoundMessage)
		return
	}

	if errors.Is(err, serviceprovider.ErrTaskNotFound) {
		response.NotFound(c, notFoundMessage)
		return
	}

	if errors.Is(err, serviceprovider.ErrResourceNotBelong) ||
		errors.Is(err, serviceprovider.ErrInvalidArgument) ||
		errors.Is(err, serviceprovider.ErrDefaultConflict) {
		response.BadRequest(c, err.Error())
		return
	}

	response.InternalError(c, internalMessage)
}
