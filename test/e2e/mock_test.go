//go:build e2e

package e2e

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	apiHost      = "api.track.toggl.com"
	fixtureToken = "toggl-e2e-test-token"
)

func startMockAPI(t *testing.T) string {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Host != apiHost {
			t.Errorf("unexpected method or host: %s %s", r.Method, r.Host)
			http.Error(w, "unexpected method or host", http.StatusBadRequest)
			return
		}
		token, password, ok := r.BasicAuth()
		if !ok || token != fixtureToken || password != "api_token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v9/workspaces/123/clients":
			io.WriteString(w, `[{"id":2,"name":"Example Client"}]`)
		case "/api/v9/workspaces/123/projects":
			io.WriteString(w, `[{"id":7,"name":"Example Project","active":true}]`)
		case "/api/v9/me/time_entries":
			query := r.URL.Query()
			if query.Get("start_date") != "2026-10-30T15:00:00Z" ||
				query.Get("end_date") != "2026-11-01T15:00:00Z" || query.Get("meta") != "true" {
				t.Errorf("unexpected time entries query: %s", r.URL.RawQuery)
				http.Error(w, "unexpected time entries query", http.StatusBadRequest)
				return
			}
			io.WriteString(w, `[
				{"id":1,"project_id":7,"start":"2026-10-30T15:00:00Z","stop":"2026-10-30T15:30:00Z","duration":1800},
				{"id":2,"project_id":7,"start":"2026-10-31T15:00:00Z","stop":"2026-10-31T16:00:00Z","duration":3600}
			]`)
		default:
			t.Errorf("unexpected API path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})

	cert, caFile := makeMockCertificate(t)
	listener, err := net.Listen("tcp4", "127.0.0.1:443")
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second},
		TLS:      &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return caFile
}

func makeMockCertificate(t *testing.T) (tls.Certificate, string) {
	t.Helper()

	now := time.Now()
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "toggl-cli E2E test CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caPublic, caKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, caPublic, caKey)
	if err != nil {
		t.Fatal(err)
	}

	leaf := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		DNSNames:     []string{apiHost},
		NotBefore:    ca.NotBefore,
		NotAfter:     ca.NotAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafPublic, leafKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, leafPublic, caKey)
	if err != nil {
		t.Fatal(err)
	}

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0600); err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}, caFile
}
