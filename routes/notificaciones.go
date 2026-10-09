package routes

import (
	"ancosur-api/controller"
	middleware "ancosur-api/middlewares"
	"ancosur-api/services"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificacionRoutes struct {
	DB *pgxpool.Pool
}

type SuscripcionPushRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	DeviceLabel string `json:"device_label"`
	UserAgent   string `json:"user_agent"`
}

type SuscripcionPushResponse struct {
	ID          string     `json:"id"`
	Channel     string     `json:"channel"`
	DeviceLabel string     `json:"device_label"`
	UserAgent   string     `json:"user_agent"`
	IsActive    bool       `json:"is_active"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func RutasNotificaciones(
	api *gin.RouterGroup,
	db *pgxpool.Pool,
) {
	notificacionRoutes := &NotificacionRoutes{
		DB: db,
	}

	// Público: el navegador la necesita para suscribirse
	api.GET(
		"/notifications/vapid-public-key",
		notificacionRoutes.GetVAPIDPublicKey,
	)

	// Protegido: dashboard
	protected := api.Group("/notifications")
	protected.Use(middleware.AuthMiddleware())

	protected.POST("/subscriptions", notificacionRoutes.CreateSubscription)
	protected.GET("/subscriptions", notificacionRoutes.GetSubscriptions)
	protected.DELETE("/subscriptions/:id", notificacionRoutes.DeleteSubscription)
	protected.POST("/test", notificacionRoutes.SendTest)
}

func (h *NotificacionRoutes) GetVAPIDPublicKey(
	c *gin.Context,
) {
	key := strings.TrimSpace(
		os.Getenv("VAPID_PUBLIC_KEY"),
	)

	if key == "" {
		controller.Error(
			c,
			http.StatusServiceUnavailable,
			"notifications.not_configured",
			"Las notificaciones push no están configuradas.",
			nil,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"key": key,
		},
	)
}

/*
Registra el dispositivo del usuario autenticado.
El user_id sale del JWT, nunca del body.
Si el usuario supera 5 dispositivos activos,
se desactivan los más antiguos.
*/
func (h *NotificacionRoutes) CreateSubscription(
	c *gin.Context,
) {
	userID := strings.TrimSpace(c.GetString("user_id"))

	var body SuscripcionPushRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		controller.Error(
			c,
			http.StatusBadRequest,
			"notifications.invalid_body",
			"Los datos de la suscripción no son válidos.",
			nil,
		)
		return
	}

	body.Endpoint = strings.TrimSpace(body.Endpoint)
	body.Keys.P256dh = strings.TrimSpace(body.Keys.P256dh)
	body.Keys.Auth = strings.TrimSpace(body.Keys.Auth)

	if !strings.HasPrefix(body.Endpoint, "https://") ||
		body.Keys.P256dh == "" ||
		body.Keys.Auth == "" {
		controller.Error(
			c,
			http.StatusBadRequest,
			"notifications.invalid_subscription",
			"La suscripción debe incluir endpoint HTTPS y claves p256dh/auth.",
			nil,
		)
		return
	}

	ctx := c.Request.Context()

	tx, err := h.DB.Begin(ctx)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"notifications.subscribe_error",
			"No se pudo registrar el dispositivo.",
			nil,
		)
		return
	}

	defer tx.Rollback(ctx)

	var subscriptionID string

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO notify.push_subscriptions (
				user_id,
				endpoint,
				p256dh,
				auth,
				user_agent,
				device_label
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				NULLIF($5, ''),
				NULLIF($6, '')
			)
			ON CONFLICT (endpoint) DO UPDATE
			SET
				user_id = EXCLUDED.user_id,
				p256dh = EXCLUDED.p256dh,
				auth = EXCLUDED.auth,
				user_agent = EXCLUDED.user_agent,
				device_label = EXCLUDED.device_label,
				is_active = TRUE,
				updated_at = NOW()
			RETURNING id::text
		`,
		userID,
		body.Endpoint,
		body.Keys.P256dh,
		body.Keys.Auth,
		strings.TrimSpace(body.UserAgent),
		strings.TrimSpace(body.DeviceLabel),
	).Scan(&subscriptionID)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"notifications.subscribe_error",
			"No se pudo registrar el dispositivo.",
			nil,
		)
		return
	}

	_, err = tx.Exec(
		ctx,
		`
			UPDATE notify.push_subscriptions
			SET
				is_active = FALSE,
				updated_at = NOW()
			WHERE user_id = $1::uuid
				AND is_active
				AND id NOT IN (
					SELECT id
					FROM notify.push_subscriptions
					WHERE user_id = $1::uuid
						AND is_active
					ORDER BY updated_at DESC
					LIMIT $2
				)
		`,
		userID,
		services.MaxDispositivosUsuario,
	)

	if err == nil {
		err = tx.Commit(ctx)
	}

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"notifications.subscribe_error",
			"No se pudo registrar el dispositivo.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusCreated,
		"notifications.subscribed",
		"Dispositivo registrado para notificaciones.",
		gin.H{
			"id": subscriptionID,
		},
	)
}

func (h *NotificacionRoutes) GetSubscriptions(
	c *gin.Context,
) {
	userID := strings.TrimSpace(c.GetString("user_id"))

	rows, err := h.DB.Query(
		c.Request.Context(),
		`
			SELECT
				id::text,
				channel,
				COALESCE(device_label, ''),
				COALESCE(user_agent, ''),
				is_active,
				last_used_at,
				created_at
			FROM notify.push_subscriptions
			WHERE user_id = $1::uuid
				AND is_active
			ORDER BY updated_at DESC
		`,
		userID,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"notifications.list_error",
			"No se pudieron obtener los dispositivos.",
			nil,
		)
		return
	}

	defer rows.Close()

	suscripciones := []SuscripcionPushResponse{}

	for rows.Next() {
		var item SuscripcionPushResponse

		if err := rows.Scan(
			&item.ID,
			&item.Channel,
			&item.DeviceLabel,
			&item.UserAgent,
			&item.IsActive,
			&item.LastUsedAt,
			&item.CreatedAt,
		); err != nil {
			controller.Error(
				c,
				http.StatusInternalServerError,
				"notifications.list_error",
				"No se pudieron obtener los dispositivos.",
				nil,
			)
			return
		}

		suscripciones = append(suscripciones, item)
	}

	controller.Success(
		c,
		http.StatusOK,
		"notifications.list",
		"Dispositivos registrados.",
		suscripciones,
	)
}

/*
Desactiva el dispositivo. Solo el dueño o un admin.
*/
func (h *NotificacionRoutes) DeleteSubscription(
	c *gin.Context,
) {
	id := strings.TrimSpace(c.Param("id"))
	userID := strings.TrimSpace(c.GetString("user_id"))
	rol := strings.ToLower(strings.TrimSpace(c.GetString("user_rol")))

	commandTag, err := h.DB.Exec(
		c.Request.Context(),
		`
			UPDATE notify.push_subscriptions
			SET
				is_active = FALSE,
				updated_at = NOW()
			WHERE id::text = $1
				AND (
					user_id = $2::uuid
					OR $3::text = 'admin'
				)
		`,
		id,
		userID,
		rol,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"notifications.delete_error",
			"No se pudo eliminar el dispositivo.",
			nil,
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"notifications.not_found",
			"Dispositivo no encontrado.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusOK,
		"notifications.deleted",
		"Dispositivo eliminado.",
		gin.H{
			"id": id,
		},
	)
}

func (h *NotificacionRoutes) SendTest(
	c *gin.Context,
) {
	userID := strings.TrimSpace(c.GetString("user_id"))

	encoladas, err := services.EncolarPrueba(
		c.Request.Context(),
		h.DB,
		userID,
	)

	if err != nil {
		controller.Error(
			c,
			http.StatusInternalServerError,
			"notifications.test_error",
			"No se pudo enviar la prueba.",
			nil,
		)
		return
	}

	if encoladas == 0 {
		controller.Error(
			c,
			http.StatusNotFound,
			"notifications.no_devices",
			"No tienes dispositivos con notificaciones activadas.",
			nil,
		)
		return
	}

	controller.Success(
		c,
		http.StatusAccepted,
		"notifications.test_queued",
		"Notificación de prueba en camino.",
		gin.H{
			"dispositivos": encoladas,
		},
	)
}
