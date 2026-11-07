package whatsmeow_service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var errVersionUnavailable = errors.New("version unavailable")

func TestParseWhatsAppWebVersionDirectClientRevision(t *testing.T) {
	version, err := parseWhatsAppWebVersion([]byte(`{"client_revision":123456}`))
	if err != nil {
		t.Fatalf("parseWhatsAppWebVersion() error = %v", err)
	}
	if *version != (clientVersion{Major: 2, Minor: 3000, Patch: 123456}) {
		t.Fatalf("version = %#v, want 2.3000.123456", version)
	}
}

func TestParseWhatsAppWebVersionEscapedClientRevision(t *testing.T) {
	version, err := parseWhatsAppWebVersion([]byte(`{\"client_revision\":654321}`))
	if err != nil {
		t.Fatalf("parseWhatsAppWebVersion() error = %v", err)
	}
	if version.Patch != 654321 {
		t.Fatalf("patch = %d, want 654321", version.Patch)
	}
}

func TestParseWhatsAppWebVersionRejectsInvalidBody(t *testing.T) {
	_, err := parseWhatsAppWebVersion([]byte("self.addEventListener('fetch', () => {})"))
	if err == nil {
		t.Fatal("parseWhatsAppWebVersion() error = nil, want invalid body error")
	}
}

func TestFetchWhatsAppWebVersionRejectsNon2xxResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"client_revision":999999}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := fetchWhatsAppWebVersionFrom(server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 503") {
		t.Fatalf("fetchWhatsAppWebVersion() error = %v, want status error", err)
	}
}

func TestResolveClientVersionPreservesManualOverride(t *testing.T) {
	called := false
	version, err := resolveClientVersion(2, 3000, 42, func() (*clientVersion, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("resolveClientVersion() error = %v", err)
	}
	if called {
		t.Fatal("automatic version fetch called for complete manual override")
	}
	if version != (clientVersion{Major: 2, Minor: 3000, Patch: 42}) {
		t.Fatalf("version = %#v, want manual override", version)
	}
}

func TestResolveClientVersionMakesFetchFailureExplicit(t *testing.T) {
	_, err := resolveClientVersion(0, 0, 0, func() (*clientVersion, error) {
		return nil, errVersionUnavailable
	})
	if err == nil || !strings.Contains(err.Error(), "automatic WhatsApp Web version resolution failed") {
		t.Fatalf("resolveClientVersion() error = %v, want explicit resolution error", err)
	}
}
