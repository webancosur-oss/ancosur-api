package routes

import (
	"ancosur-api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterLeadRoutes(api *gin.RouterGroup, leadHandler *handlers.LeadHandler) {
	leads := api.Group("/leads")
	{
		leads.GET("", leadHandler.GetAllLeads)
		leads.POST("", leadHandler.CreateLead)

	}
}
