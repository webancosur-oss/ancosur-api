package routes

import (
	"ancosur-api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterInvestmentRoutes(
	api *gin.RouterGroup,
	investmentHandler *handlers.InvestmentHandler,
) {
	investments := api.Group("/investments")
	{
		investments.POST(
			"",
			investmentHandler.CreateInvestment,
		)

		investments.GET(
			"",
			investmentHandler.GetInvestments,
		)

		investments.GET(
			"/:id",
			investmentHandler.GetInvestmentByID,
		)

		investments.PUT(
			"/:id",
			investmentHandler.UpdateInvestment,
		)

		investments.PATCH(
			"/:id",
			investmentHandler.UpdateInvestment,
		)

		investments.DELETE(
			"/:id",
			investmentHandler.DeleteInvestment,
		)
	}

	api.POST(
		"/investment-leads",
		investmentHandler.CreateInvestmentLead,
	)
}
