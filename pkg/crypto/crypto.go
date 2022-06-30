package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
)

const (
	// KeyLength is key length
	KeyLength = 1024
	// KeyType is key type
	KeyType = "RSA PRIVATE KEY"
)

// GenerateKey generates key
func GenerateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, KeyLength)
}

// Provider is a crypto provider
type Provider struct {
	privateKey *rsa.PrivateKey
	label      string
}

// NewProvider creates new provider
func NewProvider(rsaKey []byte, label string) (*Provider, error) {
	block, _ := pem.Decode(rsaKey)
	if block == nil {
		return nil, errors.New("bad key data")
	}

	if block.Type != KeyType {
		return nil, errors.New("unknown key type")
	}

	// Decode the RSA private key
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &Provider{privateKey: key, label: label}, nil
}

// ExportPrivateKey exports private key
func (p Provider) ExportPrivateKey() string {
	data := x509.MarshalPKCS1PrivateKey(p.privateKey)
	bytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  KeyType,
			Bytes: data,
		},
	)
	return string(bytes)
}

// EncryptData encrypts data
func (p Provider) EncryptData(source []byte) (string, error) {
	hash := sha256.New()
	length := len(source)
	step := p.privateKey.PublicKey.Size() - 2*hash.Size() - 2

	if step <= 0 {
		return "", errors.New("bad step value")
	}

	var bytes []byte
	for start := 0; start < length; start += step {
		finish := start + step
		if finish > length {
			finish = length
		}

		blockBytes, err := rsa.EncryptOAEP(hash, rand.Reader, &p.privateKey.PublicKey, source[start:finish], []byte(p.label))
		if err != nil {
			return "", err
		}

		bytes = append(bytes, blockBytes...)
	}

	return hex.EncodeToString(bytes), nil
}

// DecryptKey decrypts data
func (p Provider) DecryptData(source string) ([]byte, error) {
	bytes, err := hex.DecodeString(source)
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	length := len(bytes)
	step := p.privateKey.PublicKey.Size()

	if step <= 0 {
		return nil, errors.New("bad step value")
	}

	var decryptedBytes []byte
	for start := 0; start < length; start += step {
		finish := start + step
		if finish > length {
			finish = length
		}

		blockBytes, err := rsa.DecryptOAEP(hash, rand.Reader, p.privateKey, bytes[start:finish], []byte(p.label))
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, blockBytes...)
	}

	return decryptedBytes, nil
}
