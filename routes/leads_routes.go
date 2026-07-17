package routes

import (
	"ancosur-api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterLeadRoutes(
	api *gin.RouterGroup,
	leadHandler *handlers.LeadHandler,
) {
	leads := api.Group("/leads")
	{
		// Crear un lead
		leads.POST(
			"",
			leadHandler.CreateLead,
		)

		// Listar todos los leads
		leads.GET(
			"",
			leadHandler.GetAllLeads,
		)

		// Obtener un lead por ID
		leads.GET(
			"/:id",
			leadHandler.GetLeadByID,
		)

		// Actualizar un lead
		leads.PUT(
			"/:id",
			leadHandler.UpdateLead,
		)

		// Actualización parcial
		leads.PATCH(
			"/:id",
			leadHandler.UpdateLead,
		)

		// Eliminación lógica
		leads.DELETE(
			"/:id",
			leadHandler.DeleteLead,
		)
	}
}