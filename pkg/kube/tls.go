package kube

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TLSSignals holds parsed certificate data from a kubernetes.io/tls secret
type TLSSignals struct {
	SecretName string
	Namespace  string
	Issuer     string
	Subject    string
	NotBefore  time.Time
	NotAfter   time.Time
	IsValid    bool
	ParseError string
}

// CollectTLSSignals parses a TLS secret and validates its certificate expiry
func CollectTLSSignals(
	client *kubernetes.Clientset,
	name, namespace string,
) (*TLSSignals, error) {

	ctx := context.Background()

	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"secret %q not found in namespace %q: %w",
			name, namespace, err)
	}

	if secret.Type != corev1.SecretTypeTLS {
		return nil, fmt.Errorf("secret %q is not of type kubernetes.io/tls", name)
	}

	signals := &TLSSignals{
		SecretName: name,
		Namespace:  namespace,
	}

	crtData, ok := secret.Data["tls.crt"]
	if !ok {
		signals.ParseError = "missing tls.crt"
		return signals, nil
	}

	block, _ := pem.Decode(crtData)
	if block == nil {
		signals.ParseError = "failed to decode PEM block"
		return signals, nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		signals.ParseError = err.Error()
		return signals, nil
	}

	signals.Issuer = cert.Issuer.CommonName
	signals.Subject = cert.Subject.CommonName
	signals.NotBefore = cert.NotBefore
	signals.NotAfter = cert.NotAfter

	now := time.Now()
	signals.IsValid = now.After(cert.NotBefore) && now.Before(cert.NotAfter)

	return signals, nil
}
