package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/* =========================================================
   MODELOS
========================================================= */

type BlogPost struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Slug          string          `json:"slug"`
	Category      string          `json:"category"`
	Excerpt       string          `json:"excerpt"`
	CoverImageURL *string         `json:"cover_image_url"`
	Content       json.RawMessage `json:"content"`
	Status        string          `json:"status"`
	AuthorName    *string         `json:"author_name"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	PublishedAt   *time.Time      `json:"published_at"`
}

/* =========================================================
   REQUEST CREAR
========================================================= */

type CreateBlogRequest struct {
	Title         string          `json:"title" binding:"required"`
	Slug          string          `json:"slug" binding:"required"`
	Category      string          `json:"category" binding:"required"`
	Excerpt       string          `json:"excerpt"`
	CoverImageURL *string         `json:"cover_image_url"`
	Content       json.RawMessage `json:"content" binding:"required"`
	Status        string          `json:"status"`
	AuthorName    string          `json:"author_name"`
}

/* =========================================================
   REQUEST ACTUALIZAR
========================================================= */

type UpdateBlogRequest struct {
	Title         string          `json:"title"`
	Slug          string          `json:"slug"`
	Category      string          `json:"category"`
	Excerpt       string          `json:"excerpt"`
	CoverImageURL *string         `json:"cover_image_url"`
	Content       json.RawMessage `json:"content"`
	Status        string          `json:"status"`
	AuthorName    string          `json:"author_name"`
}

/* =========================================================
   RUTAS BLOG
========================================================= */

func RutasBlog(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	blog := api.Group("/blog")

	/* =====================================================
	   IMÁGENES
	===================================================== */

	blog.POST(
		"/imagen",
		SubirImagenBlog(db),
	)

	blog.GET(
		"/imagen/:id",
		VerImagenBlog(db),
	)

	/* =====================================================
	   ADMIN
	===================================================== */

	blog.GET(
		"/admin/:id",
		ObtenerBlogPorID(db),
	)

	/* =====================================================
	   CREAR
	===================================================== */

	blog.POST(
		"",
		CrearBlog(db),
	)

	/* =====================================================
	   LISTAR
	===================================================== */

	blog.GET(
		"",
		ListarBlog(db),
	)

	/* =====================================================
	   ACTUALIZAR
	   
	   IMPORTANTE:
	   Acepta tanto ID como SLUG.

	   PATCH /api/blog/:id
	   PATCH /api/blog/:slug
	===================================================== */

	blog.PATCH(
		"/:id",
		ActualizarBlog(db),
	)

	/* =====================================================
	   ELIMINAR
	   
	   Acepta tanto ID como SLUG.
	===================================================== */

	blog.DELETE(
		"/:id",
		EliminarBlog(db),
	)

	/* =====================================================
	   WEB PÚBLICA
	   
	   GET /api/blog/:slug

	   Esta ruta debe permanecer al final.
	===================================================== */

	blog.GET(
		"/:slug",
		ObtenerBlogPorSlug(db),
	)
}

/* =========================================================
   HELPERS
========================================================= */

func normalizeBlogStatus(
	status string,
) string {
	status = strings.ToLower(
		strings.TrimSpace(status),
	)

	if status == "publicado" {
		return "publicado"
	}

	return "borrador"
}

func normalizeSlug(
	slug string,
) string {
	return strings.Trim(
		strings.ToLower(
			slug,
		),
		" ",
	)
}

/* =========================================================
   VALIDAR CONTENT
========================================================= */

func validateBlogContent(
	content json.RawMessage,
) error {
	if len(content) == 0 {
		return nil
	}

	var value interface{}

	if err := json.Unmarshal(
		content,
		&value,
	); err != nil {
		return err
	}

	return nil
}

/* =========================================================
   CREAR BLOG
========================================================= */

func CrearBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		var req CreateBlogRequest

		if err :=
			c.ShouldBindJSON(&req); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Datos del artículo inválidos.",
					"error":   err.Error(),
				},
			)

			return
		}

		req.Title =
			strings.TrimSpace(
				req.Title,
			)

		req.Slug =
			normalizeSlug(
				req.Slug,
			)

		req.Category =
			strings.TrimSpace(
				req.Category,
			)

		req.Excerpt =
			strings.TrimSpace(
				req.Excerpt,
			)

		req.AuthorName =
			strings.TrimSpace(
				req.AuthorName,
			)

		/* =================================================
		   VALIDACIONES
		================================================= */

		if req.Title == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El título es obligatorio.",
				},
			)

			return
		}

		if req.Slug == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El slug es obligatorio.",
				},
			)

			return
		}

		if req.Category == "" {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La categoría es obligatoria.",
				},
			)

			return
		}

		if len(req.Content) == 0 {
			req.Content =
				json.RawMessage(`[]`)
		}

		if err :=
			validateBlogContent(
				req.Content,
			); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El contenido del artículo no tiene un JSON válido.",
					"error":   err.Error(),
				},
			)

			return
		}

		status :=
			normalizeBlogStatus(
				req.Status,
			)

		if req.AuthorName == "" {
			req.AuthorName =
				"ANCOSUR Inmobiliaria"
		}

		/* =================================================
		   CONTEXTO
		================================================= */

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		/* =================================================
		   VERIFICAR SLUG
		================================================= */

		var exists bool

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT EXISTS(
					SELECT 1
					FROM blog_posts
					WHERE slug = $1
				)
				`,
				req.Slug,
			).Scan(&exists)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo verificar el slug.",
					"error":   err.Error(),
				},
			)

			return
		}

		if exists {

			c.JSON(
				http.StatusConflict,
				gin.H{
					"success": false,
					"message": "El slug ya existe. Utiliza otro slug.",
				},
			)

			return
		}

		/* =================================================
		   ID
		================================================= */

		id :=
			uuid.New()

		var publishedAt *time.Time

		if status == "publicado" {

			now :=
				time.Now()

			publishedAt =
				&now
		}

		/* =================================================
		   INSERT
		================================================= */

		var post BlogPost

		err =
			db.QueryRow(
				ctx,
				`
				INSERT INTO blog_posts (
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					$6,
					$7::jsonb,
					$8,
					$9,
					NOW(),
					NOW(),
					$10
				)
				RETURNING
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				`,
				id,
				req.Title,
				req.Slug,
				req.Category,
				req.Excerpt,
				req.CoverImageURL,
				req.Content,
				status,
				req.AuthorName,
				publishedAt,
			).
			Scan(
				&post.ID,
				&post.Title,
				&post.Slug,
				&post.Category,
				&post.Excerpt,
				&post.CoverImageURL,
				&post.Content,
				&post.Status,
				&post.AuthorName,
				&post.CreatedAt,
				&post.UpdatedAt,
				&post.PublishedAt,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo crear el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "Artículo creado correctamente.",
				"data":    post,
			},
		)
	}
}

/* =========================================================
   LISTAR BLOG
========================================================= */

func ListarBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		status :=
			strings.TrimSpace(
				c.Query("status"),
			)

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		query := `
			SELECT
				id,
				title,
				slug,
				category,
				excerpt,
				cover_image_url,
				content,
				status,
				author_name,
				created_at,
				updated_at,
				published_at
			FROM blog_posts
		`

		args :=
			[]interface{}{}

		if status != "" {

			status =
				normalizeBlogStatus(
					status,
				)

			query += `
				WHERE status = $1
			`

			args =
				append(
					args,
					status,
				)
		}

		query += `
			ORDER BY created_at DESC
		`

		rows, err :=
			db.Query(
				ctx,
				query,
				args...,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudieron cargar los artículos.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer rows.Close()

		posts :=
			make(
				[]BlogPost,
				0,
			)

		for rows.Next() {

			var post BlogPost

			err :=
				rows.Scan(
					&post.ID,
					&post.Title,
					&post.Slug,
					&post.Category,
					&post.Excerpt,
					&post.CoverImageURL,
					&post.Content,
					&post.Status,
					&post.AuthorName,
					&post.CreatedAt,
					&post.UpdatedAt,
					&post.PublishedAt,
				)

			if err != nil {

				c.JSON(
					http.StatusInternalServerError,
					gin.H{
						"success": false,
						"message": "Error leyendo los artículos.",
						"error":   err.Error(),
					},
				)

				return
			}

			posts =
				append(
					posts,
					post,
				)
		}

		if err :=
			rows.Err(); err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "Error procesando los artículos.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"total":   len(posts),
				"data":    posts,
			},
		)
	}
}

/* =========================================================
   OBTENER POR ID
========================================================= */

func ObtenerBlogPorID(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id :=
			strings.TrimSpace(
				c.Param("id"),
			)

		if id == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "ID inválido.",
				},
			)

			return
		}

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var post BlogPost

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				FROM blog_posts
				WHERE id = $1
				`,
				id,
			).
			Scan(
				&post.ID,
				&post.Title,
				&post.Slug,
				&post.Category,
				&post.Excerpt,
				&post.CoverImageURL,
				&post.Content,
				&post.Status,
				&post.AuthorName,
				&post.CreatedAt,
				&post.UpdatedAt,
				&post.PublishedAt,
			)

		if err != nil {

			if err ==
				pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Artículo no encontrado.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    post,
			},
		)
	}
}

/* =========================================================
   OBTENER POR SLUG
========================================================= */

func ObtenerBlogPorSlug(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		slug :=
			normalizeSlug(
				c.Param("slug"),
			)

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var post BlogPost

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				FROM blog_posts
				WHERE slug = $1
				  AND status = 'publicado'
				`,
				slug,
			).
			Scan(
				&post.ID,
				&post.Title,
				&post.Slug,
				&post.Category,
				&post.Excerpt,
				&post.CoverImageURL,
				&post.Content,
				&post.Status,
				&post.AuthorName,
				&post.CreatedAt,
				&post.UpdatedAt,
				&post.PublishedAt,
			)

		if err != nil {

			if err ==
				pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Artículo no encontrado.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"data":    post,
			},
		)
	}
}

/* =========================================================
   ACTUALIZAR BLOG
========================================================= */

func ActualizarBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		/*
			IMPORTANTE:

			El parámetro puede ser:

			/api/blog/UUID
			o
			/api/blog/mi-slug

			El frontend actualmente manda SLUG.
		*/

		identifier :=
			strings.TrimSpace(
				c.Param("id"),
			)

		if identifier == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Identificador del artículo inválido.",
				},
			)

			return
		}

		var req UpdateBlogRequest

		if err :=
			c.ShouldBindJSON(&req); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Datos inválidos.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =================================================
		   NORMALIZAR
		================================================= */

		req.Title =
			strings.TrimSpace(
				req.Title,
			)

		req.Slug =
			normalizeSlug(
				req.Slug,
			)

		req.Category =
			strings.TrimSpace(
				req.Category,
			)

		req.Excerpt =
			strings.TrimSpace(
				req.Excerpt,
			)

		req.AuthorName =
			strings.TrimSpace(
				req.AuthorName,
			)

		status :=
			normalizeBlogStatus(
				req.Status,
			)

		if req.AuthorName == "" {
			req.AuthorName =
				"ANCOSUR Inmobiliaria"
		}

		/* =================================================
		   VALIDACIONES
		================================================= */

		if req.Title == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El título es obligatorio.",
				},
			)

			return
		}

		if req.Slug == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El slug es obligatorio.",
				},
			)

			return
		}

		if req.Category == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La categoría es obligatoria.",
				},
			)

			return
		}

		if len(req.Content) == 0 {

			req.Content =
				json.RawMessage(`[]`)
		}

		if err :=
			validateBlogContent(
				req.Content,
			); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "El contenido no tiene JSON válido.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =================================================
		   CONTEXTO
		================================================= */

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		/* =================================================
		   BUSCAR ARTÍCULO
		   
		   POR ID O POR SLUG
		================================================= */

		var (
			currentID          string
			currentSlug        string
			currentPublishedAt *time.Time
		)

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					id,
					slug,
					published_at
				FROM blog_posts
				WHERE id::text = $1
				   OR slug = $1
				LIMIT 1
				`,
				identifier,
			).
			Scan(
				&currentID,
				&currentSlug,
				&currentPublishedAt,
			)

		if err != nil {

			if err ==
				pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "El artículo que intentas actualizar no existe.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo localizar el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =================================================
		   VERIFICAR SLUG DUPLICADO
		   
		   EXCLUIMOS EL PROPIO ARTÍCULO.
		================================================= */

		var slugExists bool

		err =
			db.QueryRow(
				ctx,
				`
				SELECT EXISTS(
					SELECT 1
					FROM blog_posts
					WHERE slug = $1
					  AND id::text <> $2
				)
				`,
				req.Slug,
				currentID,
			).
			Scan(
				&slugExists,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo verificar el nuevo slug.",
					"error":   err.Error(),
				},
			)

			return
		}

		if slugExists {

			c.JSON(
				http.StatusConflict,
				gin.H{
					"success": false,
					"message": "El slug ya pertenece a otro artículo.",
				},
			)

			return
		}

		/* =================================================
		   PUBLISHED_AT
		================================================= */

		var publishedAt *time.Time

		if status == "publicado" {

			/*
				Si ya estaba publicado,
				conservamos la fecha original.
			*/

			if currentPublishedAt != nil {

				publishedAt =
					currentPublishedAt

			} else {

				now :=
					time.Now()

				publishedAt =
					&now
			}

		} else {

			/*
				Si vuelve a borrador,
				quitamos published_at.
			*/

			publishedAt = nil
		}

		/* =================================================
		   ACTUALIZAR
		================================================= */

		var post BlogPost

		err =
			db.QueryRow(
				ctx,
				`
				UPDATE blog_posts
				SET
					title = $1,
					slug = $2,
					category = $3,
					excerpt = $4,
					cover_image_url = $5,
					content = $6::jsonb,
					status = $7,
					author_name = $8,
					updated_at = NOW(),
					published_at = $9
				WHERE id::text = $10

				RETURNING
					id,
					title,
					slug,
					category,
					excerpt,
					cover_image_url,
					content,
					status,
					author_name,
					created_at,
					updated_at,
					published_at
				`,
				req.Title,
				req.Slug,
				req.Category,
				req.Excerpt,
				req.CoverImageURL,
				req.Content,
				status,
				req.AuthorName,
				publishedAt,
				currentID,
			).
			Scan(
				&post.ID,
				&post.Title,
				&post.Slug,
				&post.Category,
				&post.Excerpt,
				&post.CoverImageURL,
				&post.Content,
				&post.Status,
				&post.AuthorName,
				&post.CreatedAt,
				&post.UpdatedAt,
				&post.PublishedAt,
			)

		if err != nil {

			/*
				Esto ya NO debe convertirse
				en un 500 genérico sin información.
			*/

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo actualizar el artículo.",
					"error":   err.Error(),
					"details": gin.H{
						"identifier":  identifier,
						"current_id":  currentID,
						"current_slug": currentSlug,
					},
				},
			)

			return
		}

		/* =================================================
		   RESPUESTA
		================================================= */

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Artículo actualizado correctamente.",
				"data":    post,
			},
		)
	}
}

/* =========================================================
   ELIMINAR
========================================================= */

func EliminarBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		/*
			Aceptamos ID o SLUG.
		*/

		identifier :=
			strings.TrimSpace(
				c.Param("id"),
			)

		if identifier == "" {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Identificador inválido.",
				},
			)

			return
		}

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		/* =================================================
		   ELIMINAR POR ID O SLUG
		================================================= */

		result, err :=
			db.Exec(
				ctx,
				`
				DELETE FROM blog_posts
				WHERE id::text = $1
				   OR slug = $1
				`,
				identifier,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo eliminar el artículo.",
					"error":   err.Error(),
				},
			)

			return
		}

		if result.RowsAffected() == 0 {

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"success": false,
					"message": "Artículo no encontrado.",
				},
			)

			return
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"success": true,
				"message": "Artículo eliminado correctamente.",
			},
		)
	}
}

/* =========================================================
   SUBIR IMAGEN
========================================================= */

func SubirImagenBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		fileHeader, err :=
			c.FormFile("image")

		if err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "No se recibió ninguna imagen.",
				},
			)

			return
		}

		/* =================================================
		   TAMAÑO
		================================================= */

		if fileHeader.Size >
			8*1024*1024 {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "La imagen no puede superar los 8 MB.",
				},
			)

			return
		}

		/* =================================================
		   TIPO
		================================================= */

		contentType :=
			fileHeader.Header.Get(
				"Content-Type",
			)

		allowedTypes :=
			map[string]bool{
				"image/jpeg": true,
				"image/png":  true,
				"image/webp": true,
			}

		if !allowedTypes[
			contentType,
		] {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"success": false,
					"message": "Formato de imagen no permitido. Usa JPG, PNG o WEBP.",
				},
			)

			return
		}

		/* =================================================
		   ABRIR
		================================================= */

		file, err :=
			fileHeader.Open()

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo abrir la imagen.",
					"error":   err.Error(),
				},
			)

			return
		}

		defer file.Close()

		/*
			Usamos io.ReadAll en lugar de
			file.Read para garantizar que
			se lea el archivo completo.
		*/

		data, err :=
			io.ReadAll(file)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo leer la imagen.",
					"error":   err.Error(),
				},
			)

			return
		}

		/* =================================================
		   ID
		================================================= */

		id :=
			uuid.New()

		filename :=
			fileHeader.Filename

		/* =================================================
		   DB
		================================================= */

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				20*time.Second,
			)

		defer cancel()

		_, err =
			db.Exec(
				ctx,
				`
				INSERT INTO blog_images (
					id,
					file_name,
					content_type,
					file_size,
					file_data,
					created_at
				)
				VALUES (
					$1,
					$2,
					$3,
					$4,
					$5,
					NOW()
				)
				`,
				id,
				filename,
				contentType,
				int64(len(data)),
				data,
			)

		if err != nil {

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo guardar la imagen.",
					"error":   err.Error(),
				},
			)

			return
		}

		imageURL :=
			fmt.Sprintf(
				"/api/blog/imagen/%s",
				id.String(),
			)

		c.JSON(
			http.StatusCreated,
			gin.H{
				"success": true,
				"message": "Imagen subida correctamente.",
				"data": gin.H{
					"id":  id.String(),
					"url": imageURL,
				},
			},
		)
	}
}

/* =========================================================
   SERVIR IMAGEN
========================================================= */

func VerImagenBlog(
	db *pgxpool.Pool,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		id :=
			strings.TrimSpace(
				c.Param("id"),
			)

		ctx, cancel :=
			context.WithTimeout(
				c.Request.Context(),
				10*time.Second,
			)

		defer cancel()

		var (
			contentType string
			data        []byte
		)

		err :=
			db.QueryRow(
				ctx,
				`
				SELECT
					content_type,
					file_data
				FROM blog_images
				WHERE id = $1
				`,
				id,
			).
			Scan(
				&contentType,
				&data,
			)

		if err != nil {

			if err ==
				pgx.ErrNoRows {

				c.JSON(
					http.StatusNotFound,
					gin.H{
						"success": false,
						"message": "Imagen no encontrada.",
					},
				)

				return
			}

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"success": false,
					"message": "No se pudo obtener la imagen.",
					"error":   err.Error(),
				},
			)

			return
		}

		c.Data(
			http.StatusOK,
			contentType,
			data,
		)
	}
}