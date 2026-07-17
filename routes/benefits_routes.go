package routes

import (
	"ancosur-api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterBenefitsRoutes(
	api *gin.RouterGroup,
	benefitsHandler *handlers.BenefitsHandler,
) {
	benefits := api.Group("/benefits")
	{
		terrains := benefits.Group("/terrains")
		{
			terrains.POST(
				"",
				benefitsHandler.CreateTerrain,
			)

			terrains.GET(
				"",
				benefitsHandler.GetTerrains,
			)

			terrains.GET(
				"/:id",
				benefitsHandler.GetTerrainByID,
			)

			terrains.PUT(
				"/:id",
				benefitsHandler.UpdateTerrain,
			)

			terrains.DELETE(
				"/:id",
				benefitsHandler.DeleteTerrain,
			)
		}

		referrals := benefits.Group("/referrals")
		{
			referrals.POST(
				"",
				benefitsHandler.CreateReferral,
			)

			referrals.GET(
				"",
				benefitsHandler.GetReferrals,
			)

			referrals.GET(
				"/:id",
				benefitsHandler.GetReferralByID,
			)

			referrals.PUT(
				"/:id",
				benefitsHandler.UpdateReferral,
			)

			referrals.DELETE(
				"/:id",
				benefitsHandler.DeleteReferral,
			)
		}
	}

	/*
		Alias para los endpoints que ya utiliza
		el frontend actual.
	*/
	api.POST(
		"/terrain-leads",
		benefitsHandler.CreateTerrain,
	)

	api.POST(
		"/referral-leads",
		benefitsHandler.CreateReferral,
	)
}