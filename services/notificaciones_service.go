package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
	NOTIFICACIONES PUSH DE LEADS

	Patrón outbox: el formulario solo inserta filas en
	notify.notification_outbox; el worker las envía en
	segundo plano. El formulario nunca espera al push.
*/

const (
	EventoLeadCreado       = "lead.created"
	EventoLeadRepetido     = "lead.updated"
	EventoPrueba           = "test"
	MaxDispositivosUsuario = 5
	maxIntentosEnvio       = 5
	tamanoLoteEnvio        = 50
	intervaloWorker        = 2 * time.Second
)

var esperasReintento = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
	60 * time.Minute,
}

/* ESQUEMA */

var esquemaNotificaciones = []string{
	`CREATE SCHEMA IF NOT EXISTS notify`,

	`CREATE TABLE IF NOT EXISTS notify.push_subscriptions (
		id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id       uuid NOT NULL REFERENCES usuarios_dashboard(id) ON DELETE CASCADE,
		channel       text NOT NULL DEFAULT 'webpush',
		endpoint      text NOT NULL UNIQUE,
		p256dh        text,
		auth          text,
		user_agent    text,
		device_label  text,
		is_active     boolean NOT NULL DEFAULT true,
		last_used_at  timestamptz,
		created_at    timestamptz NOT NULL DEFAULT now(),
		updated_at    timestamptz NOT NULL DEFAULT now()
	)`,

	`CREATE INDEX IF NOT EXISTS push_subscriptions_user_activas_idx
		ON notify.push_subscriptions (user_id)
		WHERE is_active`,

	`CREATE TABLE IF NOT EXISTS notify.notification_outbox (
		id              bigserial PRIMARY KEY,
		event_type      text NOT NULL,
		lead_id         uuid REFERENCES leads_web(id) ON DELETE CASCADE,
		subscription_id uuid NOT NULL REFERENCES notify.push_subscriptions(id) ON DELETE CASCADE,
		payload         jsonb NOT NULL,
		status          text NOT NULL DEFAULT 'pending',
		attempts        int NOT NULL DEFAULT 0,
		next_attempt_at timestamptz NOT NULL DEFAULT now(),
		status_code     int,
		last_error      text,
		sent_at         timestamptz,
		created_at      timestamptz NOT NULL DEFAULT now(),
		UNIQUE (event_type, lead_id, subscription_id)
	)`,

	`CREATE INDEX IF NOT EXISTS notification_outbox_pendientes_idx
		ON notify.notification_outbox (next_attempt_at)
		WHERE status = 'pending'`,
}

/*
Crea el esquema notify si no existe.
El proyecto no usa herramienta de migraciones.
*/
func AsegurarEsquemaNotificaciones(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	for _, sentencia := range esquemaNotificaciones {
		if _, err := db.Exec(ctx, sentencia); err != nil {
			return fmt.Errorf(
				"esquema notify: %w",
				err,
			)
		}
	}

	return nil
}

/* DESTINATARIOS */

/*
admin, marketing y ventas reciben todos los leads.
asesor solo recibe los leads asignados a él.
*/
func DebeRecibirLead(
	rol string,
	usuarioAsesorID string,
	leadAsesorID string,
) bool {
	switch strings.ToLower(strings.TrimSpace(rol)) {
	case "admin", "marketing", "ventas":
		return true

	case "asesor":
		usuarioAsesorID = strings.TrimSpace(usuarioAsesorID)

		return usuarioAsesorID != "" &&
			strings.EqualFold(
				usuarioAsesorID,
				strings.TrimSpace(leadAsesorID),
			)
	}

	return false
}

/* ENCOLAR */

type LeadWebNotificacion struct {
	LeadID       string
	Nombre       string
	Proyecto     string
	Interes      string
	AccionCRM    string
	AsesorID     string
	AsesorNombre string
}

type PayloadPush struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

/*
Se llama después de la respuesta del CRM,
cuando ya se conoce el asesor asignado.
*/
func EncolarLeadWeb(
	ctx context.Context,
	db *pgxpool.Pool,
	lead LeadWebNotificacion,
) (int64, error) {
	evento, payload :=
		armarPayloadLead(
			lead,
		)

	payloadJSON, err :=
		json.Marshal(
			payload,
		)

	if err != nil {
		return 0, err
	}

	rows, err :=
		db.Query(
			ctx,
			`
			SELECT
				s.id::text,
				COALESCE(u.rol, ''),
				COALESCE(u.asesor_id::text, '')
			FROM notify.push_subscriptions AS s
			INNER JOIN usuarios_dashboard AS u
				ON u.id = s.user_id
			WHERE s.is_active
				AND COALESCE(u.activo, TRUE) = TRUE
			`,
		)

	if err != nil {
		return 0, err
	}

	var suscripciones []string

	for rows.Next() {
		var subscriptionID, rol, usuarioAsesorID string

		if err := rows.Scan(
			&subscriptionID,
			&rol,
			&usuarioAsesorID,
		); err != nil {
			rows.Close()
			return 0, err
		}

		if DebeRecibirLead(
			rol,
			usuarioAsesorID,
			lead.AsesorID,
		) {
			suscripciones =
				append(
					suscripciones,
					subscriptionID,
				)
		}
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(suscripciones) == 0 {
		return 0, nil
	}

	tag, err :=
		db.Exec(
			ctx,
			`
			INSERT INTO notify.notification_outbox (
				event_type,
				lead_id,
				subscription_id,
				payload
			)
			SELECT
				$1::text,
				$2::uuid,
				subscription_id::uuid,
				$4::jsonb
			FROM UNNEST($3::text[]) AS subscription_id
			ON CONFLICT DO NOTHING
			`,
			evento,
			lead.LeadID,
			suscripciones,
			string(payloadJSON),
		)

	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}

func armarPayloadLead(
	lead LeadWebNotificacion,
) (string, PayloadPush) {
	evento := EventoLeadCreado
	titulo := "Nuevo lead"

	if strings.Contains(
		strings.ToLower(lead.AccionCRM),
		"actualiz",
	) {
		evento = EventoLeadRepetido
		titulo = "Lead repetido"
	}

	if proyecto := strings.TrimSpace(lead.Proyecto); proyecto != "" {
		titulo += " · " + proyecto
	}

	var partes []string

	if nombre := strings.TrimSpace(lead.Nombre); nombre != "" {
		partes = append(partes, nombre)
	}

	if interes := strings.TrimSpace(lead.Interes); interes != "" {
		partes = append(partes, interes)
	}

	if asesor := strings.TrimSpace(lead.AsesorNombre); asesor != "" {
		partes = append(partes, "Asesor: "+asesor)
	} else {
		partes = append(partes, "Sin asesor asignado")
	}

	return evento, PayloadPush{
		Title: titulo,
		Body:  strings.Join(partes, " · "),
		URL:   URLLeadPush(lead.LeadID),
		Tag:   "lead-" + lead.LeadID,
	}
}

/*
PUSH_LEAD_URL permite ajustar la ruta del dashboard.
Ejemplo: /leads-web/{id}
*/
func URLLeadPush(
	leadID string,
) string {
	plantilla :=
		strings.TrimSpace(
			os.Getenv("PUSH_LEAD_URL"),
		)

	if plantilla == "" {
		plantilla = "/leads/{id}"
	}

	return strings.ReplaceAll(
		plantilla,
		"{id}",
		leadID,
	)
}

/*
Encola una notificación de prueba para cada
dispositivo activo del usuario.
*/
func EncolarPrueba(
	ctx context.Context,
	db *pgxpool.Pool,
	userID string,
) (int64, error) {
	payload, err :=
		json.Marshal(
			PayloadPush{
				Title: "Notificaciones activadas",
				Body:  "Este dispositivo recibirá los avisos de leads nuevos.",
				URL:   "/",
				Tag:   "test",
			},
		)

	if err != nil {
		return 0, err
	}

	tag, err :=
		db.Exec(
			ctx,
			`
			INSERT INTO notify.notification_outbox (
				event_type,
				subscription_id,
				payload
			)
			SELECT
				$2::text,
				s.id,
				$3::jsonb
			FROM notify.push_subscriptions AS s
			WHERE s.user_id = $1::uuid
				AND s.is_active
			`,
			userID,
			EventoPrueba,
			string(payload),
		)

	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}

/* CANALES */

type SuscripcionPush struct {
	ID       string
	Endpoint string
	P256dh   string
	Auth     string
}

type ResultadoEnvio struct {
	StatusCode int
}

/*
Cada canal (Web Push, Telegram, FCM) implementa
esta interfaz sin tocar el flujo de leads.
*/
type Notifier interface {
	Send(
		ctx context.Context,
		sub SuscripcionPush,
		payload []byte,
	) (ResultadoEnvio, error)
}

type WebPushNotifier struct {
	Subject    string
	PublicKey  string
	PrivateKey string
}

/*
Requiere VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY
y VAPID_SUBJECT (mailto:...).
*/
func NewWebPushNotifierDesdeEnv() (*WebPushNotifier, bool) {
	notifier :=
		&WebPushNotifier{
			Subject:    normalizarSubjectVAPID(os.Getenv("VAPID_SUBJECT")),
			PublicKey:  LimpiarClaveVAPID(os.Getenv("VAPID_PUBLIC_KEY"), "VAPID_PUBLIC_KEY"),
			PrivateKey: LimpiarClaveVAPID(os.Getenv("VAPID_PRIVATE_KEY"), "VAPID_PRIVATE_KEY"),
		}

	if notifier.Subject == "" ||
		notifier.PublicKey == "" ||
		notifier.PrivateKey == "" {
		return nil, false
	}

	if err := validarClaveBase64(notifier.PublicKey, 65); err != nil {
		log.Println("VAPID_PUBLIC_KEY inválida:", err)
		return nil, false
	}

	// La librería puede generar claves privadas de 31 bytes (cero inicial)
	if err := validarClaveBase64(notifier.PrivateKey, 32, 31); err != nil {
		log.Println("VAPID_PRIVATE_KEY inválida:", err)
		return nil, false
	}

	return notifier, true
}

/*
Tolera errores comunes al pegar la variable en
Railway: espacios, comillas o "NOMBRE=" delante.
*/
func LimpiarClaveVAPID(
	valor string,
	nombre string,
) string {
	valor = strings.TrimSpace(valor)
	valor = strings.TrimPrefix(valor, nombre+"=")
	valor = strings.Trim(valor, "\"' \t\r\n")

	return valor
}

func decodificarBase64Flexible(
	valor string,
) ([]byte, error) {
	valor = strings.TrimRight(valor, "=")
	valor = strings.NewReplacer("+", "-", "/", "_").Replace(valor)

	return base64.RawURLEncoding.DecodeString(valor)
}

func validarClaveBase64(
	valor string,
	bytesEsperados ...int,
) error {
	decodificada, err := decodificarBase64Flexible(valor)

	if err != nil {
		return err
	}

	for _, esperado := range bytesEsperados {
		if len(decodificada) == esperado {
			return nil
		}
	}

	return fmt.Errorf(
		"mide %d bytes, se esperaban %v",
		len(decodificada),
		bytesEsperados,
	)
}

/*
webpush-go agrega "mailto:" por su cuenta salvo
que sea una URL https. Si la variable ya lo trae,
quedaría "mailto:mailto:..." y el servicio de push
rechaza la firma (403).
*/
func normalizarSubjectVAPID(
	subject string,
) string {
	subject = strings.TrimSpace(subject)

	if len(subject) >= len("mailto:") &&
		strings.EqualFold(subject[:len("mailto:")], "mailto:") {
		subject = strings.TrimSpace(subject[len("mailto:"):])
	}

	return subject
}

func (n *WebPushNotifier) Send(
	ctx context.Context,
	sub SuscripcionPush,
	payload []byte,
) (ResultadoEnvio, error) {
	if _, err := decodificarBase64Flexible(sub.P256dh); err != nil {
		return ResultadoEnvio{}, fmt.Errorf("p256dh de la suscripción inválida: %w", err)
	}

	if _, err := decodificarBase64Flexible(sub.Auth); err != nil {
		return ResultadoEnvio{}, fmt.Errorf("auth de la suscripción inválida: %w", err)
	}

	resp, err :=
		webpush.SendNotificationWithContext(
			ctx,
			payload,
			&webpush.Subscription{
				Endpoint: sub.Endpoint,
				Keys: webpush.Keys{
					P256dh: sub.P256dh,
					Auth:   sub.Auth,
				},
			},
			&webpush.Options{
				Subscriber:      n.Subject,
				VAPIDPublicKey:  n.PublicKey,
				VAPIDPrivateKey: n.PrivateKey,
				TTL:             3600,
				Urgency:         webpush.UrgencyHigh,
			},
		)

	if resp == nil {
		return ResultadoEnvio{}, err
	}

	defer resp.Body.Close()

	return ResultadoEnvio{
		StatusCode: resp.StatusCode,
	}, err
}

/* CLASIFICAR RESULTADO */

type DecisionEnvio struct {
	Estado     string // sent | pending | failed
	Reintentar time.Duration
	Desactivar bool
}

/*
attempts ya incluye el intento actual (1 = primero).
*/
func EsperaReintento(
	attempts int,
) time.Duration {
	if attempts < 1 {
		attempts = 1
	}

	if attempts > len(esperasReintento) {
		attempts = len(esperasReintento)
	}

	return esperasReintento[attempts-1]
}

func ClasificarEnvio(
	statusCode int,
	err error,
	attempts int,
) DecisionEnvio {
	switch {
	case err == nil &&
		statusCode >= 200 &&
		statusCode < 300:
		return DecisionEnvio{Estado: "sent"}

	case statusCode == http.StatusNotFound ||
		statusCode == http.StatusGone:
		return DecisionEnvio{
			Estado:     "failed",
			Desactivar: true,
		}

	case statusCode == 0 ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= 500:
		if attempts >= maxIntentosEnvio {
			return DecisionEnvio{Estado: "failed"}
		}

		return DecisionEnvio{
			Estado:     "pending",
			Reintentar: EsperaReintento(attempts),
		}
	}

	return DecisionEnvio{Estado: "failed"}
}

/* WORKER */

type trabajoPush struct {
	OutboxID    int64
	Payload     []byte
	Attempts    int
	Activa      bool
	Suscripcion SuscripcionPush
}

/*
Procesa la cola cada 2 s hasta que ctx se cancela.
El lote en curso termina aunque ctx se cancele.
*/
func IniciarWorkerNotificaciones(
	ctx context.Context,
	db *pgxpool.Pool,
	notifier Notifier,
) {
	log.Println("Worker de notificaciones push iniciado.")

	ticker := time.NewTicker(intervaloWorker)
	defer ticker.Stop()

	for {
		for {
			procesados, err :=
				procesarLotePush(
					db,
					notifier,
				)

			if err != nil {
				log.Println("PUSH worker:", err)
				break
			}

			if procesados < tamanoLoteEnvio ||
				ctx.Err() != nil {
				break
			}
		}

		select {
		case <-ctx.Done():
			log.Println("Worker de notificaciones push detenido.")
			return

		case <-ticker.C:
		}
	}
}

func procesarLotePush(
	db *pgxpool.Pool,
	notifier Notifier,
) (int, error) {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			60*time.Second,
		)

	defer cancel()

	trabajos, err :=
		tomarLotePush(
			ctx,
			db,
		)

	if err != nil {
		return 0, err
	}

	for _, trabajo := range trabajos {
		procesarTrabajoPush(
			ctx,
			db,
			notifier,
			trabajo,
		)
	}

	return len(trabajos), nil
}

/*
Lease: marca el lote con next_attempt_at + 5 min
para que otra réplica no lo tome mientras se envía.
*/
func tomarLotePush(
	ctx context.Context,
	db *pgxpool.Pool,
) ([]trabajoPush, error) {
	rows, err :=
		db.Query(
			ctx,
			`
			WITH picked AS (
				SELECT id
				FROM notify.notification_outbox
				WHERE status = 'pending'
					AND next_attempt_at <= now()
				ORDER BY id
				LIMIT $1
				FOR UPDATE SKIP LOCKED
			)
			UPDATE notify.notification_outbox AS o
			SET
				attempts = o.attempts + 1,
				next_attempt_at = now() + interval '5 minutes'
			FROM picked, notify.push_subscriptions AS s
			WHERE o.id = picked.id
				AND s.id = o.subscription_id
			RETURNING
				o.id,
				o.payload::text,
				o.attempts,
				s.is_active,
				s.id::text,
				s.endpoint,
				COALESCE(s.p256dh, ''),
				COALESCE(s.auth, '')
			`,
			tamanoLoteEnvio,
		)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var trabajos []trabajoPush

	for rows.Next() {
		var trabajo trabajoPush
		var payload string

		if err := rows.Scan(
			&trabajo.OutboxID,
			&payload,
			&trabajo.Attempts,
			&trabajo.Activa,
			&trabajo.Suscripcion.ID,
			&trabajo.Suscripcion.Endpoint,
			&trabajo.Suscripcion.P256dh,
			&trabajo.Suscripcion.Auth,
		); err != nil {
			return nil, err
		}

		trabajo.Payload = []byte(payload)
		trabajos = append(trabajos, trabajo)
	}

	return trabajos, rows.Err()
}

func procesarTrabajoPush(
	ctx context.Context,
	db *pgxpool.Pool,
	notifier Notifier,
	trabajo trabajoPush,
) {
	if !trabajo.Activa {
		marcarOutbox(
			ctx,
			db,
			trabajo.OutboxID,
			DecisionEnvio{Estado: "failed"},
			0,
			"suscripción inactiva",
		)

		return
	}

	inicio := time.Now()

	envioCtx, cancel :=
		context.WithTimeout(
			ctx,
			15*time.Second,
		)

	resultado, err :=
		notifier.Send(
			envioCtx,
			trabajo.Suscripcion,
			trabajo.Payload,
		)

	cancel()

	decision :=
		ClasificarEnvio(
			resultado.StatusCode,
			err,
			trabajo.Attempts,
		)

	ultimoError := ""

	if err != nil {
		ultimoError = err.Error()
	} else if decision.Estado != "sent" {
		ultimoError = fmt.Sprintf(
			"HTTP %d",
			resultado.StatusCode,
		)
	}

	log.Printf(
		"PUSH outbox_id=%d subscription_id=%s status_code=%d estado=%s intento=%d latencia_ms=%d error=%q",
		trabajo.OutboxID,
		trabajo.Suscripcion.ID,
		resultado.StatusCode,
		decision.Estado,
		trabajo.Attempts,
		time.Since(inicio).Milliseconds(),
		ultimoError,
	)

	marcarOutbox(
		ctx,
		db,
		trabajo.OutboxID,
		decision,
		resultado.StatusCode,
		ultimoError,
	)

	if decision.Estado == "sent" {
		_, _ = db.Exec(
			ctx,
			`
			UPDATE notify.push_subscriptions
			SET last_used_at = now()
			WHERE id = $1::uuid
			`,
			trabajo.Suscripcion.ID,
		)
	}

	if decision.Desactivar {
		_, _ = db.Exec(
			ctx,
			`
			UPDATE notify.push_subscriptions
			SET
				is_active = false,
				updated_at = now()
			WHERE id = $1::uuid
			`,
			trabajo.Suscripcion.ID,
		)
	}
}

func marcarOutbox(
	ctx context.Context,
	db *pgxpool.Pool,
	outboxID int64,
	decision DecisionEnvio,
	statusCode int,
	ultimoError string,
) {
	_, err :=
		db.Exec(
			ctx,
			`
			UPDATE notify.notification_outbox
			SET
				status = $2::text,
				status_code = NULLIF($3::int, 0),
				last_error = NULLIF($4::text, ''),
				next_attempt_at = now() + make_interval(secs => $5::float8),
				sent_at = CASE
					WHEN $2::text = 'sent' THEN now()
					ELSE sent_at
				END
			WHERE id = $1
			`,
			outboxID,
			decision.Estado,
			statusCode,
			ultimoError,
			decision.Reintentar.Seconds(),
		)

	if err != nil {
		log.Println(
			"PUSH: no se pudo actualizar outbox",
			outboxID,
			err,
		)
	}
}
