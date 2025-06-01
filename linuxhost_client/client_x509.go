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
	models "terraform-provider-linuxhost/models"
)

type CertificateData struct {
	Sha256Fingerprint [32]byte
	SerialNumber      [20]byte
	Subject           string
}

// type multilineParserContext struct {
// 	currentKey *string
// }

func CertificateInfo(certificateContent string) CertificateData {
	/// Load the certificate (PEM format)
	// certPEM, err := os.ReadFile(certificatePath)
	// if err != nil {
	// 	log.Fatalf("Error reading certificate file (%s): %v", certificatePath, err)
	// }

	/// Decode the PEM block
	block, _ := pem.Decode([]byte(certificateContent))
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
	serialBytes := cert.SerialNumber.Bytes()
	var serialNumber [20]byte
	if len(serialBytes) <= 20 {
		copy(serialNumber[20-len(serialBytes):], serialBytes)
	}
	hash := sha256.Sum256(cert.Raw)

	return CertificateData{
		Subject:           subject,
		SerialNumber:      serialNumber,
		Sha256Fingerprint: hash,
	}
}

//	func decodeBytesString(hexFingerprint string) ([]byte, error) {
//		var fingerprint []byte
//		/// Remove colons
//		hexFingerprint = strings.ReplaceAll(hexFingerprint, ":", "")
//		/// Decode hex
//		bytes, err := hex.DecodeString(hexFingerprint)
//		if err != nil {
//			return fingerprint, err
//		}
//		if len(bytes) != 32 {
//			return fingerprint, fmt.Errorf("invalid fingerprint length: %d", len(bytes))
//		}
//		/// Copy to [32]byte
//		copy(fingerprint[:], bytes)
//		return fingerprint, nil
//	}
func EncodeBytesString(fingerprint []byte) string {
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

// var re_keyMatcher = regexp.MustCompile("^ *(.*): ?(.*)$")

// func keyValueHandler(p *Parser[CertificateData, multilineParserContext], line string) {
// 	if match := re_keyMatcher.FindStringSubmatch(line); match != nil {
// 		p.CurrentContext.currentKey = &match[1]
// 	} else {
// 		p.CurrentContext.currentKey = nil
// 	}
// 	if p.CurrentContext.currentKey == nil {
// 		return
// 	}
// 	if *p.CurrentContext.currentKey == "Certificate" {
// 		p.AddItem(CertificateData{})
// 	}
// }

// func fingerprintHandler(p *Parser[CertificateData, multilineParserContext], line string) {
// 	prefix := "SHA256 FINGERPRINT="
// 	upper := strings.ToUpper(line)
// 	if !strings.HasPrefix(upper, prefix) {
// 		// fmt.Printf("Line isn't a sha: %s", line)
// 		return
// 	}
// 	fingerprint, _ := parseFingerprint(strings.TrimPrefix(upper, prefix))
// 	fmt.Printf("Got certificate with fingerprint %s", fingerprint)

// 	p.AddItem(CertificateData{
// 		Sha256Fingerprint: fingerprint,
// 	})
// }

// func detailHandler(p *Parser[CertificateData, multilineParserContext], line string) {
// 	if p.CurrentItem == nil {
// 		fmt.Println("No current certificate")
// 		return
// 	}
// 	parts := strings.SplitN(line, "=", 2)
// 	if len(parts) != 2 {
// 		return
// 	}
// 	key := strings.TrimSpace(parts[0])
// 	value := strings.TrimSpace(parts[1])

// 	if key == "subject" {
// 		p.CurrentItem.Subject = value
// 	}
// }

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
	// currentContext := &multilineParserContext{}
	// parser := NewParser[CertificateData](currentContext)
	// // parser.AddHandler(nextCertificateHandler)
	// parser.AddHandler(keyValueHandler)
	// // parser.AddHandler(fingerprintHandler)
	// parser.AddHandler(detailHandler)

	// parser.Parse(output)
	// return parser.Items
}
func RefreshRemoteCertificates(client *SSHClientContext) []*x509.Certificate {
	c := "cat /etc/ssl/certs/ca-certificates.crt"
	// cmd := "sudo bash -c 'while openssl x509 -noout -fingerprint -sha256 -serial -issuer -dates -subject ; do :; done < /etc/ssl/certs/ca-certificates.crt'"
	// c := "openssl crl2pkcs7 -nocrl -certfile /etc/ssl/certs/ca-certificates.crt | openssl pkcs7 -print_certs -fingerprint -sha256 -serial -issuer -dates -subject -noout"
	cmd := "sudo bash -c '" + c + "'"
	result := NewSSHCommandContext(client).Exec(cmd)

	if result.Error != nil {
		fmt.Println("Error", result.Error)
	}
	return parseX509Certificates(result.Output)
}

func CertificateContent(data models.CaCertificateModel) (*string, error) {
	if data.Source.IsUnknown() || data.Source.IsNull() {
		str := data.Certificate.ValueString()
		return &str, nil
	} else {
		localFile, err := os.ReadFile(data.Source.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to open local file: %v", err)
		}
		str := string(localFile)
		return &str, nil

	}
}
