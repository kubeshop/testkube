package license

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/keygen-sh/jsonapi-go"
	"github.com/keygen-sh/keygen-go/v3"

	"github.com/kubeshop/testkube/pkg/diagnostics/validators"
)

// KeygenOfflinePublicKey will be set through build process when changed
var KeygenOfflinePublicKey = ""

type License struct {
	License      *keygen.License
	Entitlements keygen.Entitlements
	IsOffline    bool `json:"isOffline"`
}

func NewOfflineLicenseValidator(key, file string) OfflineLicenseValidator {
	return OfflineLicenseValidator{
		LicenseKey:  key,
		LicenseFile: file,
	}
}

type OfflineLicenseValidator struct {
	LicenseFile string
	LicenseKey  string
}

func (v OfflineLicenseValidator) Name() string {
	return "Offline license validation/signing check"
}

// Validate validates a given license key for format / length correctness without calling external services
func (v OfflineLicenseValidator) Validate(_ any) (r validators.ValidationResult) {
	l, err := v.ValidateOfflineLicenseCert(v.LicenseKey, v.LicenseFile)
	if l == nil {
		return r.WithError(err)
	}

	if l.License.Expiry.Before(time.Now()) {
		return r.WithError(ErrOfflineLicenseExpired.
			WithDetails(fmt.Sprintf("Your license file expired in %s, please renew your license", l.License.Expiry)))
	}

	left := time.Until(*l.License.Expiry)
	return r.WithValidStatus().WithAdditionalInfo(fmt.Sprintf("license is still valid %d days", int(left.Hours())/24))
}

func (v *OfflineLicenseValidator) ValidateOfflineLicenseCert(key string, file string) (l *License, e validators.Error) {
	if KeygenOfflinePublicKey == "" {
		key, ok := os.LookupEnv("KEYGEN_PUBLIC_KEY")
		if ok {
			KeygenOfflinePublicKey = key
		}
	}
	keygen.PublicKey = KeygenOfflinePublicKey
	keygen.LicenseKey = key

	// Verify the license file's signature
	lic := &keygen.LicenseFile{Certificate: file}
	err := lic.Verify()
	switch {
	case errors.Is(err, keygen.ErrPublicKeyMissing):
		return nil, ErrOfflineLicensePublicKeyMissing
	case errors.Is(err, keygen.ErrLicenseFileNotGenuine):
		return nil, ErrOfflineLicenseLicenseFileIsNotGenuine
	case err != nil:
		return nil, ErrOfflineLicenseVerificationInvalid.WithDetails(err.Error())
	}

	cert, err := certificate(lic)
	if err != nil {
		return nil, ErrOfflineLicenseCertificateInvalid.WithDetails(err.Error())
	}

	var dataset *keygen.LicenseFileDataset
	switch {
	case strings.HasPrefix(cert.Alg, "aes-256"):
		// The license key is used as the key for the symmetric encryption.
		dataset, err = lic.Decrypt(key)
	case strings.HasPrefix(cert.Alg, "base64"):
		dataset, err = decode(cert)
	}

	switch {
	case errors.Is(err, keygen.ErrSystemClockUnsynced):
		return nil, ErrOfflineLicenseClockTamperingDetected
	case errors.Is(err, keygen.ErrLicenseFileExpired):
		return nil, ErrOfflineLicenseFileExpired
	case err != nil:
		return nil, ErrOfflineLicenseDecodingError.WithDetails(err.Error())
	}

	if dataset == nil {
		return nil, ErrOfflineLicenseDatasetIsMissing
	}

	return &License{
		License:      &dataset.License,
		Entitlements: dataset.Entitlements,
		IsOffline:    true,
	}, e
}

func decode(cert *keygenCertificate) (*keygen.LicenseFileDataset, error) {
	if cert.Alg != "base64+ed25519" {
		return nil, keygen.ErrLicenseFileNotSupported
	}

	// continue here decode with base64 and json parse properly
	data, err := base64.StdEncoding.DecodeString(cert.Enc)
	if err != nil {
		return nil, &keygen.LicenseFileError{Err: err}
	}

	// Unmarshal
	dataset := &keygen.LicenseFileDataset{}

	if _, err := jsonapi.Unmarshal(data, dataset); err != nil {
		return nil, err
	}

	if dataset.TTL != 0 && time.Now().After(dataset.Expiry) {
		return dataset, keygen.ErrLicenseFileExpired
	}

	return dataset, nil
}

type keygenCertificate struct {
	Enc string `json:"enc"`
	Sig string `json:"sig"`
	Alg string `json:"alg"`
}

func certificate(lic *keygen.LicenseFile) (*keygenCertificate, error) {
	payload := strings.TrimSpace(lic.Certificate)

	// Remove header and footer
	payload = strings.TrimPrefix(payload, "-----BEGIN LICENSE FILE-----")
	payload = strings.TrimSuffix(payload, "-----END LICENSE FILE-----")
	payload = strings.TrimSpace(payload)

	// Decode
	dec, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, &keygen.LicenseFileError{Err: err}
	}

	// Unmarshal
	var cert *keygenCertificate
	if err := json.Unmarshal(dec, &cert); err != nil {
		return nil, &keygen.LicenseFileError{Err: err}
	}

	return cert, nil
}
