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
		log.Fatalf("Error reading certificate file: %v", err)
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
func parseFingerprint(hexFingerprint string) ([32]byte, error) {
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
func EncodeFingerprint(fingerprint [32]byte) string {
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

func fingerprintHandler(p *Parser[CertificateData], line string) {
	if !strings.HasPrefix(line, "SHA256 Fingerprint=") {
		return
	}
	fingerprint, _ := parseFingerprint(strings.TrimPrefix(line, "SHA256 Fingerprint="))

	p.AddItem(CertificateData{
		Sha256Fingerprint: fingerprint,
	})
}

func detailHandler(p *Parser[CertificateData], line string) {
	if p.CurrentItem == nil {
		fmt.Println("No current certificate")
		return
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if key == "subject" {
		p.CurrentItem.Subject = value
	}
}

func SetRemoteCaTrust(context SSHCommandContext) SSHCommandContext {
	result := context.Exec("sudo update-ca-certificates -f")
	if result.Error != nil {
		return result
	}
	return result
}

func parseOpenSSLCertificates(output string) []CertificateData {
	parser := NewParser[CertificateData]()
	parser.AddHandler(fingerprintHandler)
	parser.AddHandler(detailHandler)

	parser.Parse(output)
	return parser.Items
}
func RefreshRemoteCertificates(client *SSHClientContext) []CertificateData {
	cmd := "sudo bash -c 'while openssl x509 -noout -fingerprint -sha256 -serial -issuer -dates -subject ; do :; done < /etc/ssl/certs/ca-certificates.crt'"
	result := NewSSHCommandContext(client).Exec(cmd)

	if result.Error != nil {
		fmt.Println("Error", result.Error)
	}
	return parseOpenSSLCertificates(result.Output)
}
