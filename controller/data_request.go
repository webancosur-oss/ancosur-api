package controller

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CheckBody(
	c *gin.Context,
	dest any,
) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		Error(
			c,
			http.StatusBadRequest,
			"request.invalid_json",
			"El JSON enviado no es válido.",
			gin.H{
				"error": err.Error(),
			},
		)

		return false
	}

	return true
}

func CheckBodyMap(
	c *gin.Context,
) (map[string]interface{}, bool) {
	var dataRequest map[string]interface{}

	if err := c.ShouldBindJSON(&dataRequest); err != nil {
		Error(
			c,
			http.StatusBadRequest,
			"request.invalid_json",
			"El JSON enviado no es válido.",
			gin.H{
				"error": err.Error(),
			},
		)

		return nil, false
	}

	if len(dataRequest) <= 0 {
		Error(
			c,
			http.StatusBadRequest,
			"request.empty_body",
			"No se obtuvo información.",
			nil,
		)

		return nil, false
	}

	return dataRequest, true
}

func CheckBodyArray(
	c *gin.Context,
) ([]map[string]interface{}, bool) {
	var dataRequest []map[string]interface{}

	if err := c.ShouldBindJSON(&dataRequest); err != nil {
		Error(
			c,
			http.StatusBadRequest,
			"request.invalid_json",
			"El JSON enviado no es válido.",
			gin.H{
				"error": err.Error(),
			},
		)

		return nil, false
	}

	if len(dataRequest) <= 0 {
		Error(
			c,
			http.StatusBadRequest,
			"request.empty_body",
			"No se obtuvo información.",
			nil,
		)

		return nil, false
	}

	return dataRequest, true
}

func CheckRawBody(
	c *gin.Context,
) ([]byte, bool) {
	body, err := io.ReadAll(c.Request.Body)

	if err != nil {
		Error(
			c,
			http.StatusInternalServerError,
			"request.body_error",
			"No se pudo leer el cuerpo de la solicitud.",
			gin.H{
				"error": err.Error(),
			},
		)

		return nil, false
	}

	if len(body) <= 0 {
		Error(
			c,
			http.StatusBadRequest,
			"request.empty_body",
			"No se obtuvo información.",
			nil,
		)

		return nil, false
	}

	return body, true
}

func Required(
	c *gin.Context,
	value string,
	fieldName string,
) bool {
	if value == "" {
		Error(
			c,
			http.StatusBadRequest,
			"request.required_field",
			"El campo "+fieldName+" es obligatorio.",
			nil,
		)

		return false
	}

	return true
}

func NewRequestError(
	message string,
) error {
	return errors.New(message)
}