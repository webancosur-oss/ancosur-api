package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ancosur-api/email"
	"ancosur-api/services"
)

// ============================================================
// REQUEST: FORMULARIO WEB
// ============================================================

type ReclamoWebRequest struct {
	Establecimiento string `json:"establecimiento"`
	Tipo            string `json:"tipo" binding:"required"`
	Nombres         string `json:"nombres" binding:"required"`
	Apellidos       string `json:"apellidos" binding:"required"`
	Email           string `json:"email" binding:"required"`
	Telefono        string `json:"telefono"`

	TipoDocumento   string `json:"tipo_documento" binding:"required"`
	NumeroDocumento string `json:"numero_documento" binding:"required"`

	Departamento string `json:"departamento"`
	Provincia    string `json:"provincia"`
	Distrito     string `json:"distrito"`
	Domicilio    string `json:"domicilio"`

	EsMenor bool `json:"es_menor"`

	TutorNombres         string `json:"tutor_nombres"`
	TutorApellidos       string `json:"tutor_apellidos"`
	TutorTipoDocumento   string `json:"tutor_tipo_documento"`
	TutorNumeroDocumento string `json:"tutor_numero_documento"`
	TutorTelefono        string `json:"tutor_telefono"`
	TutorEmail           string `json:"tutor_email"`

	TipoBien                    string      `json:"tipo_bien"`
	Proyecto                    string      `json:"proyecto" binding:"required"`
	TipoProyecto                string      `json:"tipo_proyecto"`
	EstadoProyecto              string      `json:"estado_proyecto"`
	CiudadProyecto              string      `json:"ciudad_proyecto"`
	DireccionProyecto           string      `json:"direccion_proyecto"`
	EdificioTorreBloque         string      `json:"edificio_torre_bloque"`
	UnidadInmobiliaria          string      `json:"unidad_inmobiliaria"`
	NumeroOperacion             string      `json:"numero_operacion"`
	DescripcionProductoServicio string      `json:"descripcion_producto_servicio"`
	MontoReclamado              interface{} `json:"monto_reclamado"`

	Detalle string `json:"detalle" binding:"required"`
	Pedido  string `json:"pedido" binding:"required"`

	AutorizaNotificacionEmail bool   `json:"autoriza_notificacion_email"`
	CorreoNotificacion        string `json:"correo_notificacion"`
	Conformidad               *bool  `json:"conformidad" binding:"required"`
	Canal                     string `json:"canal"`
	FechaFrontend             string `json:"fecha_frontend"`
}

// ============================================================
// REQUEST: ACTUALIZACION ADMINISTRATIVA
// ============================================================

type ActualizarReclamoRequest struct {
	Estado            string `json:"estado"`
	Prioridad         string `json:"prioridad"`
	ResponsableID     *int64 `json:"responsable_id"`
	ResponsableNombre string `json:"responsable_nombre"`
	Respuesta         string `json:"respuesta"`
	MedioRespuesta    string `json:"medio_respuesta"`
	Comentario        string `json:"comentario"`
}

// ============================================================
// MODELO DASHBOARD
// ============================================================

type ReclamoWeb struct {
	ID int64 `json:"id"`

	NumeroCorrelativo int64  `json:"numero_correlativo"`
	CodigoRegistro    string `json:"codigo_registro"`
	Anio              int    `json:"anio"`

	Tipo            string `json:"tipo"`
	Establecimiento string `json:"establecimiento"`

	Nombres   string `json:"nombres"`
	Apellidos string `json:"apellidos"`

	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`
	Domicilio       string `json:"domicilio"`
	Telefono        string `json:"telefono"`
	Email           string `json:"email"`

	Departamento string `json:"departamento"`
	Provincia    string `json:"provincia"`
	Distrito     string `json:"distrito"`

	EsMenor bool `json:"es_menor"`

	RepresentanteNombres         string `json:"representante_nombres"`
	RepresentanteApellidos       string `json:"representante_apellidos"`
	RepresentanteTipoDocumento   string `json:"representante_tipo_documento"`
	RepresentanteNumeroDocumento string `json:"representante_numero_documento"`
	RepresentanteTelefono        string `json:"representante_telefono"`
	RepresentanteEmail           string `json:"representante_email"`

	TipoContratado string `json:"tipo_contratado"`
	Proyecto       string `json:"proyecto"`
	EstadoProyecto string `json:"estado_proyecto"`

	Edificio        string `json:"edificio"`
	Unidad          string `json:"unidad"`
	NumeroOperacion string `json:"numero_operacion"`

	ProductoServicio string   `json:"producto_servicio"`
	MontoReclamado   *float64 `json:"monto_reclamado,omitempty"`

	Detalle        string `json:"detalle"`
	PedidoConcreto string `json:"pedido_concreto"`

	Estado    string `json:"estado"`
	Prioridad string `json:"prioridad"`

	ResponsableID     *int64 `json:"responsable_id"`
	ResponsableNombre string `json:"responsable_nombre"`

	Respuesta      string     `json:"respuesta"`
	FechaRespuesta *time.Time `json:"fecha_respuesta"`
	MedioRespuesta string     `json:"medio_respuesta"`

	CorreoConstanciaEstado      string     `json:"correo_constancia_estado"`
	CorreoConstanciaEnviadoAt   *time.Time `json:"correo_constancia_enviado_at"`
	CorreoConstanciaEntregadoAt *time.Time `json:"correo_constancia_entregado_at"`
	CorreoConstanciaMessageID   string     `json:"correo_constancia_message_id"`
	CorreoConstanciaError       string     `json:"correo_constancia_error"`

	CorreoInternoEstado      string     `json:"correo_interno_estado"`
	CorreoInternoEnviadoAt   *time.Time `json:"correo_interno_enviado_at"`
	CorreoInternoEntregadoAt *time.Time `json:"correo_interno_entregado_at"`
	CorreoInternoMessageID   string     `json:"correo_interno_message_id"`
	CorreoInternoError       string     `json:"correo_interno_error"`

	AutorizaNotificacionEmail bool `json:"autoriza_notificacion_email"`

	ConstanciaURL        string     `json:"constancia_url"`
	ConstanciaGeneradaAt *time.Time `json:"constancia_generada_at"`

	Canal     string `json:"canal"`
	IPOrigen  string `json:"ip_origen"`
	UserAgent string `json:"user_agent"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Se entrega solo como JSON, para el dashboard y el documento.
	DatosOriginales json.RawMessage `json:"datos_originales,omitempty"`
}

type HistorialItem struct {
	ID             int64     `json:"id"`
	ReclamacionID  int64     `json:"reclamacion_id"`
	Accion         string    `json:"accion"`
	EstadoAnterior string    `json:"estado_anterior"`
	EstadoNuevo    string    `json:"estado_nuevo"`
	Comentario     string    `json:"comentario"`
	UsuarioID      *int64    `json:"usuario_id"`
	UsuarioNombre  string    `json:"usuario_nombre"`
	CreatedAt      time.Time `json:"created_at"`
}

type EmailLogItem struct {
	ID            int64      `json:"id"`
	ReclamacionID int64      `json:"reclamacion_id"`
	Destinatario  string     `json:"destinatario"`
	Tipo          string     `json:"tipo"`
	Proveedor     string     `json:"proveedor"`
	MessageID     string     `json:"message_id"`
	Estado        string     `json:"estado"`
	EnviadoAt     *time.Time `json:"enviado_at"`
	EntregadoAt   *time.Time `json:"entregado_at"`
	Error         string     `json:"error"`
	Metadata      any        `json:"metadata"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ReclamoEstadisticas struct {
	Total       int64           `json:"total"`
	Reclamos    int64           `json:"reclamos"`
	Quejas      int64           `json:"quejas"`
	Pendientes  int64           `json:"pendientes"`
	EnRevision  int64           `json:"en_revision"`
	EnProceso   int64           `json:"en_proceso"`
	Atendidos   int64           `json:"atendidos"`
	Cerrados    int64           `json:"cerrados"`
	Urgentes    int64           `json:"urgentes"`
	Alta        int64           `json:"alta"`
	PorProyecto []ReporteConteo `json:"por_proyecto"`
	PorMes      []ReporteConteo `json:"por_mes"`
}

type ReporteConteo struct {
	Etiqueta string `json:"etiqueta"`
	Total    int64  `json:"total"`
}

// ============================================================
// RUTAS
// ============================================================

func RutasReclamos(api *gin.RouterGroup, db *pgxpool.Pool) {
	api.POST("/reclamos", crearReclamo(db))
	api.GET("/reclamos", listarReclamos(db))
	api.GET("/reclamos/estadisticas", estadisticasReclamos(db))
	api.GET("/reclamos/reportes", reportesReclamos(db))
	api.GET("/reclamos/:id", obtenerReclamo(db))
	api.GET("/reclamos/:id/historial", obtenerHistorial(db))
	api.GET("/reclamos/:id/emails", obtenerEmails(db))
	api.GET("/reclamos/:id/pdf", generarPDFReclamo(db))
	api.PUT("/reclamos/:id", actualizarReclamo(db))
}

// ============================================================
// POST - CREAR
// ============================================================

func leerAdjuntos(c *gin.Context) ([]email.Attachment, error) {
	attachments := make([]email.Attachment, 0)
	if c.Request.MultipartForm == nil {
		return attachments, nil
	}
	files := c.Request.MultipartForm.File["adjuntos"]
	for _, header := range files {
		if header == nil {
			continue
		}
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("no se pudo abrir el archivo %s: %w", header.Filename, err)
		}
		content, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return nil, fmt.Errorf("no se pudo leer el archivo %s: %w", header.Filename, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("no se pudo cerrar el archivo %s: %w", header.Filename, closeErr)
		}
		contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
		if contentType == "" {
			if dot := strings.LastIndex(header.Filename, "."); dot >= 0 && dot+1 < len(header.Filename) {
				contentType = mime.TypeByExtension(strings.ToLower(header.Filename[dot:]))
			}
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		attachments = append(attachments, email.Attachment{Filename: header.Filename, Content: content, ContentType: contentType})
	}
	return attachments, nil
}

func crearReclamo(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request ReclamoWebRequest

		contentType := c.GetHeader("Content-Type")
		if strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
			if err := c.Request.ParseMultipartForm(25 << 20); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "No se pudo procesar el formulario multipart.", "error": err.Error()})
				return
			}

			dataJSON := strings.TrimSpace(c.PostForm("data"))
			if dataJSON == "" {
				// Compatibilidad: algunos formularios pueden enviar JSON directamente bajo 'json'.
				dataJSON = strings.TrimSpace(c.PostForm("json"))
			}
			if dataJSON == "" {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Falta el campo data del formulario."})
				return
			}
			if err := json.Unmarshal([]byte(dataJSON), &request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "El contenido del formulario no es JSON válido.", "error": err.Error()})
				return
			}
		} else {
			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Los datos enviados no son válidos.", "error": err.Error()})
				return
			}
		}

		normalizarReclamo(&request)
		if err := validarReclamo(request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
			return
		}

		attachments, err := leerAdjuntos(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "No se pudieron procesar los archivos adjuntos.",
				"error":   err.Error(),
			})
			return
		}

		ipOrigen := obtenerIPCliente(c)
		userAgent := strings.TrimSpace(c.GetHeader("User-Agent"))

		// Se conserva el contenido exacto enviado para poder reproducir la constancia PDF.
		datosOriginales, err := json.Marshal(request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo preparar la información del reclamo.", "error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		tx, err := db.Begin(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo iniciar el registro.", "error": err.Error()})
			return
		}
		defer tx.Rollback(ctx)

		var (
			id        int64
			numero    int64
			codigo    string
			anio      int
			createdAt time.Time
		)

		var monto any = nil
		if v, ok := montoNumerico(request.MontoReclamado); ok {
			monto = v
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO libro_reclamaciones (
				establecimiento,
				tipo,
				nombres,
				apellidos,
				tipo_documento,
				numero_documento,
				domicilio,
				telefono,
				email,
				departamento,
				provincia,
				distrito,
				es_menor,
				representante_nombres,
				representante_apellidos,
				representante_tipo_documento,
				representante_numero_documento,
				representante_telefono,
				representante_email,
				tipo_contratado,
				proyecto,
				estado_proyecto,
				edificio,
				unidad,
				numero_operacion,
				producto_servicio,
				monto_reclamado,
				detalle,
				pedido_concreto,
				estado,
				prioridad,
				autoriza_notificacion_email,
				canal,
				ip_origen,
				user_agent,
				datos_originales
			)
			VALUES (
				$1,$2,$3,$4,$5,$6,
				NULLIF($7,''),NULLIF($8,''),$9,
				NULLIF($10,''),NULLIF($11,''),NULLIF($12,''),
				$13,
				NULLIF($14,''),NULLIF($15,''),NULLIF($16,''),NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),
				NULLIF($20,''),
				$21,NULLIF($22,''),NULLIF($23,''),NULLIF($24,''),NULLIF($25,''),NULLIF($26,''),
				$27,$28,$29,
				'pendiente','normal',$30,
				'web',NULLIF($31,'')::inet,NULLIF($32,''),$33::jsonb
			)
			RETURNING id, numero_correlativo, codigo_registro, anio, created_at
		`,
			request.Establecimiento,
			request.Tipo,
			request.Nombres,
			request.Apellidos,
			request.TipoDocumento,
			request.NumeroDocumento,
			request.Domicilio,
			request.Telefono,
			request.Email,
			request.Departamento,
			request.Provincia,
			request.Distrito,
			request.EsMenor,
			request.TutorNombres,
			request.TutorApellidos,
			request.TutorTipoDocumento,
			request.TutorNumeroDocumento,
			request.TutorTelefono,
			request.TutorEmail,
			request.TipoBien,
			request.Proyecto,
			request.EstadoProyecto,
			request.EdificioTorreBloque,
			request.UnidadInmobiliaria,
			request.NumeroOperacion,
			request.DescripcionProductoServicio,
			monto,
			request.Detalle,
			request.Pedido,
			request.AutorizaNotificacionEmail,
			ipOrigen,
			userAgent,
			string(datosOriginales),
		).Scan(&id, &numero, &codigo, &anio, &createdAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo guardar el reclamo en la base de datos.", "error": err.Error()})
			return
		}

		if _, err = tx.Exec(ctx, `
			INSERT INTO libro_reclamaciones_historial (
				reclamacion_id, accion, estado_anterior, estado_nuevo, comentario
			)
			VALUES ($1, 'registro', NULL, 'pendiente', 'Registro inicial desde el Libro de Reclamaciones web.')
		`, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo registrar la bitácora.", "error": err.Error()})
			return
		}

		// El correo al consumidor puede ser el email principal o el correo
		// alternativo indicado expresamente para recibir la notificación.
		consumerRecipient := request.Email
		if request.AutorizaNotificacionEmail && request.CorreoNotificacion != "" {
			consumerRecipient = request.CorreoNotificacion
		}

		// El servicio resuelve el destinatario interno desde el .env.
		emailService := services.NewEmailService()
		internalRecipient := emailService.InternalRecipient()

		if request.AutorizaNotificacionEmail {
			if _, err = tx.Exec(ctx, `
				INSERT INTO libro_reclamaciones_emails (
					reclamacion_id, destinatario, tipo, proveedor, estado, metadata
				)
				VALUES ($1,$2,'consumidor','smtp','pendiente',$3::jsonb)
			`, id, consumerRecipient, `{"evento":"constancia_registro"}`); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "No se pudo registrar la notificación al consumidor.",
					"error":   err.Error(),
				})
				return
			}
		}

		if internalRecipient != "" {
			if _, err = tx.Exec(ctx, `
				INSERT INTO libro_reclamaciones_emails (
					reclamacion_id, destinatario, tipo, proveedor, estado, metadata
				)
				VALUES ($1,$2,'interno','smtp','pendiente',$3::jsonb)
			`, id, internalRecipient, `{"evento":"registro_interno"}`); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "No se pudo registrar la notificación interna.",
					"error":   err.Error(),
				})
				return
			}
		}

		constanciaURL := fmt.Sprintf("/api/reclamos/%d/pdf", id)
		if _, err = tx.Exec(ctx, `
			UPDATE libro_reclamaciones
			SET constancia_url = $2, constancia_generada_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, id, constanciaURL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "No se pudo preparar la constancia del registro.",
				"error":   err.Error(),
			})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "No se pudo confirmar el registro.",
				"error":   err.Error(),
			})
			return
		}

		// La base de datos ya confirmó el reclamo. Desde este punto el envío de
		// correos NO debe bloquear la respuesta HTTP: SMTP puede tardar varios
		// segundos y no debe hacer esperar al usuario.
		mailData := services.ReclamoMailDataFromRequest(services.ReclamoMailInput{
			Fecha: request.FechaFrontend,
			Tipo:  request.Tipo, Nombres: request.Nombres, Apellidos: request.Apellidos,
			Email: request.Email, Telefono: request.Telefono,
			TipoDocumento: request.TipoDocumento, NumeroDocumento: request.NumeroDocumento,
			Departamento: request.Departamento, Provincia: request.Provincia, Distrito: request.Distrito, Domicilio: request.Domicilio,
			EsMenor: request.EsMenor, TutorNombres: request.TutorNombres, TutorApellidos: request.TutorApellidos,
			TutorTipoDocumento: request.TutorTipoDocumento, TutorNumeroDocumento: request.TutorNumeroDocumento,
			Establecimiento: request.Establecimiento, Proyecto: request.Proyecto, TipoProyecto: request.TipoProyecto,
			EstadoProyecto: request.EstadoProyecto, CiudadProyecto: request.CiudadProyecto, DireccionProyecto: request.DireccionProyecto,
			TipoBien: request.TipoBien, Edificio: request.EdificioTorreBloque, Unidad: request.UnidadInmobiliaria,
			NumeroOperacion: request.NumeroOperacion, Descripcion: request.DescripcionProductoServicio,
			Detalle: request.Detalle, Pedido: request.Pedido, Canal: request.Canal,
			CodigoRegistro: codigo, NumeroCorrelativo: numero, Anio: anio,
		})

		// Copias independientes para que el proceso en segundo plano no dependa
		// del Context de la petición HTTP, que termina al responder al cliente.
		attachmentsForEmail := append([]email.Attachment(nil), attachments...)

		// Respondemos inmediatamente. El correo continúa en background.
		response := gin.H{
			"success": true,
			"message": "Reclamación registrada correctamente.",
			"ticket":  codigo, "codigo_registro": codigo, "code": codigo,
			"attachments_count": len(attachments),
			"email": gin.H{
				"consumidor": gin.H{
					"destinatario": consumerRecipient,
					"estado":       "pendiente",
				},
				"interno": gin.H{
					"destinatario": internalRecipient,
					"estado":       "pendiente",
				},
			},
			"data": gin.H{
				"id": id, "numero_correlativo": numero, "codigo_registro": codigo, "anio": anio,
				"created_at": createdAt, "estado": "pendiente",
				"correo_notificacion": request.AutorizaNotificacionEmail,
				"constancia_url":      constanciaURL,
			},
		}

		go procesarEmailsReclamoAsync(
			db,
			id,
			mailData,
			consumerRecipient,
			internalRecipient,
			attachmentsForEmail,
		)

		c.JSON(http.StatusCreated, response)
	}
}

// procesarEmailsReclamoAsync ejecuta los dos envíos fuera del ciclo de vida
// de la petición HTTP. Así el registro responde rápido y SMTP no bloquea al usuario.
func procesarEmailsReclamoAsync(
	db *pgxpool.Pool,
	reclamacionID int64,
	mailData email.ReclamoTemplateData,
	consumerRecipient string,
	internalRecipient string,
	attachments []email.Attachment,
) {
	service := services.NewEmailService()

	// El servicio envía consumidor e interno en paralelo.
	consumerResult, internalResult := service.SendReclamoEmails(
		mailData,
		consumerRecipient,
		attachments,
	)

	// El proceso de fondo no utiliza el contexto HTTP. Cada actualización
	// tiene su propio timeout para evitar dejar conexiones de BD abiertas.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	now := time.Now()

	updateEmailLog := func(emailType, recipient string, result services.EmailSendResult) {
		if strings.TrimSpace(recipient) == "" {
			return
		}

		_, err := db.Exec(ctx, `
			UPDATE libro_reclamaciones_emails
			SET proveedor='smtp',
			    message_id=NULLIF($3::text,''),
			    estado=$4::varchar,
			    enviado_at=CASE WHEN $4::varchar='enviado' THEN $5::timestamptz ELSE enviado_at END,
			    error=NULLIF($6::text,''),
			    metadata=COALESCE(metadata,'{}'::jsonb) ||
			             jsonb_build_object('message_id',NULLIF($3::text,''),'resultado',$4::varchar)
			WHERE reclamacion_id=$1
			  AND tipo=$2
			  AND destinatario=$7
			  AND estado='pendiente'
		`, reclamacionID, emailType, result.MessageID, result.Status, now, result.Error, recipient)

		if err != nil {
			fmt.Printf("[RECLAMOS] Error actualizando log de email %s: %v\n", emailType, err)
		}
	}

	updateEmailLog("consumidor", consumerRecipient, consumerResult)
	updateEmailLog("interno", internalRecipient, internalResult)

	_, err := db.Exec(ctx, `
		UPDATE libro_reclamaciones
		SET correo_constancia_estado=$2::varchar,
		    correo_constancia_enviado_at=CASE WHEN $2::varchar='enviado' THEN $3::timestamptz ELSE correo_constancia_enviado_at END,
		    correo_constancia_message_id=NULLIF($4::text,''),
		    correo_constancia_error=NULLIF($5::text,''),
		    correo_interno_estado=$6::varchar,
		    correo_interno_enviado_at=CASE WHEN $6::varchar='enviado' THEN $3::timestamptz ELSE correo_interno_enviado_at END,
		    correo_interno_message_id=NULLIF($7::text,''),
		    correo_interno_error=NULLIF($8::text,''),
		    updated_at=NOW()
		WHERE id=$1
	`, reclamacionID,
		consumerResult.Status, now, consumerResult.MessageID, consumerResult.Error,
		internalResult.Status, internalResult.MessageID, internalResult.Error,
	)
	if err != nil {
		fmt.Printf("[RECLAMOS] Error actualizando estados de correo: %v\n", err)
	}
}

// ============================================================
// GET - LISTAR / DASHBOARD
// ============================================================

func listarReclamos(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()

		page := parsePositiveInt(c.Query("page"), 1)
		limit := parsePositiveInt(c.Query("limit"), 50)
		if limit > 200 {
			limit = 200
		}
		offset := (page - 1) * limit

		where, args := buildReclamoFilters(c)

		var total int64
		countSQL := `SELECT COUNT(*) FROM libro_reclamaciones r ` + where
		if err := db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo contar las reclamaciones.", "error": err.Error()})
			return
		}

		argsData := append([]any{}, args...)
		limitPos := len(argsData) + 1
		offsetPos := len(argsData) + 2
		argsData = append(argsData, limit, offset)

		query := `
			SELECT
				r.id, r.numero_correlativo, r.codigo_registro, r.anio, r.tipo,
				COALESCE(r.establecimiento,''), r.nombres, r.apellidos,
				r.tipo_documento, r.numero_documento,
				COALESCE(r.domicilio,''), COALESCE(r.telefono,''), r.email,
				COALESCE(r.departamento,''), COALESCE(r.provincia,''), COALESCE(r.distrito,''),
				COALESCE(r.es_menor,false),
				COALESCE(r.representante_nombres,''), COALESCE(r.representante_apellidos,''),
				COALESCE(r.representante_tipo_documento,''), COALESCE(r.representante_numero_documento,''),
				COALESCE(r.representante_telefono,''), COALESCE(r.representante_email,''),
				COALESCE(r.tipo_contratado,''), r.proyecto, COALESCE(r.estado_proyecto,''),
				COALESCE(r.edificio,''), COALESCE(r.unidad,''), COALESCE(r.numero_operacion,''),
				COALESCE(r.producto_servicio,''), r.monto_reclamado, r.detalle, r.pedido_concreto,
				r.estado, r.prioridad, r.responsable_id, COALESCE(r.responsable_nombre,''),
				COALESCE(r.respuesta,''), r.fecha_respuesta, COALESCE(r.medio_respuesta,''),
				COALESCE(r.correo_constancia_estado,'pendiente'), r.correo_constancia_enviado_at,
				r.correo_constancia_entregado_at, COALESCE(r.correo_constancia_message_id,''),
				COALESCE(r.correo_constancia_error,''),
				COALESCE(r.correo_interno_estado,'pendiente'), r.correo_interno_enviado_at,
				r.correo_interno_entregado_at, COALESCE(r.correo_interno_message_id,''),
				COALESCE(r.correo_interno_error,''), COALESCE(r.autoriza_notificacion_email,false),
				COALESCE(r.constancia_url,''), r.constancia_generada_at,
				COALESCE(r.canal,''), COALESCE(r.ip_origen::text,''), COALESCE(r.user_agent,''),
				r.created_at, r.updated_at
			FROM libro_reclamaciones r ` + where + fmt.Sprintf(`
			ORDER BY r.created_at DESC
			LIMIT $%d OFFSET $%d`, limitPos, offsetPos)

		rows, err := db.Query(ctx, query, argsData...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudieron cargar las reclamaciones.", "error": err.Error()})
			return
		}
		defer rows.Close()

		items := make([]ReclamoWeb, 0)
		for rows.Next() {
			var r ReclamoWeb
			if err := scanReclamo(rows, &r, false); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo leer una reclamación.", "error": err.Error()})
				return
			}
			items = append(items, r)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudieron recorrer las reclamaciones.", "error": err.Error()})
			return
		}

		pages := int64(0)
		if total > 0 {
			pages = (total + int64(limit) - 1) / int64(limit)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"total":   total,
			"page":    page,
			"limit":   limit,
			"pages":   pages,
			"data":    items,
		})
	}
}

// ============================================================
// GET - DETALLE
// ============================================================

func obtenerReclamo(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		var r ReclamoWeb
		err := db.QueryRow(ctx, reclamoSelectByID(), id).Scan(reclamoScanDestinations(&r, true)...)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "La reclamación no existe."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo cargar la reclamación.", "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": r})
	}
}

// ============================================================
// GET - HISTORIAL
// ============================================================

func obtenerHistorial(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, reclamacion_id, accion,
			       COALESCE(estado_anterior,''), COALESCE(estado_nuevo,''),
			       COALESCE(comentario,''), usuario_id,
			       COALESCE(usuario_nombre,''), created_at
			FROM libro_reclamaciones_historial
			WHERE reclamacion_id = $1
			ORDER BY created_at DESC, id DESC
		`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo cargar el historial.", "error": err.Error()})
			return
		}
		defer rows.Close()

		items := make([]HistorialItem, 0)
		for rows.Next() {
			var h HistorialItem
			if err := rows.Scan(&h.ID, &h.ReclamacionID, &h.Accion, &h.EstadoAnterior, &h.EstadoNuevo, &h.Comentario, &h.UsuarioID, &h.UsuarioNombre, &h.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo leer el historial.", "error": err.Error()})
				return
			}
			items = append(items, h)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo recorrer el historial.", "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "total": len(items), "data": items})
	}
}

// ============================================================
// GET - EMAILS
// ============================================================

func obtenerEmails(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, reclamacion_id, destinatario, tipo,
			       COALESCE(proveedor,''), COALESCE(message_id,''), estado,
			       enviado_at, entregado_at, COALESCE(error,''), metadata, created_at
			FROM libro_reclamaciones_emails
			WHERE reclamacion_id = $1
			ORDER BY created_at DESC, id DESC
		`, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudieron cargar los registros de correo.", "error": err.Error()})
			return
		}
		defer rows.Close()

		items := make([]EmailLogItem, 0)
		for rows.Next() {
			var e EmailLogItem
			if err := rows.Scan(&e.ID, &e.ReclamacionID, &e.Destinatario, &e.Tipo, &e.Proveedor, &e.MessageID, &e.Estado, &e.EnviadoAt, &e.EntregadoAt, &e.Error, &e.Metadata, &e.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo leer un registro de correo.", "error": err.Error()})
				return
			}
			items = append(items, e)
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "total": len(items), "data": items})
	}
}

// ============================================================
// PUT - ACTUALIZAR
// ============================================================

func actualizarReclamo(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		var request ActualizarReclamoRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Los datos enviados no son válidos.", "error": err.Error()})
			return
		}

		request.Estado = strings.ToLower(strings.TrimSpace(request.Estado))
		request.Prioridad = strings.ToLower(strings.TrimSpace(request.Prioridad))
		request.ResponsableNombre = strings.TrimSpace(request.ResponsableNombre)
		request.Respuesta = strings.TrimSpace(request.Respuesta)
		request.MedioRespuesta = strings.TrimSpace(request.MedioRespuesta)
		request.Comentario = strings.TrimSpace(request.Comentario)

		estados := map[string]bool{
			"pendiente": true, "en_revision": true, "en_proceso": true, "atendido": true, "cerrado": true,
		}
		prioridades := map[string]bool{
			"baja": true, "normal": true, "alta": true, "urgente": true,
		}
		if request.Estado != "" && !estados[request.Estado] {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "El estado enviado no es válido."})
			return
		}
		if request.Prioridad != "" && !prioridades[request.Prioridad] {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "La prioridad enviada no es válida."})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		var estadoAnterior string
		if err := db.QueryRow(ctx, `SELECT estado FROM libro_reclamaciones WHERE id = $1`, id).Scan(&estadoAnterior); err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "La reclamación no existe."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo consultar la reclamación.", "error": err.Error()})
			return
		}

		var fechaRespuesta any
		if request.Respuesta != "" {
			fechaRespuesta = time.Now()
		}

		var nuevoEstado string
		err := db.QueryRow(ctx, `
			UPDATE libro_reclamaciones
			SET estado = COALESCE(NULLIF($2,''),estado),
			    prioridad = COALESCE(NULLIF($3,''),prioridad),
			    responsable_id = COALESCE($4,responsable_id),
			    responsable_nombre = COALESCE(NULLIF($5,''),responsable_nombre),
			    respuesta = COALESCE(NULLIF($6,''),respuesta),
			    fecha_respuesta = COALESCE($7,fecha_respuesta),
			    medio_respuesta = COALESCE(NULLIF($8,''),medio_respuesta),
			    updated_at = NOW()
			WHERE id = $1
			RETURNING estado
		`, id, request.Estado, request.Prioridad, request.ResponsableID, request.ResponsableNombre, request.Respuesta, fechaRespuesta, request.MedioRespuesta).Scan(&nuevoEstado)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo actualizar la reclamación.", "error": err.Error()})
			return
		}

		accion := "actualizacion"
		if request.Respuesta != "" {
			accion = "respuesta"
		}
		if estadoAnterior != nuevoEstado {
			accion = "cambio_estado"
		}

		_, err = db.Exec(ctx, `
			INSERT INTO libro_reclamaciones_historial (
				reclamacion_id, accion, estado_anterior, estado_nuevo,
				comentario, usuario_nombre
			)
			VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''))
		`, id, accion, estadoAnterior, nuevoEstado, request.Comentario, request.ResponsableNombre)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "La reclamación fue actualizada, pero no se pudo registrar el historial.",
				"data":    gin.H{"id": id, "estado": nuevoEstado},
				"warning": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Reclamación actualizada correctamente.",
			"data":    gin.H{"id": id, "estado": nuevoEstado},
		})
	}
}

// ============================================================
// GET - ESTADISTICAS
// ============================================================

func estadisticasReclamos(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		var s ReclamoEstadisticas
		err := db.QueryRow(ctx, `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE tipo='reclamo') AS reclamos,
				COUNT(*) FILTER (WHERE tipo='queja') AS quejas,
				COUNT(*) FILTER (WHERE estado='pendiente') AS pendientes,
				COUNT(*) FILTER (WHERE estado='en_revision') AS en_revision,
				COUNT(*) FILTER (WHERE estado='en_proceso') AS en_proceso,
				COUNT(*) FILTER (WHERE estado='atendido') AS atendidos,
				COUNT(*) FILTER (WHERE estado='cerrado') AS cerrados,
				COUNT(*) FILTER (WHERE prioridad='urgente') AS urgentes,
				COUNT(*) FILTER (WHERE prioridad='alta') AS alta
			FROM libro_reclamaciones
		`).Scan(&s.Total, &s.Reclamos, &s.Quejas, &s.Pendientes, &s.EnRevision, &s.EnProceso, &s.Atendidos, &s.Cerrados, &s.Urgentes, &s.Alta)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudieron cargar las estadísticas.", "error": err.Error()})
			return
		}

		s.PorProyecto = make([]ReporteConteo, 0)
		rows, err := db.Query(ctx, `
			SELECT COALESCE(NULLIF(proyecto,''),'Sin proyecto') AS etiqueta, COUNT(*)
			FROM libro_reclamaciones
			GROUP BY 1
			ORDER BY 2 DESC, 1 ASC
			LIMIT 30
		`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var item ReporteConteo
				if rows.Scan(&item.Etiqueta, &item.Total) == nil {
					s.PorProyecto = append(s.PorProyecto, item)
				}
			}
		}

		s.PorMes = make([]ReporteConteo, 0)
		rows2, err := db.Query(ctx, `
			SELECT TO_CHAR(date_trunc('month', created_at), 'YYYY-MM') AS etiqueta, COUNT(*)
			FROM libro_reclamaciones
			GROUP BY 1
			ORDER BY 1 DESC
			LIMIT 24
		`)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var item ReporteConteo
				if rows2.Scan(&item.Etiqueta, &item.Total) == nil {
					s.PorMes = append(s.PorMes, item)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
	}
}

// ============================================================
// GET - REPORTES DETALLADOS
// ============================================================

func reportesReclamos(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		where, args := buildReclamoFilters(c)

		// Resumen del universo filtrado.
		var total int64
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM libro_reclamaciones r `+where, args...).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo generar el reporte.", "error": err.Error()})
			return
		}

		tipo := make([]ReporteConteo, 0)
		rows, err := db.Query(ctx, `SELECT tipo, COUNT(*) FROM libro_reclamaciones r `+where+` GROUP BY tipo ORDER BY COUNT(*) DESC`, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var x ReporteConteo
				if rows.Scan(&x.Etiqueta, &x.Total) == nil {
					tipo = append(tipo, x)
				}
			}
		}

		estado := make([]ReporteConteo, 0)
		rows2, err := db.Query(ctx, `SELECT estado, COUNT(*) FROM libro_reclamaciones r `+where+` GROUP BY estado ORDER BY COUNT(*) DESC`, args...)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var x ReporteConteo
				if rows2.Scan(&x.Etiqueta, &x.Total) == nil {
					estado = append(estado, x)
				}
			}
		}

		prioridad := make([]ReporteConteo, 0)
		rows3, err := db.Query(ctx, `SELECT prioridad, COUNT(*) FROM libro_reclamaciones r `+where+` GROUP BY prioridad ORDER BY COUNT(*) DESC`, args...)
		if err == nil {
			defer rows3.Close()
			for rows3.Next() {
				var x ReporteConteo
				if rows3.Scan(&x.Etiqueta, &x.Total) == nil {
					prioridad = append(prioridad, x)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"total":         total,
				"por_tipo":      tipo,
				"por_estado":    estado,
				"por_prioridad": prioridad,
			},
		})
	}
}

// ============================================================
// GET - PDF DE LA CONSTANCIA
//
// IMPORTANTE: el PDF NO se almacena como binario. Se conserva el
// contenido del registro en datos_originales JSONB y la URL de la
// constancia. El PDF se reconstruye cuando el dashboard lo solicita.
// ============================================================
func generarPDFReclamo(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseID(c)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		var (
			datos    reclamoPDFData
			original string
		)

		err := db.QueryRow(ctx, `
			SELECT
				r.id,
				r.numero_correlativo,
				r.codigo_registro,
				r.created_at,
				r.tipo,
				COALESCE(r.establecimiento, ''),
				COALESCE(r.nombres, ''),
				COALESCE(r.apellidos, ''),
				COALESCE(r.tipo_documento, ''),
				COALESCE(r.numero_documento, ''),
				COALESCE(r.domicilio, ''),
				COALESCE(r.telefono, ''),
				COALESCE(r.email, ''),
				COALESCE(r.departamento, ''),
				COALESCE(r.provincia, ''),
				COALESCE(r.distrito, ''),
				COALESCE(r.es_menor, false),
				COALESCE(r.representante_nombres, ''),
				COALESCE(r.representante_apellidos, ''),
				COALESCE(r.representante_tipo_documento, ''),
				COALESCE(r.representante_numero_documento, ''),
				COALESCE(r.representante_telefono, ''),
				COALESCE(r.representante_email, ''),
				COALESCE(r.tipo_contratado, ''),
				COALESCE(r.proyecto, ''),
				COALESCE(r.estado_proyecto, ''),
				COALESCE(r.edificio, ''),
				COALESCE(r.unidad, ''),
				COALESCE(r.numero_operacion, ''),
				COALESCE(r.producto_servicio, ''),
				r.monto_reclamado,
				COALESCE(r.detalle, ''),
				COALESCE(r.pedido_concreto, ''),
				COALESCE(r.estado, 'pendiente'),
				COALESCE(r.prioridad, 'normal'),
				COALESCE(r.responsable_nombre, ''),
				COALESCE(r.respuesta, ''),
				COALESCE(r.medio_respuesta, ''),
				r.fecha_respuesta,
				COALESCE(r.canal, ''),
				COALESCE(r.autoriza_notificacion_email, false),
				COALESCE(r.datos_originales, '{}'::jsonb)::text
			FROM libro_reclamaciones r
			WHERE r.id = $1
		`, id).Scan(
			&datos.ID,
			&datos.NumeroCorrelativo,
			&datos.Codigo,
			&datos.Fecha,
			&datos.Tipo,
			&datos.Establecimiento,
			&datos.Nombres,
			&datos.Apellidos,
			&datos.TipoDocumento,
			&datos.NumeroDocumento,
			&datos.Domicilio,
			&datos.Telefono,
			&datos.Email,
			&datos.Departamento,
			&datos.Provincia,
			&datos.Distrito,
			&datos.EsMenor,
			&datos.RepresentanteNombres,
			&datos.RepresentanteApellidos,
			&datos.RepresentanteTipoDocumento,
			&datos.RepresentanteNumeroDocumento,
			&datos.RepresentanteTelefono,
			&datos.RepresentanteEmail,
			&datos.TipoContratado,
			&datos.Proyecto,
			&datos.EstadoProyecto,
			&datos.Edificio,
			&datos.Unidad,
			&datos.NumeroOperacion,
			&datos.ProductoServicio,
			&datos.MontoReclamado,
			&datos.Detalle,
			&datos.Pedido,
			&datos.Estado,
			&datos.Prioridad,
			&datos.Responsable,
			&datos.Respuesta,
			&datos.MedioRespuesta,
			&datos.FechaRespuesta,
			&datos.Canal,
			&datos.AutorizaNotificacionEmail,
			&original,
		)

		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"message": "La reclamación no existe.",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "No se pudo obtener la información del registro.",
				"error":   err.Error(),
			})
			return
		}

		if datos.Codigo == "" {
			datos.Codigo = fmt.Sprintf("ANC-LR-%d-%06d", datos.Fecha.Year(), datos.NumeroCorrelativo)
		}

		pdf, err := buildReclamoPDF(datos, original)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "No se pudo construir el documento PDF.",
				"error":   err.Error(),
			})
			return
		}

		filename := strings.NewReplacer(
			`"`, "",
			"/", "-",
			"\\", "-",
		).Replace(datos.Codigo)

		c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s.pdf"`, filename))
		c.Header("Cache-Control", "private, no-store, max-age=0")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, "application/pdf", pdf)
	}
}

type reclamoPDFData struct {
	ID                           int64
	NumeroCorrelativo            int64
	Codigo                       string
	Fecha                        time.Time
	Tipo                         string
	Establecimiento              string
	Nombres                      string
	Apellidos                    string
	TipoDocumento                string
	NumeroDocumento              string
	Domicilio                    string
	Telefono                     string
	Email                        string
	Departamento                 string
	Provincia                    string
	Distrito                     string
	EsMenor                      bool
	RepresentanteNombres         string
	RepresentanteApellidos       string
	RepresentanteTipoDocumento   string
	RepresentanteNumeroDocumento string
	RepresentanteTelefono        string
	RepresentanteEmail           string
	TipoContratado               string
	Proyecto                     string
	EstadoProyecto               string
	Edificio                     string
	Unidad                       string
	NumeroOperacion              string
	ProductoServicio             string
	MontoReclamado               *float64
	Detalle                      string
	Pedido                       string
	Estado                       string
	Prioridad                    string
	Responsable                  string
	Respuesta                    string
	MedioRespuesta               string
	FechaRespuesta               *time.Time
	Canal                        string
	AutorizaNotificacionEmail    bool
}

const (
	ancosurRUC          = "20601146682"
	ancosurDomicilio    = "Av. San Carlos N.° 1481, Huancayo, Junín, Perú"
	ancosurTelefono     = "971 069 763"
	ancosurLegalNotice  = "La formulación del reclamo no impide acudir a otras vías de solución de controversias ni constituye requisito previo para interponer una denuncia ante el INDECOPI."
	ancosurResponseTerm = "El proveedor debe dar respuesta al reclamo o queja en un plazo máximo de quince (15) días hábiles, improrrogables."
)

type reclamoPDFBlock struct {
	kind  string
	key   string
	value string
}

func buildReclamoPDF(data reclamoPDFData, original string) ([]byte, error) {

	value := func(v string) string {
		v = strings.TrimSpace(normalizarPDF(v))
		if v == "" {
			return "No consignado"
		}
		return v
	}

	fullName := strings.TrimSpace(strings.Join(
		[]string{data.Nombres, data.Apellidos},
		" ",
	))

	location := strings.Trim(strings.Join(
		[]string{
			strings.TrimSpace(data.Departamento),
			strings.TrimSpace(data.Provincia),
			strings.TrimSpace(data.Distrito),
		},
		" / ",
	), " /")

	tipo := strings.ToUpper(value(data.Tipo))
	if tipo == "RECLAMO" {
		tipo = "RECLAMO"
	} else {
		tipo = "QUEJA"
	}

	estado := strings.ToUpper(strings.ReplaceAll(value(data.Estado), "_", " "))
	prioridad := strings.ToUpper(strings.ReplaceAll(value(data.Prioridad), "_", " "))

	providerLocation := value(data.Establecimiento)
	if providerLocation == "No consignado" {
		providerLocation = ancosurDomicilio
	}

	productOrService := value(data.ProductoServicio)
	if productOrService == "No consignado" {
		productOrService = value(data.TipoContratado)
	}

	detail := value(data.Detalle)
	pedido := value(data.Pedido)

	responseText := value(data.Respuesta)
	if strings.TrimSpace(data.Respuesta) == "" {
		responseText = "Pendiente de atención."
	}

	responseDate := "Pendiente"
	if data.FechaRespuesta != nil {
		responseDate = data.FechaRespuesta.Format("02/01/2006 15:04")
	}

	representative := strings.TrimSpace(strings.Join(
		[]string{data.RepresentanteNombres, data.RepresentanteApellidos},
		" ",
	))

	blocks := []reclamoPDFBlock{
		{kind: "header", key: "LIBRO DE RECLAMACIONES", value: "HOJA DE RECLAMACIÓN / CONSTANCIA DIGITAL"},
		{kind: "meta", key: "N.° DE HOJA", value: fmt.Sprintf("%08d", data.NumeroCorrelativo)},
		{kind: "meta", key: "FECHA", value: data.Fecha.Format("02/01/2006")},
		{kind: "meta", key: "HORA", value: data.Fecha.Format("15:04")},
		{kind: "meta", key: "TIPO", value: tipo},
		{kind: "meta", key: "CÓDIGO DE SEGUIMIENTO", value: data.Codigo},
		{kind: "section", key: "PROVEEDOR", value: ""},
		{kind: "kv", key: "Razón social", value: "ANCOSUR S.A.C."},
		{kind: "kv", key: "RUC", value: ancosurRUC},
		{kind: "kv", key: "Domicilio", value: ancosurDomicilio},
		{kind: "kv", key: "Establecimiento / código", value: providerLocation},
		{kind: "section", key: "1. IDENTIFICACIÓN DEL CONSUMIDOR RECLAMANTE", value: ""},
		{kind: "kv", key: "Nombres y apellidos", value: fullName},
		{kind: "kv", key: "DNI / CE", value: strings.TrimSpace(data.TipoDocumento + " - " + data.NumeroDocumento)},
		{kind: "kv", key: "Teléfono / celular", value: value(data.Telefono)},
		{kind: "kv", key: "Correo electrónico", value: value(data.Email)},
		{kind: "kv", key: "Domicilio", value: value(data.Domicilio)},
		{kind: "kv", key: "Ubicación", value: location},
	}

	if data.EsMenor {
		blocks = append(blocks,
			reclamoPDFBlock{kind: "kv", key: "Padre / madre / representante", value: value(representative)},
			reclamoPDFBlock{kind: "kv", key: "Documento del representante", value: strings.TrimSpace(data.RepresentanteTipoDocumento + " - " + data.RepresentanteNumeroDocumento)},
		)
		if strings.TrimSpace(data.RepresentanteTelefono) != "" {
			blocks = append(blocks, reclamoPDFBlock{kind: "kv", key: "Teléfono del representante", value: data.RepresentanteTelefono})
		}
		if strings.TrimSpace(data.RepresentanteEmail) != "" {
			blocks = append(blocks, reclamoPDFBlock{kind: "kv", key: "Correo del representante", value: data.RepresentanteEmail})
		}
	}

	blocks = append(blocks,
		reclamoPDFBlock{kind: "section", key: "2. IDENTIFICACIÓN DEL BIEN CONTRATADO", value: ""},
		reclamoPDFBlock{kind: "kv", key: "Producto / servicio", value: productOrService},
		reclamoPDFBlock{kind: "kv", key: "Proyecto relacionado", value: value(data.Proyecto)},
		reclamoPDFBlock{kind: "kv", key: "Tipo de proyecto", value: value(data.TipoContratado)},
		reclamoPDFBlock{kind: "kv", key: "Etapa / estado", value: value(data.EstadoProyecto)},
		reclamoPDFBlock{kind: "kv", key: "Edificio / torre / bloque", value: value(data.Edificio)},
		reclamoPDFBlock{kind: "kv", key: "Departamento / lote / unidad", value: value(data.Unidad)},
		reclamoPDFBlock{kind: "kv", key: "N.° de operación", value: value(data.NumeroOperacion)},
		reclamoPDFBlock{kind: "kv", key: "Monto reclamado", value: formatoMonto(data.MontoReclamado)},
		reclamoPDFBlock{kind: "section", key: "3. DETALLE DE LA RECLAMACIÓN Y PEDIDO DEL CONSUMIDOR", value: ""},
		reclamoPDFBlock{kind: "label", key: "DETALLE", value: detail},
		reclamoPDFBlock{kind: "label", key: "PEDIDO", value: pedido},
		reclamoPDFBlock{kind: "section", key: "4. OBSERVACIONES Y ACCIONES ADOPTADAS POR EL PROVEEDOR", value: ""},
		reclamoPDFBlock{kind: "kv", key: "Estado de atención", value: estado},
		reclamoPDFBlock{kind: "kv", key: "Prioridad interna", value: prioridad},
		reclamoPDFBlock{kind: "kv", key: "Responsable de atención", value: value(data.Responsable)},
		reclamoPDFBlock{kind: "kv", key: "Medio de respuesta", value: value(data.MedioRespuesta)},
		reclamoPDFBlock{kind: "kv", key: "Fecha de comunicación de respuesta", value: responseDate},
		reclamoPDFBlock{kind: "label", key: "RESPUESTA / ACCIONES ADOPTADAS", value: responseText},
		reclamoPDFBlock{kind: "section", key: "CONSTANCIA DE PRESENTACIÓN VIRTUAL", value: ""},
		reclamoPDFBlock{kind: "paragraph", key: "", value: "Registro presentado mediante la plataforma virtual del Libro de Reclamaciones. La conformidad electrónica registrada en la plataforma constituye el mecanismo implementado para acreditar la presentación y conformidad del consumidor."},
		reclamoPDFBlock{kind: "kv", key: "Canal de registro", value: value(data.Canal)},
		reclamoPDFBlock{kind: "kv", key: "Código de seguimiento", value: data.Codigo},
		reclamoPDFBlock{kind: "legal", key: "", value: ancosurResponseTerm},
		reclamoPDFBlock{kind: "legal", key: "", value: ancosurLegalNotice},
	)

	if len(original) > 0 {
		blocks = append(blocks,
			reclamoPDFBlock{
				kind:  "audit",
				key:   "Identificador interno de integridad",
				value: fmt.Sprintf("%x", simpleHash([]byte(original))),
			},
		)
	}

	return buildCorporateReclamoPDF(blocks, data.Codigo)
}

func buildCorporateReclamoPDF(blocks []reclamoPDFBlock, codigo string) ([]byte, error) {
	// A4 en puntos PDF.
	const (
		pageW        = 595.0
		pageH        = 842.0
		marginLeft   = 42.0
		marginRight  = 42.0
		marginTop    = 112.0
		marginBottom = 48.0

		greenR = 0.0
		greenG = 0.654902
		greenB = 0.309804

		blackR = 0.055
		blackG = 0.055
		blackB = 0.055

		grayR = 0.38
		grayG = 0.38
		grayB = 0.38

		lightR = 0.965
		lightG = 0.972
		lightB = 0.968

		borderR = 0.86
		borderG = 0.88
		borderB = 0.87
	)

	type line struct {
		text  string
		style string
	}

	parsed := make([]line, 0, len(blocks)*3)

	appendWrapped := func(text, style string, max int) {
		text = normalizarPDF(strings.TrimSpace(text))
		if text == "" {
			return
		}
		for _, part := range wrapPDFLine(text, max) {
			parsed = append(parsed, line{text: part, style: style})
		}
	}

	for _, block := range blocks {
		switch block.kind {
		case "header":
			appendWrapped(block.key, "title", 56)
			appendWrapped(block.value, "subtitle", 78)
		case "section":
			parsed = append(parsed, line{text: block.key, style: "section"})
		case "meta":
			parsed = append(parsed, line{text: block.key + ": " + block.value, style: "meta"})
		case "kv":
			appendWrapped(block.key+": "+block.value, "kv", 84)
		case "label":
			parsed = append(parsed, line{text: block.key, style: "label"})
			appendWrapped(block.value, "body", 88)
			parsed = append(parsed, line{text: "", style: "space"})
		case "paragraph":
			appendWrapped(block.value, "body", 88)
			parsed = append(parsed, line{text: "", style: "space"})
		case "legal":
			appendWrapped(block.value, "legal", 88)
			parsed = append(parsed, line{text: "", style: "space"})
		case "audit":
			appendWrapped(block.key+": "+block.value, "audit", 88)
		}
	}

	type page struct {
		lines []line
	}
	pages := make([]page, 0, 4)
	current := page{lines: make([]line, 0, 70)}
	y := pageH - marginTop

	lineHeight := func(style string) float64 {
		switch style {
		case "title":
			return 25
		case "subtitle":
			return 16
		case "section":
			return 25
		case "meta":
			return 18
		case "kv":
			return 15
		case "label":
			return 16
		case "legal":
			return 11
		case "audit":
			return 10
		case "space":
			return 7
		default:
			return 14
		}
	}

	flush := func() {
		if len(current.lines) > 0 {
			pages = append(pages, current)
			current = page{lines: make([]line, 0, 70)}
		}
		y = pageH - marginTop
	}

	for _, item := range parsed {
		h := lineHeight(item.style)
		if y-h < marginBottom {
			flush()
		}
		current.lines = append(current.lines, item)
		y -= h
	}
	flush()

	if len(pages) == 0 {
		pages = append(pages, page{lines: []line{{text: "Sin información", style: "body"}}})
	}

	var pdf strings.Builder
	pdf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	objects := make([]string, 0, 8+len(pages)*2)
	catalogID := 1
	pagesID := 2
	fontRegularID := 3
	fontBoldID := 4
	nextID := 5

	pageIDs := make([]int, len(pages))
	contentIDs := make([]int, len(pages))
	for i := range pages {
		pageIDs[i] = nextID
		nextID++
		contentIDs[i] = nextID
		nextID++
	}

	objects = append(objects,
		fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID),
	)
	kids := make([]string, 0, len(pageIDs))
	for _, id := range pageIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}
	objects = append(objects,
		fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pageIDs), strings.Join(kids, " ")),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>",
	)

	for pageIndex, pg := range pages {
		var content strings.Builder

		// Franja superior.
		content.WriteString("q\n")
		fmt.Fprintf(&content, "%.6f %.6f %.6f rg\n", greenR, greenG, greenB)
		fmt.Fprintf(&content, "0 %.2f %.2f 78 re f\n", pageH-78, pageW)
		content.WriteString("Q\n")

		// Marca corporativa.
		writePDFText(&content, "F2", 17, marginLeft, pageH-31,
			"ANCOSUR", 1, 1, 1)
		writePDFText(&content, "F1", 7, marginLeft+1, pageH-44,
			"INMOBILIARIA", 1, 1, 1)
		writePDFText(&content, "F2", 7, pageW-marginRight-108, pageH-31,
			"DOCUMENTO DE GESTION", 1, 1, 1)

		// Código de seguimiento en la cabecera.
		writePDFText(&content, "F1", 7, pageW-marginRight-108, pageH-47,
			normalizarPDF(codigo), 1, 1, 1)

		// Marco lateral sutil.
		content.WriteString("q\n")
		fmt.Fprintf(&content, "%.6f %.6f %.6f RG\n", borderR, borderG, borderB)
		fmt.Fprintf(&content, "%.2f w\n", 0.6)
		fmt.Fprintf(&content, "%.2f %.2f %.2f %.2f re S\n",
			marginLeft-12, 48.0, pageW-marginLeft-marginRight+24, pageH-108)
		content.WriteString("Q\n")

		currentY := pageH - 100

		for _, item := range pg.lines {
			switch item.style {
			case "title":
				writePDFText(&content, "F2", 18, marginLeft, currentY,
					item.text, blackR, blackG, blackB)
				currentY -= 24

			case "subtitle":
				writePDFText(&content, "F1", 8.5, marginLeft, currentY,
					item.text, grayR, grayG, grayB)
				currentY -= 18

			case "section":
				currentY -= 4

				content.WriteString("q\n")
				fmt.Fprintf(&content, "%.6f %.6f %.6f rg\n", greenR, greenG, greenB)
				fmt.Fprintf(&content, "%.2f %.2f %.2f %.2f re f\n",
					marginLeft, currentY-5, 4, 18)

				fmt.Fprintf(&content, "%.6f %.6f %.6f rg\n", lightR, lightG, lightB)
				fmt.Fprintf(&content, "%.2f %.2f %.2f %.2f re f\n",
					marginLeft+7, currentY-5, pageW-marginLeft-marginRight-7, 18)
				content.WriteString("Q\n")

				writePDFText(&content, "F2", 8.5, marginLeft+15, currentY,
					item.text, blackR, blackG, blackB)
				currentY -= 25

			case "meta":
				// Caja de metadatos compacta.
				content.WriteString("q\n")
				fmt.Fprintf(&content, "%.6f %.6f %.6f rg\n", 0.95, 0.955, 0.952)
				fmt.Fprintf(&content, "%.2f %.2f %.2f %.2f re f\n",
					marginLeft, currentY-6, pageW-marginLeft-marginRight, 20)
				content.WriteString("Q\n")

				writePDFKeyValue(&content, item.text, marginLeft+8, currentY,
					greenR, greenG, greenB, blackR, blackG, blackB)
				currentY -= 21

			case "kv":
				usedLines := writePDFKeyValue(&content, item.text, marginLeft+4, currentY,
					grayR, grayG, grayB, blackR, blackG, blackB)
				currentY -= 15 * float64(usedLines)

			case "label":
				writePDFText(&content, "F2", 8, marginLeft+4, currentY,
					item.text, greenR, greenG, greenB)
				currentY -= 15

			case "body":
				writePDFText(&content, "F1", 8.5, marginLeft+4, currentY,
					item.text, blackR, blackG, blackB)
				currentY -= 14

			case "legal":
				writePDFText(&content, "F1", 7, marginLeft+4, currentY,
					item.text, grayR, grayG, grayB)
				currentY -= 11

			case "audit":
				writePDFText(&content, "F1", 6.5, marginLeft+4, currentY,
					item.text, grayR, grayG, grayB)
				currentY -= 10

			case "space":
				currentY -= 7
			}
		}

		// Pie.
		content.WriteString("q\n")
		fmt.Fprintf(&content, "%.6f %.6f %.6f RG\n", 0.82, 0.82, 0.82)
		fmt.Fprintf(&content, "0.5 w\n")
		fmt.Fprintf(&content, "%.2f 38 m %.2f 38 l S\n", marginLeft, pageW-marginRight)
		content.WriteString("Q\n")

		writePDFText(&content, "F1", 6.5, marginLeft, 25,
			"ANCOSUR S.A.C. | RUC "+ancosurRUC+" | "+ancosurTelefono,
			grayR, grayG, grayB)
		writePDFText(&content, "F1", 6.5, pageW-marginRight-62, 25,
			fmt.Sprintf("Página %d de %d", pageIndex+1, len(pages)),
			grayR, grayG, grayB)

		contentBytes := []byte(content.String())

		pageObject := fmt.Sprintf(
			"<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.0f %.0f] "+
				"/Resources << /Font << /F1 %d 0 R /F2 %d 0 R >> >> "+
				"/Contents %d 0 R >>",
			pagesID, pageW, pageH,
			fontRegularID, fontBoldID, contentIDs[pageIndex],
		)

		contentObject := fmt.Sprintf(
			"<< /Length %d >>\nstream\n%s\nendstream",
			len(contentBytes), contentBytes,
		)

		objects = append(objects, pageObject, contentObject)
	}

	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		objectID := i + 1
		offsets[objectID] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", objectID, object)
	}

	xrefOffset := pdf.Len()
	pdf.WriteString("xref\n")
	fmt.Fprintf(&pdf, "0 %d\n", len(objects)+1)
	pdf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}
	pdf.WriteString("trailer\n")
	fmt.Fprintf(&pdf, "<< /Size %d /Root %d 0 R >>\n", len(objects)+1, catalogID)
	pdf.WriteString("startxref\n")
	fmt.Fprintf(&pdf, "%d\n", xrefOffset)
	pdf.WriteString("%%EOF\n")

	return []byte(pdf.String()), nil
}

// writePDFText escribe texto PDF usando fuentes estándar.
func writePDFText(
	b *strings.Builder,
	font string,
	size float64,
	x float64,
	y float64,
	text string,
	r float64,
	g float64,
	bl float64,
) {
	text = escapePDFText(normalizarPDF(text))

	fmt.Fprintf(
		b,
		"BT /%s %.2f Tf %.6f %.6f %.6f rg %.2f %.2f Td (%s) Tj ET\n",
		font, size, r, g, bl, x, y, text,
	)
}

func writePDFKeyValue(
	b *strings.Builder,
	line string,
	x float64,
	y float64,
	keyR float64,
	keyG float64,
	keyB float64,
	valueR float64,
	valueG float64,
	valueB float64,
) int {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		writePDFText(b, "F1", 8.5, x, y, line, valueR, valueG, valueB)
		return 1
	}

	key := strings.TrimSpace(parts[0]) + ":"
	value := strings.TrimSpace(parts[1])
	if value == "" {
		value = "No consignado"
	}

	writePDFText(b, "F2", 8, x, y, key, keyR, keyG, keyB)

	// Ancho aproximado para Helvetica. El valor se parte aquí y el
	// llamador reserva exactamente el número de líneas utilizadas, evitando
	// que valores largos (domicilios, descripciones, etc.) se superpongan.
	keyWidth := float64(len([]rune(key)))*4.35 + 7
	maxWidth := 595.0 - 42.0 - keyWidth - x
	maxChars := int(maxWidth / 4.5)
	if maxChars < 18 {
		maxChars = 18
	}
	valueLines := wrapPDFLine(value, maxChars)
	if len(valueLines) == 0 {
		valueLines = []string{"No consignado"}
	}

	for i, valueLine := range valueLines {
		writePDFText(b, "F1", 8.5, x+keyWidth, y-float64(i)*11,
			valueLine, valueR, valueG, valueB)
	}

	return len(valueLines)
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "(", `\(`)
	s = strings.ReplaceAll(s, ")", `\)`)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// ============================================================
// SQL / SCAN
// ============================================================

func reclamoSelectByID() string {
	return `
		SELECT
			r.id, r.numero_correlativo, r.codigo_registro, r.anio, r.tipo,
			COALESCE(r.establecimiento,''), r.nombres, r.apellidos,
			r.tipo_documento, r.numero_documento,
			COALESCE(r.domicilio,''), COALESCE(r.telefono,''), r.email,
			COALESCE(r.departamento,''), COALESCE(r.provincia,''), COALESCE(r.distrito,''),
			COALESCE(r.es_menor,false),
			COALESCE(r.representante_nombres,''), COALESCE(r.representante_apellidos,''),
			COALESCE(r.representante_tipo_documento,''), COALESCE(r.representante_numero_documento,''),
			COALESCE(r.representante_telefono,''), COALESCE(r.representante_email,''),
			COALESCE(r.tipo_contratado,''), r.proyecto, COALESCE(r.estado_proyecto,''),
			COALESCE(r.edificio,''), COALESCE(r.unidad,''), COALESCE(r.numero_operacion,''),
			COALESCE(r.producto_servicio,''), r.monto_reclamado,
			r.detalle, r.pedido_concreto, r.estado, r.prioridad,
			r.responsable_id, COALESCE(r.responsable_nombre,''),
			COALESCE(r.respuesta,''), r.fecha_respuesta, COALESCE(r.medio_respuesta,''),
			COALESCE(r.correo_constancia_estado,'pendiente'), r.correo_constancia_enviado_at,
			r.correo_constancia_entregado_at, COALESCE(r.correo_constancia_message_id,''),
			COALESCE(r.correo_constancia_error,''),
			COALESCE(r.correo_interno_estado,'pendiente'), r.correo_interno_enviado_at,
			r.correo_interno_entregado_at, COALESCE(r.correo_interno_message_id,''),
			COALESCE(r.correo_interno_error,''), COALESCE(r.autoriza_notificacion_email,false),
			COALESCE(r.constancia_url,''), r.constancia_generada_at,
			COALESCE(r.canal,''), COALESCE(r.ip_origen::text,''), COALESCE(r.user_agent,''),
			r.created_at, r.updated_at, COALESCE(r.datos_originales,'{}'::jsonb)::text
		FROM libro_reclamaciones r
		WHERE r.id = $1
	`
}

func reclamoScanDestinations(r *ReclamoWeb, includeOriginal bool) []any {
	base := []any{
		&r.ID, &r.NumeroCorrelativo, &r.CodigoRegistro, &r.Anio, &r.Tipo,
		&r.Establecimiento, &r.Nombres, &r.Apellidos,
		&r.TipoDocumento, &r.NumeroDocumento,
		&r.Domicilio, &r.Telefono, &r.Email,
		&r.Departamento, &r.Provincia, &r.Distrito, &r.EsMenor,
		&r.RepresentanteNombres, &r.RepresentanteApellidos, &r.RepresentanteTipoDocumento,
		&r.RepresentanteNumeroDocumento, &r.RepresentanteTelefono, &r.RepresentanteEmail,
		&r.TipoContratado, &r.Proyecto, &r.EstadoProyecto,
		&r.Edificio, &r.Unidad, &r.NumeroOperacion, &r.ProductoServicio, &r.MontoReclamado,
		&r.Detalle, &r.PedidoConcreto, &r.Estado, &r.Prioridad,
		&r.ResponsableID, &r.ResponsableNombre, &r.Respuesta, &r.FechaRespuesta, &r.MedioRespuesta,
		&r.CorreoConstanciaEstado, &r.CorreoConstanciaEnviadoAt, &r.CorreoConstanciaEntregadoAt,
		&r.CorreoConstanciaMessageID, &r.CorreoConstanciaError,
		&r.CorreoInternoEstado, &r.CorreoInternoEnviadoAt, &r.CorreoInternoEntregadoAt,
		&r.CorreoInternoMessageID, &r.CorreoInternoError,
		&r.AutorizaNotificacionEmail, &r.ConstanciaURL, &r.ConstanciaGeneradaAt,
		&r.Canal, &r.IPOrigen, &r.UserAgent, &r.CreatedAt, &r.UpdatedAt,
	}
	if includeOriginal {
		var original string
		base = append(base, &original)
		// El caller que use esta función por id solo necesita el campo para JSON. Lo asignamos
		// mediante wrapper en scan directo cuando sea necesario.
		return base
	}
	return base
}

// scanRows de listado reutiliza la misma cantidad de columnas, sin datos_originales.
func scanReclamo(rows pgx.Rows, r *ReclamoWeb, includeOriginal bool) error {
	dests := reclamoScanDestinations(r, false)
	if includeOriginal {
		var raw string
		dests = append(dests, &raw)
		if err := rows.Scan(dests...); err != nil {
			return err
		}
		if raw != "" {
			r.DatosOriginales = json.RawMessage(raw)
		}
		return nil
	}
	return rows.Scan(dests...)
}

// ============================================================
// FILTROS
// ============================================================

func buildReclamoFilters(c *gin.Context) (string, []any) {
	parts := make([]string, 0)
	args := make([]any, 0)
	add := func(expr string, value any) {
		args = append(args, value)
		parts = append(parts, fmt.Sprintf(expr, len(args)))
	}

	if v := strings.TrimSpace(c.Query("q")); v != "" {
		args = append(args, v)
		pos := len(args)
		parts = append(parts, fmt.Sprintf(`(LOWER(r.codigo_registro) LIKE LOWER('%%' || $%d || '%%')
			OR LOWER(r.nombres || ' ' || r.apellidos) LIKE LOWER('%%' || $%d || '%%')
			OR LOWER(r.email) LIKE LOWER('%%' || $%d || '%%')
			OR LOWER(r.numero_documento) LIKE LOWER('%%' || $%d || '%%')
			OR LOWER(r.proyecto) LIKE LOWER('%%' || $%d || '%%'))`, pos, pos, pos, pos, pos))
	}
	// Se expande correctamente el placeholder repetido mediante Rebind manual.
	if v := strings.TrimSpace(c.Query("estado")); v != "" {
		add(`r.estado = $%d`, v)
	}
	if v := strings.TrimSpace(c.Query("tipo")); v != "" {
		add(`r.tipo = $%d`, v)
	}
	if v := strings.TrimSpace(c.Query("prioridad")); v != "" {
		add(`r.prioridad = $%d`, v)
	}
	if v := strings.TrimSpace(c.Query("proyecto")); v != "" {
		add(`r.proyecto = $%d`, v)
	}
	if v := strings.TrimSpace(c.Query("desde")); v != "" {
		if _, err := time.Parse("2006-01-02", v); err == nil {
			add(`r.created_at >= $%d::date`, v)
		}
	}
	if v := strings.TrimSpace(c.Query("hasta")); v != "" {
		if _, err := time.Parse("2006-01-02", v); err == nil {
			add(`r.created_at < ($%d::date + INTERVAL '1 day')`, v)
		}
	}

	if len(parts) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(parts, " AND "), args
}

// ============================================================
// VALIDACION / NORMALIZACION
// ============================================================

func normalizarReclamo(r *ReclamoWebRequest) {
	r.Establecimiento = strings.TrimSpace(r.Establecimiento)
	r.Tipo = strings.ToLower(strings.TrimSpace(r.Tipo))
	r.Nombres = strings.TrimSpace(r.Nombres)
	r.Apellidos = strings.TrimSpace(r.Apellidos)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Telefono = soloNumeros(r.Telefono)
	r.TipoDocumento = strings.ToUpper(strings.TrimSpace(r.TipoDocumento))
	r.NumeroDocumento = strings.TrimSpace(r.NumeroDocumento)
	r.Departamento = strings.TrimSpace(r.Departamento)
	r.Provincia = strings.TrimSpace(r.Provincia)
	r.Distrito = strings.TrimSpace(r.Distrito)
	r.Domicilio = strings.TrimSpace(r.Domicilio)
	r.TutorNombres = strings.TrimSpace(r.TutorNombres)
	r.TutorApellidos = strings.TrimSpace(r.TutorApellidos)
	r.TutorTipoDocumento = strings.ToUpper(strings.TrimSpace(r.TutorTipoDocumento))
	r.TutorNumeroDocumento = strings.TrimSpace(r.TutorNumeroDocumento)
	r.TutorTelefono = soloNumeros(r.TutorTelefono)
	r.TutorEmail = strings.ToLower(strings.TrimSpace(r.TutorEmail))
	r.TipoBien = strings.TrimSpace(r.TipoBien)
	r.Proyecto = strings.TrimSpace(r.Proyecto)
	r.TipoProyecto = strings.TrimSpace(r.TipoProyecto)
	r.EstadoProyecto = strings.TrimSpace(r.EstadoProyecto)
	r.CiudadProyecto = strings.TrimSpace(r.CiudadProyecto)
	r.DireccionProyecto = strings.TrimSpace(r.DireccionProyecto)
	r.EdificioTorreBloque = strings.TrimSpace(r.EdificioTorreBloque)
	r.UnidadInmobiliaria = strings.TrimSpace(r.UnidadInmobiliaria)
	r.NumeroOperacion = strings.TrimSpace(r.NumeroOperacion)
	r.DescripcionProductoServicio = strings.TrimSpace(r.DescripcionProductoServicio)
	r.Detalle = strings.TrimSpace(r.Detalle)
	r.Pedido = strings.TrimSpace(r.Pedido)
	r.CorreoNotificacion = strings.ToLower(strings.TrimSpace(r.CorreoNotificacion))
	if !r.EsMenor {
		r.TutorNombres = ""
		r.TutorApellidos = ""
		r.TutorTipoDocumento = ""
		r.TutorNumeroDocumento = ""
		r.TutorTelefono = ""
		r.TutorEmail = ""
	}
}

func validarReclamo(r ReclamoWebRequest) error {
	if r.Tipo != "reclamo" && r.Tipo != "queja" {
		return errorString("Debe seleccionar si corresponde a reclamo o queja.")
	}
	if len([]rune(r.Nombres)) < 2 {
		return errorString("Los nombres no son válidos.")
	}
	if len([]rune(r.Apellidos)) < 2 {
		return errorString("Los apellidos no son válidos.")
	}
	if !emailValido(r.Email) {
		return errorString("El correo electrónico no es válido.")
	}
	if r.NumeroDocumento == "" {
		return errorString("El número de documento es obligatorio.")
	}
	if r.Telefono != "" && !telefonoValido(r.Telefono) {
		return errorString("El celular debe tener 9 dígitos y comenzar con 9.")
	}
	if r.Proyecto == "" {
		return errorString("Debe seleccionar un proyecto.")
	}
	if r.Detalle == "" {
		return errorString("El detalle de la reclamación es obligatorio.")
	}
	if r.Pedido == "" {
		return errorString("El pedido concreto es obligatorio.")
	}
	if r.Conformidad == nil || !*r.Conformidad {
		return errorString("Debe aceptar la declaración de envío.")
	}
	if r.AutorizaNotificacionEmail && r.CorreoNotificacion != "" && !emailValido(r.CorreoNotificacion) {
		return errorString("El correo de notificación no es válido.")
	}
	if r.EsMenor && (r.TutorNombres == "" || r.TutorApellidos == "" || r.TutorNumeroDocumento == "") {
		return errorString("Debe completar los datos del representante del menor.")
	}
	return nil
}

type errorString string

func (e errorString) Error() string { return string(e) }

func soloNumeros(value string) string {
	var b strings.Builder
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func telefonoValido(v string) bool {
	if len(v) != 9 || v[0] != '9' {
		return false
	}
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func emailValido(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || strings.Count(v, "@") != 1 {
		return false
	}
	p := strings.SplitN(v, "@", 2)
	return p[0] != "" && p[1] != "" && strings.Contains(p[1], ".")
}

func montoNumerico(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func formatoMonto(v *float64) string {
	if v == nil {
		return "No especificado"
	}
	return fmt.Sprintf("S/ %.2f", *v)
}

// ============================================================
// UTILIDADES
// ============================================================

func contarAdjuntos(c *gin.Context) int {
	if c.Request.MultipartForm == nil {
		return 0
	}
	return len(c.Request.MultipartForm.File["adjuntos"])
}

func parsePositiveInt(v string, fallback int) int {
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseID(c *gin.Context) (int64, bool) {
	value := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(value, 10, 64)
	if value == "" || err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "El ID de la reclamación no es válido."})
		return 0, false
	}
	return id, true
}

func obtenerIPCliente(c *gin.Context) string {
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); forwarded != "" {
		ip := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	if realIP := strings.TrimSpace(c.GetHeader("X-Real-IP")); realIP != "" && net.ParseIP(realIP) != nil {
		return realIP
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil && net.ParseIP(host) != nil {
		return host
	}
	return ""
}

func wrapPDFLine(s string, max int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	result := make([]string, 0)
	current := ""
	for _, word := range words {
		if len(current)+len(word)+1 <= max {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
			continue
		}
		if current != "" {
			result = append(result, current)
		}
		current = word
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func normalizarPDF(s string) string {
	// 1) Corrige mojibake frecuente producido cuando UTF-8 fue interpretado
	//    como Latin-1/Windows-1252. Ej.: "N.Â°" -> "N.°" y "InformaciÃ³n" -> "Información".
	s = repararMojibake(s)

	// 2) Convierte Unicode a bytes Windows-1252/WinAnsi. Helvetica Type1
	//    estándar no entiende UTF-8 directamente; escribir los bytes UTF-8
	//    crudos es precisamente lo que provoca textos como "Â°" en algunos visores.
	return unicodeAWinAnsi(s)
}

func repararMojibake(s string) string {
	if !strings.ContainsAny(s, "ÃÂâð") {
		return s
	}

	// Revierte texto UTF-8 que fue decodificado erróneamente como
	// ISO-8859-1/Windows-1252. Esto cubre casos como:
	//   N.Â°        -> N.°
	//   InformaciÃ³n -> Información
	//   â€“         -> –
	//   â€”         -> —
	buf := make([]byte, 0, len(s))
	for _, r := range s {
		if v, ok := runeAWin1252Byte(r); ok {
			buf = append(buf, v)
			continue
		}
		return s
	}

	candidate := string(buf)
	if !utf8.ValidString(candidate) {
		return s
	}

	badBefore := contarMojibake(s)
	badAfter := contarMojibake(candidate)
	if badAfter < badBefore {
		return candidate
	}
	return s
}

func contarMojibake(s string) int {
	return strings.Count(s, "Ã") +
		strings.Count(s, "Â") +
		strings.Count(s, "â") +
		strings.Count(s, "ð") +
		strings.Count(s, "�")
}

func runeAWin1252Byte(r rune) (byte, bool) {
	if r >= 0 && r <= 0x7F {
		return byte(r), true
	}
	if r >= 0xA0 && r <= 0xFF {
		return byte(r), true
	}

	switch r {
	case 0x20AC:
		return 0x80, true
	case 0x201A:
		return 0x82, true
	case 0x192:
		return 0x83, true
	case 0x201E:
		return 0x84, true
	case 0x2026:
		return 0x85, true
	case 0x2020:
		return 0x86, true
	case 0x2021:
		return 0x87, true
	case 0x2C6:
		return 0x88, true
	case 0x2030:
		return 0x89, true
	case 0x160:
		return 0x8A, true
	case 0x2039:
		return 0x8B, true
	case 0x152:
		return 0x8C, true
	case 0x17D:
		return 0x8E, true
	case 0x2018:
		return 0x91, true
	case 0x2019:
		return 0x92, true
	case 0x201C:
		return 0x93, true
	case 0x201D:
		return 0x94, true
	case 0x2022:
		return 0x95, true
	case 0x2013:
		return 0x96, true
	case 0x2014:
		return 0x97, true
	case 0x2DC:
		return 0x98, true
	case 0x2122:
		return 0x99, true
	case 0x161:
		return 0x9A, true
	case 0x203A:
		return 0x9B, true
	case 0x153:
		return 0x9C, true
	case 0x17E:
		return 0x9E, true
	case 0x178:
		return 0x9F, true
	default:
		return 0, false
	}
}

func unicodeAWinAnsi(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		switch {
		case r >= 0x20 && r <= 0x7E:
			b.WriteByte(byte(r))
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteByte(' ')
		case r >= 0xA0 && r <= 0xFF:
			b.WriteByte(byte(r))
		default:
			// Caracteres Unicode habituales que no ocupan las posiciones
			// correspondientes en WinAnsi.
			mapped := byte(0)
			switch r {
			case 0x20AC: // €
				mapped = 0x80
			case 0x201A: // ‚
				mapped = 0x82
			case 0x192: // ƒ
				mapped = 0x83
			case 0x201E: // „
				mapped = 0x84
			case 0x2026: // …
				mapped = 0x85
			case 0x2020: // †
				mapped = 0x86
			case 0x2021: // ‡
				mapped = 0x87
			case 0x2C6: // ˆ
				mapped = 0x88
			case 0x2030: // ‰
				mapped = 0x89
			case 0x160: // Š
				mapped = 0x8A
			case 0x2039: // ‹
				mapped = 0x8B
			case 0x152: // Œ
				mapped = 0x8C
			case 0x17D: // Ž
				mapped = 0x8E
			case 0x2018: // ‘
				mapped = 0x91
			case 0x2019: // ’
				mapped = 0x92
			case 0x201C: // “
				mapped = 0x93
			case 0x201D: // ”
				mapped = 0x94
			case 0x2022: // •
				mapped = 0x95
			case 0x2013: // –
				mapped = 0x96
			case 0x2014: // —
				mapped = 0x97
			case 0x2DC: // ˜
				mapped = 0x98
			case 0x2122: // ™
				mapped = 0x99
			case 0x161: // š
				mapped = 0x9A
			case 0x203A: // ›
				mapped = 0x9B
			case 0x153: // œ
				mapped = 0x9C
			case 0x17E: // ž
				mapped = 0x9E
			case 0x178: // Ÿ
				mapped = 0x9F
			case 0x2212: // −
				mapped = '-'
			case 0x00B7: // ·
				mapped = 0xB7
			case 0x00B0: // °
				mapped = 0xB0
			default:
				// Para cualquier carácter fuera de WinAnsi usamos una marca
				// legible en vez de emitir UTF-8 inválido para una fuente Type1.
				mapped = '?'
			}
			b.WriteByte(mapped)
		}
	}
	return b.String()
}

// Hash sencillo para auditoría visual del PDF. No pretende ser un hash criptográfico.
func simpleHash(data []byte) uint64 {
	var h uint64 = 1469598103934665603
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}
