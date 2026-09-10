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
			codigo       string
			fecha        time.Time
			tipo         string
			nombres      string
			apellidos    string
			tipoDoc      string
			numDoc       string
			email        string
			telefono     string
			domicilio    string
			departamento string
			provincia    string
			distrito     string
			proyecto     string
			estadoProj   string
			edificio     string
			unidad       string
			numOperacion string
			producto     string
			monto        *float64
			detalle      string
			pedido       string
			estado       string
			prioridad    string
			respuesta    string
			medio        string
			fechaResp    *time.Time
			original     []byte
		)

		err := db.QueryRow(ctx, `
			SELECT codigo_registro, created_at, tipo,
			       nombres, apellidos, tipo_documento, numero_documento,
			       email, COALESCE(telefono,''), COALESCE(domicilio,''),
			       COALESCE(departamento,''), COALESCE(provincia,''), COALESCE(distrito,''),
			       proyecto, COALESCE(estado_proyecto,''), COALESCE(edificio,''), COALESCE(unidad,''),
			       COALESCE(numero_operacion,''), COALESCE(producto_servicio,''), monto_reclamado,
			       detalle, pedido_concreto, estado, prioridad, COALESCE(respuesta,''),
			       COALESCE(medio_respuesta,''), fecha_respuesta, COALESCE(datos_originales,'{}'::jsonb)::text
			FROM libro_reclamaciones
			WHERE id = $1
		`, id).Scan(&codigo, &fecha, &tipo, &nombres, &apellidos, &tipoDoc, &numDoc, &email, &telefono, &domicilio,
			&departamento, &provincia, &distrito, &proyecto, &estadoProj, &edificio, &unidad, &numOperacion,
			&producto, &monto, &detalle, &pedido, &estado, &prioridad, &respuesta, &medio, &fechaResp, &original)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "La reclamación no existe."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo generar la constancia.", "error": err.Error()})
			return
		}

		lines := []string{
			"ANCOSUR",
			"LIBRO DE RECLAMACIONES - CONSTANCIA",
			"",
			"Codigo de registro: " + codigo,
			"Fecha de registro: " + fecha.Format("02/01/2006 15:04"),
			"Tipo: " + tipo,
			"",
			"DATOS DEL CONSUMIDOR",
			"Nombres: " + nombres + " " + apellidos,
			"Documento: " + tipoDoc + " - " + numDoc,
			"Correo: " + email,
			"Telefono: " + telefono,
			"Domicilio: " + domicilio,
			"Ubicacion: " + strings.TrimSpace(strings.Join([]string{departamento, provincia, distrito}, " / ")),
			"",
			"INFORMACION DEL PROYECTO / SERVICIO",
			"Proyecto: " + proyecto,
			"Estado del proyecto: " + estadoProj,
			"Edificio/Torre/Bloque: " + edificio,
			"Unidad: " + unidad,
			"N. de operacion: " + numOperacion,
			"Producto/Servicio: " + producto,
			"Monto reclamado: " + formatoMonto(monto),
			"",
			"CONTENIDO",
			"Detalle:",
		}
		lines = append(lines, strings.Split(normalizarPDF(detalle), "\n")...)
		lines = append(lines, "", "Pedido concreto:")
		lines = append(lines, strings.Split(normalizarPDF(pedido), "\n")...)
		lines = append(lines, "", "ESTADO DE ATENCION")
		lines = append(lines, "Estado: "+estado, "Prioridad: "+prioridad)
		if respuesta != "" {
			lines = append(lines, "Medio de respuesta: "+medio)
			if fechaResp != nil {
				lines = append(lines, "Fecha de respuesta: "+fechaResp.Format("02/01/2006 15:04"))
			}
			lines = append(lines, "Respuesta:")
			lines = append(lines, strings.Split(normalizarPDF(respuesta), "\n")...)
		}
		lines = append(lines, "", "Documento generado desde el registro conservado en la base de datos.")

		// Incluimos una huella del JSON original para comprobar que la constancia
		// corresponde al contenido almacenado.
		if len(original) > 0 {
			lines = append(lines, "Hash-logico: "+fmt.Sprintf("%x", simpleHash(original)))
		}

		pdf, err := buildSimplePDF(lines)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "No se pudo construir el PDF.", "error": err.Error()})
			return
		}

		filename := codigo
		if filename == "" {
			filename = fmt.Sprintf("reclamo-%d", id)
		}
		filename = strings.ReplaceAll(filename, "\"", "")

		c.Data(http.StatusOK, "application/pdf", pdf)
		c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s.pdf"`, filename))
		c.Header("Cache-Control", "private, no-store")
	}
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

// ============================================================
// PDF SIMPLE SIN DEPENDENCIAS EXTERNAS
// ============================================================

// buildSimplePDF genera un PDF de texto multipágina con fuentes estándar.
// Se usa deliberadamente una implementación pequeña para que reclamos.go
// no dependa de otro módulo PDF. Para caracteres fuera de WinAnsi se hace
// transliteración básica.
func buildSimplePDF(lines []string) ([]byte, error) {
	const (
		pageW = 595
		pageH = 842
		left  = 45
		topY  = 800
		step  = 16
	)

	wrapped := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		clean := normalizarPDF(line)
		if clean == "" {
			wrapped = append(wrapped, "")
			continue
		}
		parts := wrapPDFLine(clean, 90)
		wrapped = append(wrapped, parts...)
	}

	pages := make([][]string, 0)
	current := make([]string, 0)
	maxLines := int((topY - 45) / step)
	for _, line := range wrapped {
		if len(current) >= maxLines {
			pages = append(pages, current)
			current = make([]string, 0)
		}
		current = append(current, line)
	}
	if len(current) > 0 || len(pages) == 0 {
		pages = append(pages, current)
	}

	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	b.WriteString("%\xE2\xE3\xCF\xD3\n")

	objects := make([]string, 0, 3+len(pages)*2)
	catalogID := 1
	pagesID := 2
	fontID := 3
	nextID := 4
	pageIDs := make([]int, 0, len(pages))
	contentIDs := make([]int, 0, len(pages))

	for range pages {
		pageIDs = append(pageIDs, nextID)
		nextID++
		contentIDs = append(contentIDs, nextID)
		nextID++
	}

	kids := make([]string, 0, len(pageIDs))
	for _, id := range pageIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}

	objects = append(objects,
		fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID),
		fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pageIDs), strings.Join(kids, " ")),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	)

	for i := range pages {
		content := pdfPageContent(pages[i], left, topY, step)
		objects = append(objects,
			fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", pagesID, pageW, pageH, fontID, contentIDs[i]),
			fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		)
	}

	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = b.Len()
		b.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, obj))
	}

	xref := b.Len()
	b.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	b.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, catalogID, xref))

	return []byte(b.String()), nil
}

func pdfPageContent(lines []string, left, topY, step int) string {
	var b strings.Builder
	b.WriteString("BT\n/F1 10 Tf\n")
	y := topY
	for _, line := range lines {
		if line == "" {
			y -= step
			continue
		}
		escaped := strings.ReplaceAll(line, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, "(", `\(`)
		escaped = strings.ReplaceAll(escaped, ")", `\)`)
		b.WriteString(fmt.Sprintf("1 0 0 1 %d %d Tm (%s) Tj\n", left, y, escaped))
		y -= step
	}
	b.WriteString("ET\n")
	return b.String()
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
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N",
		"“", "\"", "”", "\"", "’", "'", "–", "-", "—", "-",
	)
	return r.Replace(s)
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
