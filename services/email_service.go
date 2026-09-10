package services

import (
	"os"
	"strings"
	"sync"
	"time"

	"ancosur-api/email"
)

type EmailService struct {
	SMTP            *email.SMTPClient
	InternalAddress string
}

func NewEmailService() *EmailService {
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	from := strings.TrimSpace(os.Getenv("EMAIL_FROM"))
	if from == "" {
		from = username
	}

	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}

	return &EmailService{
		SMTP: email.NewSMTPClient(
			os.Getenv("SMTP_HOST"),
			port,
			username,
			os.Getenv("SMTP_PASSWORD"),
			from,
			os.Getenv("EMAIL_FROM_NAME"),
		),
		InternalAddress: strings.TrimSpace(os.Getenv("RECLAMOS_EMAIL_INTERNO")),
	}
}

func (s *EmailService) InternalRecipient() string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.InternalAddress)
}

type EmailSendResult struct {
	Success   bool   `json:"success"`
	Status    string `json:"status"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SendReclamoEmails envía la constancia al consumidor y la copia interna.
// El primer resultado corresponde al consumidor y el segundo al correo interno.
func (s *EmailService) SendReclamoEmails(
	data email.ReclamoTemplateData,
	consumerRecipient string,
	attachments []email.Attachment,
) (EmailSendResult, EmailSendResult) {
	consumer := EmailSendResult{Status: "omitido"}
	internal := EmailSendResult{Status: "omitido"}

	if s == nil || s.SMTP == nil {
		err := "Servicio SMTP no configurado."
		consumer = EmailSendResult{Success: false, Status: "fallido", Error: err}
		internal = EmailSendResult{Success: false, Status: "fallido", Error: err}
		return consumer, internal
	}

	consumerRecipient = strings.TrimSpace(consumerRecipient)
	internalRecipient := s.InternalRecipient()

	// Los dos correos son independientes. Ejecutarlos concurrentemente evita
	// que el segundo espere a que termine la conexión SMTP del primero.
	// Esto reduce el tiempo total de procesamiento del envío.
	var wg sync.WaitGroup

	if consumerRecipient != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consumer = s.send(data, consumerRecipient, false, attachments)
		}()
	}

	if internalRecipient != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			internal = s.send(data, internalRecipient, true, attachments)
		}()
	} else {
		internal = EmailSendResult{
			Success: false,
			Status:  "fallido",
			Error:   "RECLAMOS_EMAIL_INTERNO no está configurado.",
		}
	}

	wg.Wait()
	return consumer, internal
}

func (s *EmailService) send(
	data email.ReclamoTemplateData,
	to string,
	internal bool,
	attachments []email.Attachment,
) EmailSendResult {
	if err := s.SMTP.Validate(); err != nil {
		return EmailSendResult{Success: false, Status: "fallido", Error: err.Error()}
	}

	subject, htmlBody, textBody := email.BuildReclamoEmail(data, internal)

	replyTo := ""
	if internal {
		replyTo = strings.TrimSpace(data.Email)
	}

	messageID, err := s.SMTP.Send(
		to,
		replyTo,
		subject,
		htmlBody,
		textBody,
		attachments,
	)
	if err != nil {
		return EmailSendResult{Success: false, Status: "fallido", Error: err.Error()}
	}

	return EmailSendResult{
		Success:   true,
		Status:    "enviado",
		MessageID: messageID,
	}
}

// ReclamoMailInput evita acoplar routes directamente a la estructura de las plantillas.
type ReclamoMailInput struct {
	Fecha                string
	Tipo                 string
	Nombres              string
	Apellidos            string
	Email                string
	Telefono             string
	TipoDocumento        string
	NumeroDocumento      string
	Departamento         string
	Provincia            string
	Distrito             string
	Domicilio            string
	EsMenor              bool
	TutorNombres         string
	TutorApellidos       string
	TutorTipoDocumento   string
	TutorNumeroDocumento string
	Establecimiento      string
	Proyecto             string
	TipoProyecto         string
	EstadoProyecto       string
	CiudadProyecto       string
	DireccionProyecto    string
	TipoBien             string
	Edificio             string
	Unidad               string
	NumeroOperacion      string
	Descripcion          string
	Detalle              string
	Pedido               string
	Canal                string
	CodigoRegistro       string
	NumeroCorrelativo    int64
	Anio                 int
}

func ReclamoMailDataFromRequest(in ReclamoMailInput) email.ReclamoTemplateData {
	fecha := strings.TrimSpace(in.Fecha)
	if fecha == "" {
		fecha = time.Now().Format("02/01/2006")
	}

	return email.ReclamoTemplateData{
		Fecha:                fecha,
		Tipo:                 in.Tipo,
		Nombres:              in.Nombres,
		Apellidos:            in.Apellidos,
		Email:                in.Email,
		Telefono:             in.Telefono,
		TipoDocumento:        in.TipoDocumento,
		NumeroDocumento:      in.NumeroDocumento,
		Departamento:         in.Departamento,
		Provincia:            in.Provincia,
		Distrito:             in.Distrito,
		Domicilio:            in.Domicilio,
		EsMenor:              in.EsMenor,
		TutorNombres:         in.TutorNombres,
		TutorApellidos:       in.TutorApellidos,
		TutorTipoDocumento:   in.TutorTipoDocumento,
		TutorNumeroDocumento: in.TutorNumeroDocumento,
		Establecimiento:      in.Establecimiento,
		Proyecto:             in.Proyecto,
		TipoProyecto:         in.TipoProyecto,
		EstadoProyecto:       in.EstadoProyecto,
		CiudadProyecto:       in.CiudadProyecto,
		DireccionProyecto:    in.DireccionProyecto,
		TipoBien:             in.TipoBien,
		Edificio:             in.Edificio,
		Unidad:               in.Unidad,
		NumeroOperacion:      in.NumeroOperacion,
		Descripcion:          in.Descripcion,
		Detalle:              in.Detalle,
		Pedido:               in.Pedido,
		Canal:                in.Canal,
		CodigoRegistro:       in.CodigoRegistro,
		NumeroCorrelativo:    in.NumeroCorrelativo,
		Anio:                 in.Anio,
	}
}
