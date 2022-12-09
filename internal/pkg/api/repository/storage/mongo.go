package storage

import (
	"context"
	"crypto/tls"
	"fmt"
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

// GetMongoDatabase returns a valid database connection to the configured MongoDB database
func GetMongoDatabase(dsn, name string, certConfig *MongoSSLConfig) (db *mongo.Database, err error) {
	var mongoOptions *tls.Config
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().SetTLSConfig(mongoOptions).ApplyURI(dsn))
	if err != nil {
		return nil, err
	}

	return client.Database(name), nil
}
