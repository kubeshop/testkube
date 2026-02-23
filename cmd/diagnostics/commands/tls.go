package commands

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewTLSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Check DNS entry",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				ui.Failf("Please pass domain name")
			}
			host := args[0]
			checkTLS(host)
		},
	}

	return cmd
}

func checkTLS(host string) {
	// Test connection with modern configuration
	conf := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	fmt.Printf("TLS Diagnostic Results for %s\n", host)
	fmt.Println("=====================================")

	// Connect and get certificate info
	conn, err := tls.Dial("tcp", host, conf)
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer conn.Close()

	// Get certificate details
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) > 0 {
		cert := certs[0]
		fmt.Println("\nCertificate Details:")
		fmt.Printf("Subject: %s\n", cert.Subject)
		fmt.Printf("Issuer: %s\n", cert.Issuer)
		fmt.Printf("Valid from: %s\n", cert.NotBefore)
		fmt.Printf("Valid until: %s\n", cert.NotAfter)
		fmt.Printf("Expiring in: %d days\n", int(cert.NotAfter.Sub(time.Now()).Hours()/24))
		fmt.Printf("Serial number: %x\n", cert.SerialNumber)
		fmt.Printf("Version: %d\n", cert.Version)

		// Check for domain names
		fmt.Println("\nSAN (Subject Alternative Names):")
		for _, dns := range cert.DNSNames {
			fmt.Printf("- %s\n", dns)
		}
	}

	// Test supported TLS versions
	fmt.Println("\nTLS Version Support:")
	versions := map[uint16]string{
		tls.VersionTLS10: "TLS 1.0",
		tls.VersionTLS11: "TLS 1.1",
		tls.VersionTLS12: "TLS 1.2",
		tls.VersionTLS13: "TLS 1.3",
	}

	for version, name := range versions {
		testConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         version,
			MaxVersion:         version,
		}

		testConn, err := tls.Dial("tcp", host, testConfig)
		if err != nil {
			fmt.Printf("✗ %s: Not supported\n", name)
			continue
		}
		testConn.Close()
		fmt.Printf("✓ %s: Supported\n", name)
	}

	// Test cipher suites
	fmt.Println("\nSupported Cipher Suites:")
	state := conn.ConnectionState()
	fmt.Printf("Negotiated Cipher Suite: %s\n", tls.CipherSuiteName(state.CipherSuite))

	// Connection info
	fmt.Println("\nConnection Information:")
	fmt.Printf("Protocol: %s\n", versionToString(state.Version))
	fmt.Printf("Server Name: %s\n", state.ServerName)
	fmt.Printf("Handshake Complete: %v\n", state.HandshakeComplete)
	fmt.Printf("Mutual TLS: %v\n", state.NegotiatedProtocolIsMutual)

	// Basic security checks
	fmt.Println("\nSecurity Assessment:")
	if state.Version < tls.VersionTLS12 {
		fmt.Println("⚠️  Warning: Using TLS version below 1.2")
	} else {
		fmt.Println("✓ TLS version >= 1.2")
	}
}

func versionToString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}
