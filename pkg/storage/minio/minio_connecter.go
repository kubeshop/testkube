package minio

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Option func(*Connecter) error

// Insecure is an Option to enable TLS secure connections that skip server verification.
func Insecure() Option {
	return func(o *Connecter) error {
		if o.TlsConfig == nil {
			o.TlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		o.TlsConfig.InsecureSkipVerify = true
		o.Ssl = true
		return nil
	}
}

// RootCAs is a helper option to provide the RootCAs pool from a list of filenames.
// If Secure is not already set this will set it as well.
func RootCAs(file ...string) Option {
	return func(o *Connecter) error {
		pool := x509.NewCertPool()
		for _, f := range file {
			rootPEM, err := os.ReadFile(f)
			if err != nil || rootPEM == nil {
				return fmt.Errorf("minio: error loading or parsing rootCA file: %v", err)
			}
			ok := pool.AppendCertsFromPEM(rootPEM)
			if !ok {
				return fmt.Errorf("minio: failed to parse root certificate from %q", f)
			}
		}
		if o.TlsConfig == nil {
			o.TlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		o.TlsConfig.RootCAs = pool
		o.Ssl = true
		return nil
	}
}

// ClientCert is a helper option to provide the client certificate from a file.
// If Secure is not already set this will set it as well.
func ClientCert(certFile, keyFile string) Option {
	return func(o *Connecter) error {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("minio: error loading client certificate: %v", err)
		}
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return fmt.Errorf("minio: error parsing client certificate: %v", err)
		}
		if o.TlsConfig == nil {
			o.TlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		o.TlsConfig.Certificates = []tls.Certificate{cert}
		o.Ssl = true
		return nil
	}
}

type Connecter struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Token           string
	Bucket          string
	Ssl             bool
	TlsConfig       *tls.Config
	Opts            []Option
	Log             *zap.SugaredLogger
	client          *minio.Client
}

// NewConnecter creates a new Connecter
func NewConnecter(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, log *zap.SugaredLogger, opts ...Option) *Connecter {
	c := &Connecter{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
		Token:           token,
		Bucket:          bucket,
		Opts:            opts,
		Log:             log,
	}
	return c
}

// GetClient() connects to MinIO
func (c *Connecter) GetClient() (*minio.Client, error) {
	if c.client != nil {
		return c.client, nil
	}

	for _, opt := range c.Opts {
		if err := opt(c); err != nil {
			return nil, errors.Wrapf(err, "error connecting to server")
		}
	}
	creds := credentials.NewIAM("")
	c.Log.Debugw("connecting to server",
		"endpoint", c.Endpoint,
		"accessKeyID", c.AccessKeyID,
		"region", c.Region,
		"token", c.Token,
		"bucket", c.Bucket,
		"ssl", c.Ssl)
	if c.AccessKeyID != "" && c.SecretAccessKey != "" {
		creds = credentials.NewStaticV4(c.AccessKeyID, c.SecretAccessKey, c.Token)
	}
	transport, err := minio.DefaultTransport(c.Ssl)
	if err != nil {
		c.Log.Errorw("error creating minio transport", "error", err)
		return nil, err
	}
	transport.TLSClientConfig = c.TlsConfig
	opts := &minio.Options{
		Creds:     creds,
		Secure:    c.Ssl,
		Transport: transport,
	}
	if c.Region != "" {
		opts.Region = c.Region
	}
	mclient, err := minio.New(c.Endpoint, opts)
	if err != nil {
		c.Log.Errorw("error connecting to minio", "error", err)
		return nil, err
	}

	c.client = mclient
	return mclient, nil
}
