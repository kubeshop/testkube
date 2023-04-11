package storage

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoSSLConfig contains the configurations necessary for an SSL connection
type MongoSSLConfig struct {
	// SSLClientCertificateKeyFile specifies a path to the client certificate and private key, which must be concatenated into one file.
	SSLClientCertificateKeyFile string
	// SSLClientCertificateKeyFilePassword specifies the password to decrypt the client private key file
	SSLClientCertificateKeyFilePassword string
	// SSLCertificateAuthoritiyFile specifies the path to a single or bundle of certificate authorities
	SSLCertificateAuthoritiyFile string
}

const (
	TypeMongoDB    = "mongo"
	TypeDocDB      = "docdb"
	DocDBcaFileURI = "https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem"
)

// GetMongoDatabase returns a valid database connection to the configured MongoDB database
func GetMongoDatabase(dsn, name, dbType string, certConfig *MongoSSLConfig) (db *mongo.Database, err error) {
	var mongoOptions *tls.Config

	switch dbType {
	case TypeMongoDB, "":
		if certConfig != nil {
			mongoOptions, err = options.BuildTLSConfig(map[string]interface{}{
				"sslClientCertificateKeyFile":     certConfig.SSLClientCertificateKeyFile,
				"sslClientCertificateKeyPassword": certConfig.SSLClientCertificateKeyFilePassword,
				"sslCertificateAuthorityFile":     certConfig.SSLCertificateAuthoritiyFile,
			})
			if err != nil {
				return nil, fmt.Errorf("could not build SSL config: %w", err)
			}
		}
	case TypeDocDB:
		mongoOptions, err = getDocDBTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("could not get DocDB: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database type %s", dbType)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().SetTLSConfig(mongoOptions).ApplyURI(dsn))
	if err != nil {
		return nil, err
	}

	return client.Database(name), nil
}

func getDocDBTLSConfig() (*tls.Config, error) {
	caFilePath, err := GetDocDBcaFile()
	if err != nil {
		return nil, fmt.Errorf("could not get CA file: %w", err)
	}

	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFilePath)

	if err != nil {
		return nil, fmt.Errorf("could not read CA file: %s", err)
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)

	if !ok {
		return nil, errors.New("failed parsing pem file")
	}

	return tlsConfig, nil
}

// GetDocDBcaFile will fetch the file located at DocDBcaFileURI into a local file
// Due to size limitations we cannot use Kubernetes secrets like we use for MongoDB TLS configs
func GetDocDBcaFile() (string, error) {
	// Get the data
	resp, err := http.Get(DocDBcaFileURI)
	if err != nil {
		return "", fmt.Errorf("could not fetch file from %s: %w", DocDBcaFileURI, err)
	}
	defer resp.Body.Close()

	docDBcaPath := "/tmp/rds-combined-ca-bundle.pem"
	out, err := os.Create(docDBcaPath)
	if err != nil {
		return "", fmt.Errorf("could not create file %s: %w", docDBcaPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not write file %s: %w", docDBcaPath, err)
	}
	return docDBcaPath, nil
}
