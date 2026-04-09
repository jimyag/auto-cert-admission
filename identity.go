package autocertwebhook

import (
	"crypto/rand"
	"encoding/hex"
	"os"

	"k8s.io/klog/v2"
)

func getRuntimeIdentity() string {
	identity := os.Getenv("POD_NAME")
	if identity != "" {
		return identity
	}

	hostname, err := os.Hostname()
	if err != nil {
		klog.Errorf("Failed to get hostname: %v, generating random identity", err)
		hostname = "unknown-" + randomSuffix()
	}
	return hostname
}

func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(b)
}
