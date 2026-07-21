package routes

import (
	"ancosur-api/controller"
	middleware "ancosur-api/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EstadoLeadRoutes struct {
	DB *pgxpool.Pool
}

type EstadoLeadResponse struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
}

func RutasEstadoLeads(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	estadoRoutes := &EstadoLeadRoutes{
		DB: db,
	}

	protected := api.Group("/estado-leads")
	protected.Use(middleware.AuthMiddleware())

	protected.GET("", estadoRoutes.GetAllEstadoLeads)
}

func (h *EstadoLeadRoutes) GetAllEstadoLeads(
	c *gin.Context,
) {
	rows, err := h.DB.Query(
		c.Request.Context(),
		`
			SELECT
				id::text,
				nombre
			FROM estado_leads
			ORDER BY
				CASE LOWER(nombre)
					WHEN 'nuevo' THEN 1
					WHEN 'contactado' THEN 2
					WHEN 'atendido' THEN 3
					WHEN 'calificado' THEN 4
					WHEN 'seguimiento' THEN 5
					WHEN 'visita / proforma' THEN 6
					WHEN 'separado' THEN 7
					WHEN 'vendido' THEN 8
					WHEN 'perdido' THEN 9
					WHEN 'descartado' THEN 10
					ELSE 99
				END,
				nombre ASC
		`,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"estado_leads.list_error",
			"No se pudo obtener los estados de leads.",
			nil,
		)
		return
	}

	defer rows.Close()

	items := []EstadoLeadResponse{}

	for rows.Next() {
		var item EstadoLeadResponse

		err := rows.Scan(
			&item.ID,
			&item.Nombre,
		)

		if err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"estado_leads.scan_error",
				"No se pudo leer un estado de lead.",
				nil,
			)
			return
		}

		items = append(items, item)
	}

	controller.Success(
		c,
		http.StatusOK,
		"estado_leads.list",
		"Estados de leads obtenidos correctamente.",
		gin.H{
			"items": items,
			"total": len(items),
		},
	)
}