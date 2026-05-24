package steps

import (
	"net"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/services"
)

func TestLocalPortModelRejectsBusyPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	busyPort, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	m := NewLocalPortModel(services.Target{Host: "db.internal", Port: 5432})
	m.input.SetValue(strconv.Itoa(busyPort))

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	portModel := model.(*LocalPortModel)
	if cmd != nil {
		t.Fatal("expected no advance command for busy port")
	}
	if !strings.Contains(portModel.err, "already in use") {
		t.Fatalf("expected in-use error, got %q", portModel.err)
	}
}
