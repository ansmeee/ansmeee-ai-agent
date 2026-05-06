package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code defines a unified response code.
type Code int

const (
	CodeSuccess      Code = 0
	CodeBadRequest   Code = 1001
	CodeUnauthorized Code = 1002
	CodeNotFound     Code = 1004
	CodeInternal     Code = 5000
)

// APIResponse is the unified API response envelope.
type APIResponse struct {
	Code    Code        `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// OK sends a success response.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Fail sends an error response with the given HTTP status and response code.
func Fail(c *gin.Context, httpStatus int, code Code, message string) {
	c.JSON(httpStatus, APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// BadRequest sends a 400 response.
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, CodeBadRequest, message)
}

// InternalError sends a 500 response.
func InternalError(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, CodeInternal, message)
}
