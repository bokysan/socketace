package cert

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/bokysan/socketace/v2/internal/args"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/youmark/pkcs8"
	_ "github.com/youmark/pkcs8"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TlsConfig interface {
	GetTlsConfig() (*tls.Config, error)
}

// Config is the generic certificate configuration
type Config struct {
	CaCertificate             string  `json:"caCertificate" long:"ca-certificate" env:"CA_CERTIFICATE" description:"CA certificate(s)"`
	CaCertificateFile         string  `json:"caCertificateFile" long:"ca-certificate-file" env:"CA_CERTIFICATE_FILE" description:"File with CA certificate(s)"`
	Certificate               string  `json:"certificate" long:"certificate" env:"CERTIFICATE" description:"Authentication certificate"`
	CertificateFile           string  `json:"certificateFile" long:"certificate-file" env:"CERTIFICATE_FILE" description:"Authentication certificate"`
	PrivateKey                string  `json:"privateKey" long:"private-key" env:"PRIVATE_KEY" description:"Authentication private key"`
	PrivateKeyFile            string  `json:"privateKeyFile" long:"private-key-file" env:"PRIVATE_KEY_FILE" description:"Authentication private key"`
	PrivateKeyPassword        *string `json:"privateKeyPassword" long:"private-key-password" env:"PRIVATE_KEY_PASSWORD" description:"Decryption password"`
	PrivateKeyPasswordProgram string  `json:"privateKeyPasswordProgram" long:"private-key-password-program" env:"PRIVATE_KEY_PASSWORD_PROGRAM" description:"Program to run to get the decryption key"`
}

// ClientConfig is the certificate configuration with client-specific extensions
type ClientConfig struct {
	Config
	InsecureSkipVerify bool `json:"insecure" short:"k" long:"insecure"  env:"INSECURE" description:"Allows insecure connections"`
}

// ServerConfig is the certificate configuration with server-specific extensions
type ServerConfig struct {
	Config
	RequireClientCert bool `json:"requireClientCert" long:"require-client-cert" env:"REQUIRE_CLIENT_CERT" description:"If set, the client must authenticate with its certificate."`
}

type ConfigGetter interface {
	CertManager() TlsConfig
}

func (m *Config) GetCertificate() ([]byte, error) {
	if m.CertificateFile != "" {
		certPemBlock, err := ioutil.ReadFile(findFile(m.CertificateFile))
		if err != nil {
			return nil, errors.Wrapf(err, "Could not read certificate file: %s", m.CertificateFile)
		}
		return certPemBlock, nil
	} else if m.Certificate != "" {
		return []byte(strings.TrimSpace(m.Certificate)), nil
	}
	return nil, nil
}

func (m *Config) GetPrivateKey() (privateKeyPemBlock []byte, err error) {
	if m.PrivateKeyFile != "" {
		privateKeyPemBlock, err = ioutil.ReadFile(findFile(m.PrivateKeyFile))
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

func (m *Config) GetPrivateKeyPassword() ([]byte, error) {
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

func (m *Config) GetX509KeyPair() (*tls.Certificate, error) {
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

func (m *Config) GetCaCertificates() ([]byte, error) {
	if m.CaCertificateFile != "" {
		certPemBlock, err := ioutil.ReadFile(findFile(m.CaCertificateFile))
		if err != nil {
			return nil, errors.Wrapf(err, "Could not read ca certificate file: %s", m.CaCertificateFile)
		}
		return certPemBlock, nil
	} else if m.CaCertificate != "" {
		return []byte(strings.TrimSpace(m.CaCertificate)), nil
	}
	return nil, nil
}

func (m *Config) addCaCertificates(config *tls.Config) (err error) {
	if config == nil {
		return
	}

	caCert, err := m.GetCaCertificates()
	if err != nil {
		return errors.Wrapf(err, "Could not load CA certificates")
	}

	// ClientAuth: tls.RequireAndVerifyClientCert,

	if caCert != nil {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return errors.Errorf("Could not parse CA certificates")
		}

		config.ClientCAs = caCertPool
		config.RootCAs = caCertPool
	}

	return nil

}

func (m *Config) GetTlsConfig() (*tls.Config, error) {
	conf := &tls.Config{}

	if crt, err := m.GetX509KeyPair(); err != nil {
		return nil, errors.Wrapf(err, "Could not read certificate pair")
	} else if crt != nil {
		conf.Certificates = make([]tls.Certificate, 1)
		conf.Certificates[0] = *crt
	}
	if err := m.addCaCertificates(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func (m *ClientConfig) GetTlsConfig() (conf *tls.Config, err error) {
	log.Debugf("ClientConfig.GetTlsConfig(), InsecureSkipVerify=%v", m.InsecureSkipVerify)

	conf, err = m.Config.GetTlsConfig()

	if err == nil {
		if m.InsecureSkipVerify {
			conf.InsecureSkipVerify = true
		}
	}

	return
}

func (m *ServerConfig) GetTlsConfig() (conf *tls.Config, err error) {
	log.Debug("ServerConfig.GetTlsConfig()")
	conf, err = m.Config.GetTlsConfig()

	if err != nil {
		if m.RequireClientCert {
			conf.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	return
}

// findFile will try to locale the file based on relaltive path of the configuration location and,
// failing that, return the provided location as ist
func findFile(name string) string {
	if args.General.ConfigurationFilePath != "" {
		path := filepath.Dir(args.General.ConfigurationFilePath)
		file := filepath.Join(path, name)

		_, err := os.Stat(file)
		if !os.IsNotExist(err) {
			return file
		}
	}

	return name
}
