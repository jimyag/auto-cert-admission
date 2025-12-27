package certprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNew(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	if provider.namespace != "test-ns" {
		t.Errorf("namespace: got %q, want %q", provider.namespace, "test-ns")
	}
	if provider.name != "test-secret" {
		t.Errorf("name: got %q, want %q", provider.name, "test-secret")
	}
	if provider.client != client {
		t.Error("client not set correctly")
	}
}

func TestProvider_Ready(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	// Initially not ready
	if provider.Ready() {
		t.Error("Provider should not be ready initially")
	}

	// Set ready
	provider.ready.Store(true)
	if !provider.Ready() {
		t.Error("Provider should be ready after setting")
	}
}

func TestProvider_GetCertificate_NotLoaded(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	_, err := provider.GetCertificate(nil)
	if err == nil {
		t.Error("Expected error when certificate not loaded")
	}
}

func TestProvider_GetCertificate_Loaded(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	// Create a test certificate
	certPEM, keyPEM := generateTestCert(t)

	// Create secret with certificate
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}

	// Trigger onSecretUpdate
	provider.onSecretUpdate(secret)

	// Should be able to get certificate now
	cert, err := provider.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if cert == nil {
		t.Error("Expected non-nil certificate")
	}

	if !provider.Ready() {
		t.Error("Provider should be ready after loading certificate")
	}
}

func TestProvider_onSecretUpdate_NoCert(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	// Secret without tls.crt
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{},
	}

	provider.onSecretUpdate(secret)

	if provider.Ready() {
		t.Error("Provider should not be ready without certificate")
	}
}

func TestProvider_onSecretUpdate_NoKey(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	certPEM, _ := generateTestCert(t)

	// Secret with cert but no key
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"tls.crt": certPEM,
		},
	}

	provider.onSecretUpdate(secret)

	if provider.Ready() {
		t.Error("Provider should not be ready without key")
	}
}

func TestProvider_onSecretUpdate_InvalidCert(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	// Secret with invalid certificate data
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("invalid cert"),
			"tls.key": []byte("invalid key"),
		},
	}

	provider.onSecretUpdate(secret)

	if provider.Ready() {
		t.Error("Provider should not be ready with invalid certificate")
	}
}

func TestProvider_loadCertificate_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	ctx := context.Background()
	err := provider.loadCertificate(ctx)
	// Should not return error for not found
	if err != nil {
		t.Errorf("loadCertificate should not return error for not found: %v", err)
	}
}

func TestProvider_loadCertificate_Exists(t *testing.T) {
	certPEM, keyPEM := generateTestCert(t)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
		},
	}

	client := fake.NewSimpleClientset(secret)
	provider := New(client, "test-ns", "test-secret")

	ctx := context.Background()
	err := provider.loadCertificate(ctx)
	if err != nil {
		t.Fatalf("loadCertificate failed: %v", err)
	}

	if !provider.Ready() {
		t.Error("Provider should be ready after loading certificate")
	}
}

func TestProvider_CertificateReload(t *testing.T) {
	certPEM1, keyPEM1 := generateTestCert(t)
	certPEM2, keyPEM2 := generateTestCert(t)

	client := fake.NewSimpleClientset()
	provider := New(client, "test-ns", "test-secret")

	// Initially not ready
	if provider.Ready() {
		t.Error("Provider should not be ready initially")
	}

	// Load first certificate
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"tls.crt": certPEM1,
			"tls.key": keyPEM1,
		},
	}
	provider.onSecretUpdate(secret1)

	if !provider.Ready() {
		t.Error("Provider should be ready after loading first certificate")
	}

	cert1, err := provider.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if cert1 == nil || len(cert1.Certificate) == 0 {
		t.Fatal("Expected non-nil certificate with data")
	}

	// Load second certificate (simulating rotation)
	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-ns",
		},
		Data: map[string][]byte{
			"tls.crt": certPEM2,
			"tls.key": keyPEM2,
		},
	}
	provider.onSecretUpdate(secret2)

	cert2, err := provider.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate failed after reload: %v", err)
	}
	if cert2 == nil || len(cert2.Certificate) == 0 {
		t.Fatal("Expected non-nil certificate with data after reload")
	}

	// Compare raw certificate bytes - they should be different
	if string(cert1.Certificate[0]) == string(cert2.Certificate[0]) {
		t.Error("Certificates should be different after reload")
	}
}

// generateTestCert generates a self-signed test certificate
func generateTestCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certPEM, keyPEM
}
