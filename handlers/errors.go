package handlers

import (
	"errors"
	"log"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e AppError) Error() string {
	return e.Message
}

var (
	ErrNotFound     = &AppError{Code: "NOT_FOUND", Message: "记录不存在"}
	ErrUnauthorized = &AppError{Code: "UNAUTHORIZED", Message: "未授权操作"}
	ErrForbidden    = &AppError{Code: "FORBIDDEN", Message: "禁止访问"}
	ErrBadRequest   = &AppError{Code: "BAD_REQUEST", Message: "请求参数错误"}
	ErrServerError  = &AppError{Code: "SERVER_ERROR", Message: "服务器错误"}
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

func HandleError(err error) ErrorResponse {
	if err == nil {
		return ErrorResponse{Error: "未知错误"}
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return ErrorResponse{Error: appErr.Message, Code: appErr.Code}
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrorResponse{Error: "记录不存在", Code: "NOT_FOUND"}
	default:
		log.Printf("[ERROR] %v", err)
		return ErrorResponse{Error: "服务器错误", Code: "INTERNAL_ERROR"}
	}
}

func RespondWithError(c *gin.Context, code int, err error) {
	resp := HandleError(err)
	c.JSON(code, resp)
}
