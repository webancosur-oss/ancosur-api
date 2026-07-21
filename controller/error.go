package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorUser(
	c *gin.Context,
	err error,
	statusCode int,
) {
	message := "Error de usuario"

	if err != nil {
		message = err.Error()
	}

	Error(
		c,
		statusCode,
		"user.error",
		message,
		nil,
	)
}

func ErrorServer(
	c *gin.Context,
	err error,
) {
	message := "Error de servidor"

	if err != nil {
		message = err.Error()
	}

	Error(
		c,
		http.StatusInternalServerError,
		"server.error",
		message,
		nil,
	)
}

func ErrorsError(
	c *gin.Context,
	err error,
) {
	message := "Error"

	if err != nil {
		message = err.Error()
	}

	Error(
		c,
		http.StatusBadRequest,
		"request.error",
		message,
		nil,
	)
}

func ErrorsWarning(
	c *gin.Context,
	err error,
) {
	message := "Warning"

	if err != nil {
		message = err.Error()
	}

	Error(
		c,
		http.StatusBadRequest,
		"request.warning",
		message,
		nil,
	)
}

func ErrorsSuccess(
	c *gin.Context,
	message string,
	data any,
) {
	Success(
		c,
		http.StatusOK,
		"request.success",
		message,
		data,
	)
}

func ErrorsInfo(
	c *gin.Context,
	message string,
	data any,
) {
	Success(
		c,
		http.StatusOK,
		"request.info",
		message,
		data,
	)
}