package dkim

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

// Params configures a DKIM signature.
type Params struct {
	Domain           string
	Selector         string
	PrivateKeyPEM    string
	HeaderNames      []string
	Canonicalization string
	Headers          map[string]string
	Body             []byte
	Now              time.Time
}

// Sign returns the DKIM-Signature header value (without the header name).
func Sign(p Params) (string, error) {
	if p.Domain == "" || p.Selector == "" {
		return "", fmt.Errorf("dkim: missing domain/selector")
	}
	key, err := parseRSAPrivateKey(p.PrivateKeyPEM)
	if err != nil {
		return "", err
	}
	if p.Canonicalization == "" {
		p.Canonicalization = "relaxed/relaxed"
	}
	if len(p.HeaderNames) == 0 {
		p.HeaderNames = []string{"from", "to", "subject", "date"}
	}
	bh := BodyHash(p.Body, p.Canonicalization)
	now := p.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	hList := make([]string, len(p.HeaderNames))
	for i, n := range p.HeaderNames {
		hList[i] = strings.ToLower(n)
	}
	var sig strings.Builder
	sig.WriteString("v=1; a=rsa-sha256; c=")
	sig.WriteString(p.Canonicalization)
	sig.WriteString("; d=")
	sig.WriteString(p.Domain)
	sig.WriteString("; s=")
	sig.WriteString(p.Selector)
	sig.WriteString("; t=")
	sig.WriteString(fmt.Sprintf("%d", now.Unix()))
	sig.WriteString("; bh=")
	sig.WriteString(bh)
	sig.WriteString("; h=")
	sig.WriteString(strings.Join(hList, ":"))
	sig.WriteString("; b=")

	toSign := HeaderHashInput(p.Headers, p.HeaderNames)
	toSign += "dkim-signature:" + relaxHeaderValue(sig.String()) + "\r\n"
	sum := sha256.Sum256([]byte(toSign))
	signed, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(signed)
	return sig.String() + foldB64(b64), nil
}

func foldB64(s string) string {
	if len(s) <= 70 {
		return s
	}
	var b strings.Builder
	for len(s) > 70 {
		b.WriteString(s[:70])
		b.WriteString(" ")
		s = s[70:]
	}
	b.WriteString(s)
	return b.String()
}

func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("dkim: no PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("dkim: not RSA private key")
	}
	return rk, nil
}

// GenerateTestKey returns a PEM private key for lab signing (1024-bit for speed).
func GenerateTestKey() (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", err
	}
	b := x509.MarshalPKCS1PrivateKey(key)
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b})), nil
}
