/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package teleport

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/identityfile"
	"golang.org/x/crypto/ssh"
)

// TestReloadIdentity_PicksUpRotatedIdentity covers the failure mode that took
// the exporter down: tbot rotates the identity file every renewal interval,
// but the credentials the API client was built with kept serving the
// certificate that was on disk at startup. The exporter therefore ran on an
// expired certificate and every reconnect failed with "cert has expired".
//
// The TLS and SSH configs are fetched once here on purpose: that is what the
// API client does at dial time, and it reuses those same configs for every
// later reconnect.
func TestReloadIdentity_PicksUpRotatedIdentity(t *testing.T) {
	const (
		generation1 = "generation-1"
		generation2 = "generation-2"
	)

	path := filepath.Join(t.TempDir(), "identity")
	writeIdentityFile(t, path, generation1)

	creds, err := client.NewDynamicIdentityFileCreds(path)
	if err != nil {
		t.Fatalf("NewDynamicIdentityFileCreds: %v", err)
	}

	tlsConfig, err := creds.TLSConfig()
	if err != nil {
		t.Fatalf("TLSConfig: %v", err)
	}

	sshConfig, err := creds.SSHClientConfig()
	if err != nil {
		t.Fatalf("SSHClientConfig: %v", err)
	}
	// The identity file's known hosts entry is not what is under test here,
	// only the certificate the client offers.
	sshConfig.HostKeyCallback = func(string, net.Addr, ssh.PublicKey) error { return nil }

	if got := clientCertCommonName(t, tlsConfig); got != generation1 {
		t.Fatalf("initial client certificate = %q, want generation-1", got)
	}
	if got := offeredSSHCertKeyID(t, sshConfig); got != generation1 {
		t.Fatalf("initial SSH certificate = %q, want generation-1", got)
	}

	writeIdentityFile(t, path, generation2)

	if got := clientCertCommonName(t, tlsConfig); got != generation1 {
		t.Fatalf("client certificate = %q before reload, want generation-1", got)
	}

	c := &Client{creds: creds}
	if err := c.ReloadIdentity(); err != nil {
		t.Fatalf("ReloadIdentity: %v", err)
	}

	if got := clientCertCommonName(t, tlsConfig); got != generation2 {
		t.Fatalf("client certificate = %q after reload, want generation-2: a rotated identity is never picked up", got)
	}
	// The reported failure was "ssh: cert has expired", so the SSH path is the
	// one that has to keep working.
	if got := offeredSSHCertKeyID(t, sshConfig); got != generation2 {
		t.Fatalf("SSH certificate = %q after reload, want generation-2: a rotated identity is never picked up", got)
	}
}

// offeredSSHCertKeyID handshakes far enough for the client to offer its
// certificate and returns that certificate's key ID, which is how a caller
// tells two generations apart. The server rejects the key: the offer is all
// this needs to observe.
func offeredSSHCertKeyID(t *testing.T, cfg *ssh.ClientConfig) string {
	t.Helper()

	_, hostKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostKey)
	if err != nil {
		t.Fatalf("ssh host signer: %v", err)
	}

	var offered string
	serverConfig := &ssh.ServerConfig{
		//nolint:gosec // G408: this server only records the offered certificate, it never authenticates
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if cert, ok := key.(*ssh.Certificate); ok {
				offered = cert.KeyId
			}
			return nil, errors.New("rejected: the offered certificate is all the test needs")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	// A loopback listener rather than net.Pipe: the two handshakes write their
	// version strings at the same time, which an unbuffered pipe deadlocks on.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		//nolint:errcheck,gosec // authentication is rejected on purpose
		ssh.NewServerConn(conn, serverConfig)
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = clientConn.Close() }()

	// Authentication is expected to fail; only the certificate the client
	// offered on the way there matters.
	if conn, _, _, err := ssh.NewClientConn(clientConn, "proxy.example.com:3023", cfg); err == nil {
		_ = conn.Close()
	}
	<-done

	if offered == "" {
		t.Fatal("client offered no SSH certificate")
	}
	return offered
}

// clientCertCommonName returns the common name of the certificate the config
// offers to a server, which is how a caller tells two generations apart.
func clientCertCommonName(t *testing.T, cfg *tls.Config) string {
	t.Helper()

	if cfg.GetClientCertificate == nil {
		t.Fatal("TLS config has no GetClientCertificate callback, so the certificate cannot change")
	}

	cert, err := cfg.GetClientCertificate(&tls.CertificateRequestInfo{})
	if err != nil {
		t.Fatalf("GetClientCertificate: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("GetClientCertificate returned no certificate")
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse client certificate: %v", err)
	}
	return leaf.Subject.CommonName
}

// writeIdentityFile writes a valid identity file whose TLS and SSH
// certificates are tagged with commonName, so a test can tell which generation
// of the file is in use.
func writeIdentityFile(t *testing.T, path, commonName string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	// Self-signed, so the same certificate serves as its own CA.
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("ssh signer: %v", err)
	}

	sshCert := &ssh.Certificate{
		Key:             signer.PublicKey(),
		CertType:        ssh.UserCert,
		KeyId:           commonName,
		ValidPrincipals: []string{"-teleport-internal-join"},
		ValidAfter:      uint64(time.Now().Add(-time.Hour).Unix()), //nolint:gosec // G115: a validity window around now fits uint64
		ValidBefore:     uint64(time.Now().Add(time.Hour).Unix()),  //nolint:gosec // G115: a validity window around now fits uint64
	}
	if err := sshCert.SignCert(rand.Reader, signer); err != nil {
		t.Fatalf("sign ssh certificate: %v", err)
	}

	knownHost := append([]byte("@cert-authority *.example.com "), ssh.MarshalAuthorizedKey(signer.PublicKey())...)

	idFile := &identityfile.IdentityFile{
		PrivateKey: keyPEM,
		Certs: identityfile.Certs{
			SSH: ssh.MarshalAuthorizedKey(sshCert),
			TLS: certPEM,
		},
		CACerts: identityfile.CACerts{
			SSH: [][]byte{knownHost},
			TLS: [][]byte{certPEM},
		},
	}

	if err := identityfile.Write(idFile, path); err != nil {
		t.Fatalf("write identity file: %v", err)
	}
}
