package linuxhost_client

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"

	"fmt"
	"log"
	"os"
)

type CertificateData struct {
	Sha256Fingerprint [32]byte
	Subject           string
}

func CertificateInfo(certificatePath string) CertificateData {
	/// Load the certificate (PEM format)
	certPEM, err := os.ReadFile(certificatePath)
	if err != nil {
		log.Fatalf("Error reading certificate file (%s): %v", certificatePath, err)
	}

	/// Decode the PEM block
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		log.Fatalf("Failed to decode PEM block containing the certificate")
	}

	/// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("Error parsing certificate: %v", err)
	}

	/// Compute the fingerprint
	subject := cert.Subject.ToRDNSequence().String()
	hash := sha256.Sum256(cert.Raw)

	return CertificateData{
		Subject:           subject,
		Sha256Fingerprint: hash,
	}
}

func decodeBytesString(hexFingerprint string) ([32]byte, error) {
	var fingerprint [32]byte
	/// Remove colons
	hexFingerprint = strings.ReplaceAll(hexFingerprint, ":", "")
	/// Decode hex
	bytes, err := hex.DecodeString(hexFingerprint)
	if err != nil {
		return fingerprint, err
	}
	if len(bytes) != 32 {
		return fingerprint, fmt.Errorf("invalid fingerprint length: %d", len(bytes))
	}
	/// Copy to [32]byte
	copy(fingerprint[:], bytes)
	return fingerprint, nil
}
func EncodeBytesString(fingerprint [32]byte) string {
	/// Convert to hexadecimal string
	hexStr := hex.EncodeToString(fingerprint[:])
	var result strings.Builder
	for i := 0; i < len(hexStr); i += 2 {
		/// Add two characters
		result.WriteString(hexStr[i : i+2])
		if i+2 < len(hexStr) {
			/// Add colon separator
			result.WriteString(":")
		}
	}
	return result.String()
}

func SetRemoteCaTrust(context SSHCommandContext) SSHCommandContext {
	result := context.Exec("sudo update-ca-certificates -f")
	if result.Error != nil {
		return result
	}
	return result
}

func Sha256Fingerprint(cert *x509.Certificate) [32]byte {
	hash := sha256.Sum256(cert.Raw)
	return hash
}

func parseX509Certificates(output string) []*x509.Certificate {
	rest := []byte(output)
	certs := []*x509.Certificate{}
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatal(err)
		}
		certs = append(certs, cert)
	}
	return certs
}

func RefreshRemoteCertificates(client *SSHClientContext) []*x509.Certificate {
	c := "cat /etc/ssl/certs/ca-certificates.crt"
	cmd := "sudo bash -c '" + c + "'"
	result := NewSSHCommandContext(client).Exec(cmd)

	if result.Error != nil {
		fmt.Println("Error", result.Error)
	}
	return parseX509Certificates(result.Output)
}
