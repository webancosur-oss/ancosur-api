package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
===========================================================
MODELO PROYECTO WEB
===========================================================
*/

type ProyectoWeb struct {
	ID string `json:"id"`

	Codigo string `json:"codigo"`

	Titulo string `json:"titulo"`

	Slug string `json:"slug"`

	Tipo string `json:"tipo"`

	Ciudad string `json:"ciudad"`

	Direccion string `json:"direccion"`

	Etapa string `json:"etapa"`

	Imagen string `json:"imagen"`

	Dormitorios *string `json:"dormitorios"`

	MetrajeDesde *float64 `json:"metraje_desde"`

	MetrajeHasta *float64 `json:"metraje_hasta"`

	Estado string `json:"estado"`

	PrecioDesde *float64 `json:"precio_desde"`

	Activo bool `json:"activo"`

	Orden int `json:"orden"`

	CreadoEn time.Time `json:"created_at"`

	ActualizadoEn time.Time `json:"updated_at"`
}

/*
===========================================================
SERVICIO
===========================================================
*/

type ProyectoWebService struct {
	DB *pgxpool.Pool
}

/*
===========================================================
CONSTRUCTOR
===========================================================
*/

func NewProyectoWebService(
	db *pgxpool.Pool,
) *ProyectoWebService {

	return &ProyectoWebService{
		DB: db,
	}
}

/*
===========================================================
FILTROS
===========================================================
*/

type FiltrosProyectoWeb struct {
	Buscar string

	Estado string

	Tipo string

	Ciudad string

	Etapa string

	Activo *bool

	Limit int

	Offset int
}

/*
===========================================================
VALIDAR PROYECTO
===========================================================
*/

func validarProyectoWeb(
	proyecto *ProyectoWeb,
) error {

	if proyecto == nil {
		return errors.New(
			"el proyecto es obligatorio",
		)
	}

	proyecto.Codigo =
		strings.TrimSpace(
			proyecto.Codigo,
		)

	proyecto.Titulo =
		strings.TrimSpace(
			proyecto.Titulo,
		)

	proyecto.Slug =
		strings.TrimSpace(
			proyecto.Slug,
		)

	proyecto.Tipo =
		strings.TrimSpace(
			proyecto.Tipo,
		)

	proyecto.Ciudad =
		strings.TrimSpace(
			proyecto.Ciudad,
		)

	proyecto.Direccion =
		strings.TrimSpace(
			proyecto.Direccion,
		)

	proyecto.Etapa =
		strings.TrimSpace(
			proyecto.Etapa,
		)

	proyecto.Imagen =
		strings.TrimSpace(
			proyecto.Imagen,
		)

	proyecto.Estado =
		strings.TrimSpace(
			proyecto.Estado,
		)

	/*
		=======================================================
		CAMPOS OBLIGATORIOS
		=======================================================
	*/

	if proyecto.Codigo == "" {
		return errors.New(
			"el código es obligatorio",
		)
	}

	if proyecto.Titulo == "" {
		return errors.New(
			"el título es obligatorio",
		)
	}

	if proyecto.Slug == "" {
		return errors.New(
			"el slug es obligatorio",
		)
	}

	if proyecto.Tipo == "" {
		return errors.New(
			"el tipo es obligatorio",
		)
	}

	if proyecto.Ciudad == "" {
		return errors.New(
			"la ciudad es obligatoria",
		)
	}

	if proyecto.Direccion == "" {
		return errors.New(
			"la dirección es obligatoria",
		)
	}

	if proyecto.Etapa == "" {
		return errors.New(
			"la etapa es obligatoria",
		)
	}

	if proyecto.Imagen == "" {
		return errors.New(
			"la imagen es obligatoria",
		)
	}

	/*
		=======================================================
		TIPO
		=======================================================
	*/

	switch proyecto.Tipo {

	case "Departamento":
	case "Lote":
	case "Casa":
	case "Resort":

	default:
		return errors.New(
			"tipo inválido. Usa Departamento, Lote, Casa o Resort",
		)
	}

	/*
		=======================================================
		ESTADO
		=======================================================
	*/

	switch proyecto.Estado {

	case "disponible":

	case "vendido":

		/*
			Si está vendido,
			no debe tener precio.
		*/

		proyecto.PrecioDesde = nil

	default:

		return errors.New(
			"estado inválido. Usa disponible o vendido",
		)
	}

	/*
		=======================================================
		METRAJE DESDE
		=======================================================
	*/

	if proyecto.MetrajeDesde != nil {

		if *proyecto.MetrajeDesde <= 0 {

			return errors.New(
				"metraje_desde debe ser mayor a 0",
			)
		}
	}

	/*
		=======================================================
		METRAJE HASTA
		=======================================================
	*/

	if proyecto.MetrajeHasta != nil {

		if *proyecto.MetrajeHasta <= 0 {

			return errors.New(
				"metraje_hasta debe ser mayor a 0",
			)
		}
	}

	/*
		=======================================================
		RANGO DE METRAJE
		=======================================================
	*/

	if proyecto.MetrajeDesde != nil &&
		proyecto.MetrajeHasta != nil {

		if *proyecto.MetrajeHasta <
			*proyecto.MetrajeDesde {

			return errors.New(
				"metraje_hasta no puede ser menor que metraje_desde",
			)
		}
	}

	/*
		=======================================================
		PRECIO
		=======================================================
	*/

	if proyecto.PrecioDesde != nil {

		if *proyecto.PrecioDesde < 0 {

			return errors.New(
				"precio_desde no puede ser negativo",
			)
		}
	}

	/*
		=======================================================
		SLUG
		=======================================================
	*/

	proyecto.Slug =
		strings.ToLower(
			proyecto.Slug,
		)

	proyecto.Slug =
		strings.Trim(
			proyecto.Slug,
			"/",
		)

	if proyecto.Slug == "" {
		return errors.New(
			"el slug es obligatorio",
		)
	}

	return nil
}

/*
===========================================================
CREAR
===========================================================
*/

func (s *ProyectoWebService) Crear(
	ctx context.Context,
	proyecto *ProyectoWeb,
) error {

	if err :=
		validarProyectoWeb(
			proyecto,
		); err != nil {

		return err
	}

	const query = `
		INSERT INTO proyectos_web (
			codigo,
			titulo,
			slug,
			tipo,
			ciudad,
			direccion,
			etapa,
			imagen,
			dormitorios,
			metraje_desde,
			metraje_hasta,
			estado,
			precio_desde,
			activo,
			orden
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15
		)
		RETURNING
			id,
			created_at,
			updated_at
	`

	err := s.DB.QueryRow(
		ctx,
		query,

		proyecto.Codigo,
		proyecto.Titulo,
		proyecto.Slug,
		proyecto.Tipo,
		proyecto.Ciudad,
		proyecto.Direccion,
		proyecto.Etapa,
		proyecto.Imagen,
		proyecto.Dormitorios,
		proyecto.MetrajeDesde,
		proyecto.MetrajeHasta,
		proyecto.Estado,
		proyecto.PrecioDesde,
		proyecto.Activo,
		proyecto.Orden,
	).Scan(
		&proyecto.ID,
		&proyecto.CreadoEn,
		&proyecto.ActualizadoEn,
	)

	if err != nil {

		return fmt.Errorf(
			"no se pudo crear el proyecto: %w",
			err,
		)
	}

	return nil
}

/*
===========================================================
OBTENER POR ID
===========================================================
*/

func (s *ProyectoWebService) ObtenerPorID(
	ctx context.Context,
	id string,
) (*ProyectoWeb, error) {

	const query = `
		SELECT
			id::text,
			codigo,
			titulo,
			slug,
			tipo,
			ciudad,
			direccion,
			etapa,
			imagen,
			dormitorios,
			metraje_desde,
			metraje_hasta,
			estado,
			precio_desde,
			activo,
			orden,
			created_at,
			updated_at
		FROM proyectos_web
		WHERE id = $1
	`

	var proyecto ProyectoWeb

	err := s.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&proyecto.ID,
		&proyecto.Codigo,
		&proyecto.Titulo,
		&proyecto.Slug,
		&proyecto.Tipo,
		&proyecto.Ciudad,
		&proyecto.Direccion,
		&proyecto.Etapa,
		&proyecto.Imagen,
		&proyecto.Dormitorios,
		&proyecto.MetrajeDesde,
		&proyecto.MetrajeHasta,
		&proyecto.Estado,
		&proyecto.PrecioDesde,
		&proyecto.Activo,
		&proyecto.Orden,
		&proyecto.CreadoEn,
		&proyecto.ActualizadoEn,
	)

	if err != nil {

		return nil, err
	}

	return &proyecto, nil
}

/*
===========================================================
OBTENER POR SLUG
===========================================================

Este método es el que faltaba.

Ejemplo:

GET /api/web/proyectos/slug/moro416

Busca:

WHERE slug = 'moro416'
===========================================================
*/

func (s *ProyectoWebService) ObtenerPorSlug(
	ctx context.Context,
	slug string,
) (*ProyectoWeb, error) {

	slug =
		strings.TrimSpace(
			slug,
		)

	slug =
		strings.ToLower(
			slug,
		)

	slug =
		strings.Trim(
			slug,
			"/",
		)

	if slug == "" {

		return nil,
			errors.New(
				"el slug es obligatorio",
			)
	}

	const query = `
		SELECT
			id::text,
			codigo,
			titulo,
			slug,
			tipo,
			ciudad,
			direccion,
			etapa,
			imagen,
			dormitorios,
			metraje_desde,
			metraje_hasta,
			estado,
			precio_desde,
			activo,
			orden,
			created_at,
			updated_at
		FROM proyectos_web
		WHERE slug = $1
		LIMIT 1
	`

	var proyecto ProyectoWeb

	err := s.DB.QueryRow(
		ctx,
		query,
		slug,
	).Scan(
		&proyecto.ID,
		&proyecto.Codigo,
		&proyecto.Titulo,
		&proyecto.Slug,
		&proyecto.Tipo,
		&proyecto.Ciudad,
		&proyecto.Direccion,
		&proyecto.Etapa,
		&proyecto.Imagen,
		&proyecto.Dormitorios,
		&proyecto.MetrajeDesde,
		&proyecto.MetrajeHasta,
		&proyecto.Estado,
		&proyecto.PrecioDesde,
		&proyecto.Activo,
		&proyecto.Orden,
		&proyecto.CreadoEn,
		&proyecto.ActualizadoEn,
	)

	if err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil, nil
		}

		return nil,
			fmt.Errorf(
				"no se pudo obtener el proyecto por slug: %w",
				err,
			)
	}

	return &proyecto, nil
}

/*
===========================================================
LISTAR
===========================================================
*/

func (s *ProyectoWebService) Listar(
	ctx context.Context,
	filtros FiltrosProyectoWeb,
) ([]ProyectoWeb, int, error) {

	where := []string{
		"1 = 1",
	}

	args := []interface{}{}

	param := 1

	/*
		=======================================================
		BUSCAR
		=======================================================
	*/

	if filtros.Buscar != "" {

		where = append(
			where,
			fmt.Sprintf(
				`(
					codigo ILIKE $%d
					OR titulo ILIKE $%d
					OR slug ILIKE $%d
					OR ciudad ILIKE $%d
					OR direccion ILIKE $%d
				)`,
				param,
				param,
				param,
				param,
				param,
			),
		)

		args = append(
			args,
			"%"+filtros.Buscar+"%",
		)

		param++
	}

	/*
		=======================================================
		ESTADO
		=======================================================
	*/

	if filtros.Estado != "" {

		where = append(
			where,
			fmt.Sprintf(
				"estado = $%d",
				param,
			),
		)

		args = append(
			args,
			filtros.Estado,
		)

		param++
	}

	/*
		=======================================================
		TIPO
		=======================================================
	*/

	if filtros.Tipo != "" {

		where = append(
			where,
			fmt.Sprintf(
				"tipo = $%d",
				param,
			),
		)

		args = append(
			args,
			filtros.Tipo,
		)

		param++
	}

	/*
		=======================================================
		CIUDAD
		=======================================================
	*/

	if filtros.Ciudad != "" {

		where = append(
			where,
			fmt.Sprintf(
				"ciudad ILIKE $%d",
				param,
			),
		)

		args = append(
			args,
			"%"+filtros.Ciudad+"%",
		)

		param++
	}

	/*
		=======================================================
		ETAPA
		=======================================================
	*/

	if filtros.Etapa != "" {

		where = append(
			where,
			fmt.Sprintf(
				"etapa ILIKE $%d",
				param,
			),
		)

		args = append(
			args,
			"%"+filtros.Etapa+"%",
		)

		param++
	}

	/*
		=======================================================
		ACTIVO
		=======================================================
	*/

	if filtros.Activo != nil {

		where = append(
			where,
			fmt.Sprintf(
				"activo = $%d",
				param,
			),
		)

		args = append(
			args,
			*filtros.Activo,
		)

		param++
	}

	whereSQL :=
		strings.Join(
			where,
			" AND ",
		)

	/*
		=======================================================
		CONTAR
		=======================================================
	*/

	countQuery :=
		"SELECT COUNT(*) FROM proyectos_web WHERE " +
			whereSQL

	var total int

	err := s.DB.QueryRow(
		ctx,
		countQuery,
		args...,
	).Scan(
		&total,
	)

	if err != nil {

		return nil,
			0,
			fmt.Errorf(
				"no se pudo contar proyectos: %w",
				err,
			)
	}

	/*
		=======================================================
		LIMIT
		=======================================================
	*/

	limit := filtros.Limit

	if limit <= 0 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	offset := filtros.Offset

	if offset < 0 {
		offset = 0
	}

	/*
		=======================================================
		LISTADO
		=======================================================
	*/

	query :=
		`
		SELECT
			id::text,
			codigo,
			titulo,
			slug,
			tipo,
			ciudad,
			direccion,
			etapa,
			imagen,
			dormitorios,
			metraje_desde,
			metraje_hasta,
			estado,
			precio_desde,
			activo,
			orden,
			created_at,
			updated_at
		FROM proyectos_web
		WHERE ` +
			whereSQL +
			`
		ORDER BY
			orden ASC,
			created_at DESC
		LIMIT $` +
			fmt.Sprint(param) +
			`
		OFFSET $` +
			fmt.Sprint(param+1)

	args = append(
		args,
		limit,
		offset,
	)

	rows, err :=
		s.DB.Query(
			ctx,
			query,
			args...,
		)

	if err != nil {

		return nil,
			0,
			fmt.Errorf(
				"no se pudieron listar proyectos: %w",
				err,
			)
	}

	defer rows.Close()

	proyectos :=
		make(
			[]ProyectoWeb,
			0,
			limit,
		)

	for rows.Next() {

		var proyecto ProyectoWeb

		err := rows.Scan(
			&proyecto.ID,
			&proyecto.Codigo,
			&proyecto.Titulo,
			&proyecto.Slug,
			&proyecto.Tipo,
			&proyecto.Ciudad,
			&proyecto.Direccion,
			&proyecto.Etapa,
			&proyecto.Imagen,
			&proyecto.Dormitorios,
			&proyecto.MetrajeDesde,
			&proyecto.MetrajeHasta,
			&proyecto.Estado,
			&proyecto.PrecioDesde,
			&proyecto.Activo,
			&proyecto.Orden,
			&proyecto.CreadoEn,
			&proyecto.ActualizadoEn,
		)

		if err != nil {

			return nil,
				0,
				fmt.Errorf(
					"no se pudo leer proyecto: %w",
					err,
				)
		}

		proyectos =
			append(
				proyectos,
				proyecto,
			)
	}

	if err := rows.Err(); err != nil {

		return nil,
			0,
			fmt.Errorf(
				"error recorriendo proyectos: %w",
				err,
			)
	}

	return proyectos, total, nil
}

/*
===========================================================
ACTUALIZAR
===========================================================
*/

func (s *ProyectoWebService) Actualizar(
	ctx context.Context,
	proyecto *ProyectoWeb,
) error {

	if err :=
		validarProyectoWeb(
			proyecto,
		); err != nil {

		return err
	}

	const query = `
		UPDATE proyectos_web
		SET
			codigo = $1,
			titulo = $2,
			slug = $3,
			tipo = $4,
			ciudad = $5,
			direccion = $6,
			etapa = $7,
			imagen = $8,
			dormitorios = $9,
			metraje_desde = $10,
			metraje_hasta = $11,
			estado = $12,
			precio_desde = $13,
			activo = $14,
			orden = $15,
			updated_at = NOW()
		WHERE id = $16
		RETURNING updated_at
	`

	err := s.DB.QueryRow(
		ctx,
		query,

		proyecto.Codigo,
		proyecto.Titulo,
		proyecto.Slug,
		proyecto.Tipo,
		proyecto.Ciudad,
		proyecto.Direccion,
		proyecto.Etapa,
		proyecto.Imagen,
		proyecto.Dormitorios,
		proyecto.MetrajeDesde,
		proyecto.MetrajeHasta,
		proyecto.Estado,
		proyecto.PrecioDesde,
		proyecto.Activo,
		proyecto.Orden,
		proyecto.ID,
	).Scan(
		&proyecto.ActualizadoEn,
	)

	if err != nil {

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return errors.New(
				"proyecto no encontrado",
			)
		}

		return fmt.Errorf(
			"no se pudo actualizar el proyecto: %w",
			err,
		)
	}

	return nil
}

/*
===========================================================
ELIMINAR
===========================================================
*/

func (s *ProyectoWebService) Eliminar(
	ctx context.Context,
	id string,
) error {

	const query = `
		DELETE FROM proyectos_web
		WHERE id = $1
	`

	result, err :=
		s.DB.Exec(
			ctx,
			query,
			id,
		)

	if err != nil {

		return fmt.Errorf(
			"no se pudo eliminar el proyecto: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {

		return errors.New(
			"proyecto no encontrado",
		)
	}

	return nil
}
