package cert

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/pkg/errors"
	"github.com/youmark/pkcs8"
	_ "github.com/youmark/pkcs8"
	"io/ioutil"
	"os/exec"
	"strings"
)

type Manager struct {
	CaCertificate             string  `json:"caCertificate" long:"ca-certificate" env:"CA_CERTIFICATE" description:"CA certificate(s)"`
	CaCertificateFile         string  `json:"caCertificateFile" long:"ca-certificate-file" env:"CA_CERTIFICATE_FILE" description:"File with CA certificate(s)"`
	Certificate               string  `json:"certificate" long:"certificate" env:"CERTIFICATE" description:"Authentication certificate"`
	CertificateFile           string  `json:"certificateFile" long:"certificate-file" env:"CERTIFICATE_FILE" description:"Authentication certificate"`
	PrivateKey                string  `json:"privateKey" long:"private-key" env:"PRIVATE_KEY" description:"Authentication private key"`
	PrivateKeyFile            string  `json:"privateKeyFile" long:"private-key-file" env:"PRIVATE_KEY_FILE" description:"Authentication private key"`
	PrivateKeyPassword        *string `json:"privateKeyPassword" long:"private-key-password" env:"PRIVATE_KEY_PASSWORD" description:"Decryption password"`
	PrivateKeyPasswordProgram string  `json:"privateKeyPasswordProgram" long:"private-key-password-program" env:"PRIVATE_KEY_PASSWORD_PROGRAM" description:"Program to run to get the decryption key"`
}

type ClientAuthentication  struct {
	RequireClientCert bool `json:"requireClientCert" long:"require-client-cert" env:"REQUIRE_CLIENT_CERT" description:"If set, the client must authenticate with its certificate."`
}

func (m *Manager) GetCertificate() ([]byte, error) {
	if m.CertificateFile != "" {
		certPemBlock, err := ioutil.ReadFile(m.CertificateFile)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not read certificate file: %s", m.CertificateFile)
		}
		return certPemBlock, nil
	} else if m.Certificate != "" {
		return []byte(strings.TrimSpace(m.Certificate)), nil
	}
	return nil, nil
}

func (m *Manager) GetPrivateKey() (privateKeyPemBlock []byte, err error) {
	if m.PrivateKeyFile != "" {
		privateKeyPemBlock, err = ioutil.ReadFile(m.PrivateKeyFile)
		if err != nil {
			err = errors.Wrapf(err, "Could not read private key file: %s", m.PrivateKeyFile)
		}
	} else if m.PrivateKey != "" {
		privateKeyPemBlock = []byte(strings.TrimSpace(m.PrivateKey))
		err = nil
	}

	if err != nil {
		return
	}

	if len(privateKeyPemBlock) > 0 {
		block, _ := pem.Decode(privateKeyPemBlock)

		if block.Type == "ENCRYPTED PRIVATE KEY" {
			var password []byte
			password, err = m.GetPrivateKeyPassword()
			if err != nil {
				return nil, errors.Wrapf(err, "Failed getting the key password")
			}

			key, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, password)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not decrypt private key!")
			}

			privateKeyPemBlock, err = x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				return nil, errors.Wrapf(err, "Don't know how to handle %+v", key)
			}
			privateKeyPemBlock = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyPemBlock})

		} else if x509.IsEncryptedPEMBlock(block) {
			var password []byte
			password, err = m.GetPrivateKeyPassword()
			if err != nil {
				return nil, errors.Wrapf(err, "Failed getting the key password")
			}

			privateKeyPemBlock, err = x509.DecryptPEMBlock(block, password)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not decrypt private key!")
			}
		}
	}

	return
}

func (m *Manager) GetPrivateKeyPassword() ([]byte, error) {
	if m.PrivateKeyPassword != nil {
		return []byte(*m.PrivateKeyPassword), nil
	} else if m.PrivateKeyPasswordProgram != "" {
		cmd := exec.Command("sh", "-c", m.PrivateKeyPasswordProgram)
		out := bytes.NewBuffer([]byte{})
		cmd.Stdout = out
		if err := cmd.Run(); err != nil {
			return nil, errors.Wrapf(err, "Failed executing %s", m.PrivateKeyPasswordProgram)
		}
		return out.Bytes(), nil
	} else {
		return nil, errors.Errorf("Private key is encrypted and no password or password program defined!")
	}
}

func (m *Manager) GetX509KeyPair() (*tls.Certificate, error) {
	var certPemBlock []byte
	var privateKeyPemBlock []byte
	var useHttps bool

	if c, err := m.GetCertificate(); err != nil {
		return nil, err
	} else if c != nil && len(c) > 0 {
		certPemBlock = c
		useHttps = true
	}
	if p, err := m.GetPrivateKey(); err != nil {
		return nil, err
	} else if p != nil && len(p) > 0 {
		privateKeyPemBlock = p
		useHttps = true
	}

	if useHttps {
		if cert, err := tls.X509KeyPair(certPemBlock, privateKeyPemBlock); err != nil {
			return nil, errors.Wrapf(err, "Could not create a X509 key pair from given data!")
		} else {
			return &cert, nil
		}
	} else {
		return nil, nil
	}
}

func (m *Manager) MakeTlsConfig(cert *tls.Certificate) *tls.Config {
	conf := &tls.Config{}
	conf.Certificates = make([]tls.Certificate, 1)
	conf.Certificates[0] = *cert

	return conf
}

func (m *Manager) AddCaCertificates(config *tls.Config) {
	if config == nil {
		return
	}

	// TODO: Add CA certificates here
}