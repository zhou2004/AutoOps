package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"dodevops-api/api/monitor/model"
)

func TestDomainCertCollectorStartsRepeatsAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var sweeps atomic.Int32
	checked := make(chan uint, 20)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDomainCertChecks(ctx, 10*time.Millisecond, func() ([]model.DomainCert, error) {
			sweeps.Add(1)
			return []model.DomainCert{{ID: 1}, {ID: 2}}, nil
		}, func(ctx context.Context, cert *model.DomainCert) { checked <- cert.ID })
	}()
	for i := 0; i < 4; i++ {
		select {
		case <-checked:
		case <-time.After(time.Second):
			t.Fatal("automatic checks did not run without an HTTP request")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("collector did not stop")
	}
	if sweeps.Load() < 2 {
		t.Fatal("collector did not repeat")
	}
}

func TestDomainCertProbeTLSAndRecentExpiry(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, tc := range []struct {
		name   string
		expiry time.Time
		status int
	}{
		{"valid", now.Add(90 * 24 * time.Hour), 1},
		{"expiring", now.Add(10 * 24 * time.Hour), 2},
		{"expired-one-hour-ago", now.Add(-time.Hour), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "monitor-test"}, NotBefore: now.Add(-48 * time.Hour), NotAfter: tc.expiry, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
			der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			server.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
			server.StartTLS()
			defer server.Close()
			host, port, _ := net.SplitHostPort(server.Listener.Addr().String())
			n, _ := strconv.Atoi(port)
			row := &model.DomainCert{Domain: host, Port: n}
			(&DomainCertServiceImpl{}).checkCertContext(context.Background(), row)
			if row.Status != tc.status || row.CheckTime == "" || row.Subject != "monitor-test" {
				t.Fatalf("unexpected observation: %+v", row)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	row := &model.DomainCert{Domain: "127.0.0.1", Port: 1}
	(&DomainCertServiceImpl{}).checkCertContext(ctx, row)
	if row.Status != 4 || row.RemainingDays != -1 || row.ErrorMsg == "" {
		t.Fatalf("failure not captured: %+v", row)
	}
}
