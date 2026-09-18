package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"ancosur-api/controller"
	"ancosur-api/services"
)

func ProyectosWebRoutes(
	router *gin.Engine,
	db *pgxpool.Pool,
) {
	service :=
		services.NewProyectoWebService(
			db,
		)

	controller :=
		controller.NewProyectosWebController(
			service,
		)

	proyectos :=
		router.Group(
			"/api/web/proyectos",
		)

	// LISTAR
	proyectos.GET(
		"",
		controller.Listar,
	)

	// CREAR
	proyectos.POST(
		"",
		controller.Crear,
	)

	// OBTENER POR SLUG
	proyectos.GET(
		"/slug/:slug",
		controller.ObtenerPorSlug,
	)

	// OBTENER POR ID
	proyectos.GET(
		"/:id",
		controller.Obtener,
	)

	// ACTUALIZAR
	proyectos.PUT(
		"/:id",
		controller.Actualizar,
	)

	// ELIMINAR
	proyectos.DELETE(
		"/:id",
		controller.Eliminar,
	)
}