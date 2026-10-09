package services

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDebeRecibirLead(t *testing.T) {
	casos := []struct {
		rol, usuarioAsesor, leadAsesor string
		esperado                       bool
	}{
		{"admin", "", "", true},
		{"Marketing", "", "a1", true},
		{"ventas", "", "", true},
		{"asesor", "a1", "a1", true},
		{"asesor", "A1", "a1", true},
		{"asesor", "a1", "a2", false},
		{"asesor", "a1", "", false},
		{"asesor", "", "", false},
		{"otro", "", "", false},
	}

	for _, caso := range casos {
		if got := DebeRecibirLead(caso.rol, caso.usuarioAsesor, caso.leadAsesor); got != caso.esperado {
			t.Errorf("DebeRecibirLead(%q, %q, %q) = %v", caso.rol, caso.usuarioAsesor, caso.leadAsesor, got)
		}
	}
}

func TestEsperaReintento(t *testing.T) {
	esperados := map[int]time.Duration{
		0: 30 * time.Second,
		1: 30 * time.Second,
		2: 2 * time.Minute,
		3: 10 * time.Minute,
		4: 30 * time.Minute,
		5: 60 * time.Minute,
		9: 60 * time.Minute,
	}

	for attempts, esperado := range esperados {
		if got := EsperaReintento(attempts); got != esperado {
			t.Errorf("EsperaReintento(%d) = %v, esperado %v", attempts, got, esperado)
		}
	}
}

func TestClasificarEnvio(t *testing.T) {
	errRed := errors.New("timeout")

	casos := []struct {
		nombre     string
		status     int
		err        error
		attempts   int
		estado     string
		desactivar bool
		reintento  time.Duration
	}{
		{"ok 201", 201, nil, 1, "sent", false, 0},
		{"gone", 410, nil, 1, "failed", true, 0},
		{"not found", 404, nil, 1, "failed", true, 0},
		{"red", 0, errRed, 1, "pending", false, 30 * time.Second},
		{"429", 429, nil, 2, "pending", false, 2 * time.Minute},
		{"503", 503, nil, 4, "pending", false, 30 * time.Minute},
		{"agotado", 500, nil, 5, "failed", false, 0},
		{"400", 400, nil, 1, "failed", false, 0},
		{"403", 403, nil, 1, "failed", false, 0},
	}

	for _, caso := range casos {
		got := ClasificarEnvio(caso.status, caso.err, caso.attempts)

		if got.Estado != caso.estado ||
			got.Desactivar != caso.desactivar ||
			got.Reintentar != caso.reintento {
			t.Errorf("%s: %+v", caso.nombre, got)
		}
	}
}

func TestArmarPayloadLead(t *testing.T) {
	evento, payload := armarPayloadLead(LeadWebNotificacion{
		LeadID:       "abc",
		Nombre:       "Olinda Matamoros",
		Proyecto:     "Las Colinas de Moro",
		Interes:      "Lote - Las Colinas de Moro",
		AccionCRM:    "creado",
		AsesorNombre: "Alicia",
	})

	if evento != EventoLeadCreado ||
		payload.Title != "Nuevo lead · Las Colinas de Moro" ||
		!strings.Contains(payload.Body, "Asesor: Alicia") ||
		payload.URL != "/leads/abc" ||
		payload.Tag != "lead-abc" {
		t.Errorf("payload inesperado: %s %+v", evento, payload)
	}

	evento, payload = armarPayloadLead(LeadWebNotificacion{
		LeadID:    "abc",
		AccionCRM: "actualizado",
	})

	if evento != EventoLeadRepetido ||
		!strings.HasPrefix(payload.Title, "Lead repetido") ||
		!strings.Contains(payload.Body, "Sin asesor asignado") {
		t.Errorf("payload repetido inesperado: %s %+v", evento, payload)
	}
}
