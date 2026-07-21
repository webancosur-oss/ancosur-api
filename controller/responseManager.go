package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseManager struct {
	Success    bool   `json:"success"`
	Response   string `json:"response"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
	Data       any    `json:"data"`
}

func NewResponseManager() *ResponseManager {
	return &ResponseManager{
		Success:    true,
		Response:   "success",
		Message:    "Success",
		StatusCode: http.StatusOK,
		Status:     "Success",
		Data:       gin.H{},
	}
}

func Success(
	c *gin.Context,
	statusCode int,
	response string,
	message string,
	data any,
) {
	if data == nil {
		data = gin.H{}
	}

	c.JSON(statusCode, ResponseManager{
		Success:    true,
		Response:   response,
		Message:    message,
		StatusCode: statusCode,
		Status:     "Success",
		Data:       data,
	})
}

func Error(
	c *gin.Context,
	statusCode int,
	response string,
	message string,
	data any,
) {
	c.JSON(statusCode, ResponseManager{
		Success:    false,
		Response:   response,
		Message:    message,
		StatusCode: statusCode,
		Status:     "Error",
		Data:       data,
	})
}

func AbortError(
	c *gin.Context,
	statusCode int,
	response string,
	message string,
	data any,
) {
	c.AbortWithStatusJSON(statusCode, ResponseManager{
		Success:    false,
		Response:   response,
		Message:    message,
		StatusCode: statusCode,
		Status:     "Error",
		Data:       data,
	})
}