package email

import (
	"fmt"
	"html"
	"strings"
)

// ReclamoTemplateData contiene la información utilizada para generar
// los correos del Libro de Reclamaciones.
type ReclamoTemplateData struct {
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

// BuildReclamoEmail genera el asunto, HTML y texto plano del correo.
// No depende de imágenes externas ni de un logo remoto, por lo que funciona
// correctamente en Gmail, Outlook y clientes que bloquean imágenes.
func BuildReclamoEmail(data ReclamoTemplateData, interno bool) (string, string, string) {
	tipo := displayValue(data.Tipo)
	if strings.EqualFold(tipo, "reclamo") {
		tipo = "Reclamo"
	} else if strings.EqualFold(tipo, "queja") {
		tipo = "Queja"
	}

	codigo := displayValue(data.CodigoRegistro)
	if codigo == "No especificado" && data.NumeroCorrelativo > 0 {
		codigo = fmt.Sprintf("ANC-LR-%d-%06d", data.Anio, data.NumeroCorrelativo)
	}

	nombreCompleto := strings.TrimSpace(data.Nombres + " " + data.Apellidos)
	if nombreCompleto == "" {
		nombreCompleto = "No especificado"
	}

	asunto := fmt.Sprintf("Libro de Reclamaciones | %s | %s", tipo, codigo)
	if interno {
		asunto = fmt.Sprintf("[INTERNO] %s", asunto)
	}

	htmlBody := buildReclamoHTML(data, tipo, codigo, nombreCompleto, interno)
	textBody := buildReclamoText(data, tipo, codigo, nombreCompleto, interno)

	return asunto, htmlBody, textBody
}

func buildReclamoHTML(
	data ReclamoTemplateData,
	tipo string,
	codigo string,
	nombreCompleto string,
	interno bool,
) string {
	const green = "#00A74F"
	const black = "#111111"
	const darkGray = "#333333"
	const gray = "#6B7280"
	const light = "#F5F7F6"
	const border = "#E5E7EB"

	title := "Constancia de registro"
	intro := "Hemos registrado correctamente su información en nuestro Libro de Reclamaciones."
	if interno {
		title = "Nuevo registro recibido"
		intro = "Se ha recibido un nuevo registro en el Libro de Reclamaciones de ANCOSUR."
	}

	minorSection := ""
	if data.EsMenor {
		minorSection = sectionHTML(
			"Representante del menor",
			rowHTML("Nombres", data.TutorNombres)+
				rowHTML("Apellidos", data.TutorApellidos)+
				rowHTML("Documento", displayValue(data.TutorTipoDocumento)+" "+displayValue(data.TutorNumeroDocumento)),
		)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Libro de Reclamaciones - ANCOSUR</title>
</head>

<body style="margin:0;padding:0;background:%s;font-family:Arial,Helvetica,sans-serif;color:%s;">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="background:%s;">
<tr>
<td align="center" style="padding:28px 14px;">

<table role="presentation" width="680" cellspacing="0" cellpadding="0" border="0"
style="width:100%%;max-width:680px;background:#ffffff;border:1px solid %s;">

<!-- HEADER -->
<tr>
<td style="background:%s;padding:24px 30px;border-bottom:5px solid %s;">
	<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
	<tr>
	<td align="left">
		<div style="font-size:25px;font-weight:800;letter-spacing:1px;color:#ffffff;line-height:1;">
			ANCOSUR
		</div>
		<div style="margin-top:5px;font-size:10px;font-weight:700;letter-spacing:2.5px;color:#ffffff;">
			INMOBILIARIA
		</div>
	</td>
	<td align="right" valign="middle">
		<div style="font-size:10px;font-weight:700;letter-spacing:1.4px;color:#ffffff;text-transform:uppercase;">
			LIBRO DE RECLAMACIONES
		</div>
	</td>
	</tr>
	</table>
</td>
</tr>

<!-- INTRO -->
<tr>
<td style="padding:34px 30px 20px;">
	<div style="font-size:11px;font-weight:700;letter-spacing:1.5px;text-transform:uppercase;color:%s;">
		%s
	</div>
	<h1 style="margin:9px 0 12px;font-size:27px;line-height:1.2;color:%s;font-weight:800;">
		%s
	</h1>
	<p style="margin:0;font-size:14px;line-height:1.65;color:%s;">
		%s
	</p>
</td>
</tr>

<!-- IDENTIFICADOR -->
<tr>
<td style="padding:0 30px 22px;">
	<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0"
	style="background:%s;border-left:4px solid %s;">
	<tr>
	<td style="padding:18px 20px;">
		<div style="font-size:10px;color:%s;text-transform:uppercase;letter-spacing:1.2px;font-weight:700;">
			Código de registro
		</div>
		<div style="margin-top:5px;font-size:23px;color:%s;font-weight:800;letter-spacing:.5px;">
			%s
		</div>
		<div style="margin-top:8px;font-size:12px;color:%s;">
			%s · Fecha de registro: %s
		</div>
	</td>
	</tr>
	</table>
</td>
</tr>

<!-- DATOS DEL REGISTRO -->
%s

<!-- DATOS DEL CONSUMIDOR -->
%s

<!-- PROYECTO -->
%s

<!-- DETALLE -->
%s

<!-- FOOTER -->
<tr>
<td style="padding:8px 30px 30px;">
	<div style="height:1px;background:%s;"></div>
	<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="margin-top:22px;">
	<tr>
	<td>
		<div style="font-size:13px;font-weight:800;color:%s;">ANCOSUR S.A.C.</div>
		<div style="margin-top:5px;font-size:11px;line-height:1.6;color:%s;">
			Libro de Reclamaciones · Registro electrónico
		</div>
	</td>
	<td align="right" valign="top">
		<div style="font-size:10px;font-weight:700;color:%s;letter-spacing:.8px;">
			ANCOSUR
		</div>
		<div style="margin-top:4px;font-size:10px;color:%s;">
			%s
		</div>
	</td>
	</tr>
	</table>

	<p style="margin:20px 0 0;font-size:10px;line-height:1.6;color:%s;">
		Este correo es una comunicación automática relacionada con el registro realizado
		en el Libro de Reclamaciones. Conserve este mensaje como referencia.
	</p>
</td>
</tr>

</table>
</td>
</tr>
</table>
</body>
</html>`,
		light,
		black,
		light,
		border,
		black,
		green,
		green,
		title,
		black,
		title,
		darkGray,
		html.EscapeString(intro),
		light,
		green,
		gray,
		black,
		html.EscapeString(codigo),
		gray,
		html.EscapeString(tipo),
		html.EscapeString(displayValue(data.Fecha)),
		registroSectionHTML(data, green, black, darkGray, gray, border),
		consumerSectionHTML(data, nombreCompleto, green, black, darkGray, gray, border),
		projectSectionHTML(data, green, black, darkGray, gray, border),
		descriptionSectionHTML(data, green, black, darkGray, gray, border),
		border,
		black,
		gray,
		green,
		gray,
		html.EscapeString(codigo),
		gray,
	) + minorSection
}

func registroSectionHTML(data ReclamoTemplateData, green, black, darkGray, gray, border string) string {
	return sectionHTML(
		"Información del registro",
		rowHTML("Tipo", data.Tipo)+
			rowHTML("Canal", data.Canal)+
			rowHTML("Establecimiento", data.Establecimiento),
	)
}

func consumerSectionHTML(data ReclamoTemplateData, nombreCompleto, green, black, darkGray, gray, border string) string {
	return sectionHTML(
		"Datos del consumidor",
		rowHTML("Nombres y apellidos", nombreCompleto)+
			rowHTML("Documento", displayValue(data.TipoDocumento)+" "+displayValue(data.NumeroDocumento))+
			rowHTML("Correo electrónico", data.Email)+
			rowHTML("Teléfono", data.Telefono)+
			rowHTML("Domicilio", data.Domicilio)+
			rowHTML("Ubicación", joinNonEmpty(data.Departamento, data.Provincia, data.Distrito)),
	)
}

func projectSectionHTML(data ReclamoTemplateData, green, black, darkGray, gray, border string) string {
	return sectionHTML(
		"Producto, servicio o proyecto",
		rowHTML("Proyecto", data.Proyecto)+
			rowHTML("Tipo de proyecto", data.TipoProyecto)+
			rowHTML("Tipo de bien", data.TipoBien)+
			rowHTML("Estado del proyecto", data.EstadoProyecto)+
			rowHTML("Ciudad", data.CiudadProyecto)+
			rowHTML("Dirección", data.DireccionProyecto)+
			rowHTML("Edificio / torre / bloque", data.Edificio)+
			rowHTML("Unidad inmobiliaria", data.Unidad)+
			rowHTML("N.º de operación", data.NumeroOperacion)+
			rowHTML("Descripción", data.Descripcion),
	)
}

func descriptionSectionHTML(data ReclamoTemplateData, green, black, darkGray, gray, border string) string {
	return sectionHTML(
		"Contenido presentado",
		rowHTMLMultiline("Detalle de la reclamación", data.Detalle)+
			rowHTMLMultiline("Pedido concreto", data.Pedido),
	)
}

func sectionHTML(title, content string) string {
	return fmt.Sprintf(`
<tr>
<td style="padding:0 30px 24px;">
	<div style="font-size:12px;font-weight:800;letter-spacing:.7px;text-transform:uppercase;color:#111111;padding-bottom:10px;border-bottom:2px solid #00A74F;">
		%s
	</div>
	<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="margin-top:2px;">
	%s
	</table>
</td>
</tr>`, html.EscapeString(title), content)
}

func rowHTML(label, value string) string {
	return fmt.Sprintf(`
<tr>
<td width="36%%" valign="top" style="padding:9px 12px 9px 0;border-bottom:1px solid #E5E7EB;font-size:10px;color:#6B7280;font-weight:700;text-transform:uppercase;letter-spacing:.4px;">
	%s
</td>
<td valign="top" style="padding:9px 0;border-bottom:1px solid #E5E7EB;font-size:13px;line-height:1.5;color:#333333;">
	%s
</td>
</tr>`, html.EscapeString(label), html.EscapeString(displayValue(value)))
}

func rowHTMLMultiline(label, value string) string {
	v := html.EscapeString(displayValue(value))
	v = strings.ReplaceAll(v, "\r\n", "<br>")
	v = strings.ReplaceAll(v, "\n", "<br>")
	v = strings.ReplaceAll(v, "\r", "<br>")

	return fmt.Sprintf(`
<tr>
<td width="36%%" valign="top" style="padding:12px 12px 12px 0;border-bottom:1px solid #E5E7EB;font-size:10px;color:#6B7280;font-weight:700;text-transform:uppercase;letter-spacing:.4px;">
	%s
</td>
<td valign="top" style="padding:12px 0;border-bottom:1px solid #E5E7EB;font-size:13px;line-height:1.65;color:#333333;">
	%s
</td>
</tr>`, html.EscapeString(label), v)
}

func buildReclamoText(
	data ReclamoTemplateData,
	tipo string,
	codigo string,
	nombreCompleto string,
	interno bool,
) string {
	var b strings.Builder

	if interno {
		b.WriteString("NUEVO REGISTRO - LIBRO DE RECLAMACIONES\n\n")
	} else {
		b.WriteString("CONSTANCIA DE REGISTRO - LIBRO DE RECLAMACIONES\n\n")
	}

	b.WriteString("ANCOSUR S.A.C.\n")
	fmt.Fprintf(&b, "Código de registro: %s\n", codigo)
	fmt.Fprintf(&b, "Tipo: %s\n", tipo)
	fmt.Fprintf(&b, "Fecha: %s\n\n", displayValue(data.Fecha))

	b.WriteString("DATOS DEL CONSUMIDOR\n")
	fmt.Fprintf(&b, "Nombres y apellidos: %s\n", nombreCompleto)
	fmt.Fprintf(
		&b,
		"Documento: %s %s\n",
		displayValue(data.TipoDocumento),
		displayValue(data.NumeroDocumento),
	)
	fmt.Fprintf(&b, "Correo: %s\n", displayValue(data.Email))
	fmt.Fprintf(&b, "Teléfono: %s\n", displayValue(data.Telefono))
	fmt.Fprintf(&b, "Domicilio: %s\n", displayValue(data.Domicilio))
	fmt.Fprintf(
		&b,
		"Ubicación: %s\n\n",
		joinNonEmpty(data.Departamento, data.Provincia, data.Distrito),
	)

	if data.EsMenor {
		b.WriteString("REPRESENTANTE DEL MENOR\n")
		fmt.Fprintf(
			&b,
			"Nombres y apellidos: %s\n",
			joinNonEmpty(data.TutorNombres, data.TutorApellidos),
		)
		fmt.Fprintf(
			&b,
			"Documento: %s %s\n\n",
			displayValue(data.TutorTipoDocumento),
			displayValue(data.TutorNumeroDocumento),
		)
	}

	b.WriteString("PROYECTO / PRODUCTO / SERVICIO\n")
	fmt.Fprintf(&b, "Proyecto: %s\n", displayValue(data.Proyecto))
	fmt.Fprintf(&b, "Tipo de proyecto: %s\n", displayValue(data.TipoProyecto))
	fmt.Fprintf(&b, "Tipo de bien: %s\n", displayValue(data.TipoBien))
	fmt.Fprintf(&b, "Estado: %s\n", displayValue(data.EstadoProyecto))
	fmt.Fprintf(&b, "Ciudad: %s\n", displayValue(data.CiudadProyecto))
	fmt.Fprintf(&b, "Dirección: %s\n", displayValue(data.DireccionProyecto))
	fmt.Fprintf(&b, "Edificio / torre / bloque: %s\n", displayValue(data.Edificio))
	fmt.Fprintf(&b, "Unidad: %s\n", displayValue(data.Unidad))
	fmt.Fprintf(&b, "N.º de operación: %s\n", displayValue(data.NumeroOperacion))
	fmt.Fprintf(&b, "Descripción: %s\n\n", displayValue(data.Descripcion))

	b.WriteString("DETALLE DE LA RECLAMACIÓN\n")
	fmt.Fprintf(&b, "%s\n\n", displayValue(data.Detalle))

	b.WriteString("PEDIDO CONCRETO\n")
	fmt.Fprintf(&b, "%s\n\n", displayValue(data.Pedido))

	b.WriteString(
		"Este correo es una comunicación automática del Libro de Reclamaciones de ANCOSUR.\n",
	)

	return b.String()
}

func displayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "No especificado"
	}
	return value
}

func joinNonEmpty(values ...string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			items = append(items, value)
		}
	}
	if len(items) == 0 {
		return "No especificado"
	}
	return strings.Join(items, ", ")
}
