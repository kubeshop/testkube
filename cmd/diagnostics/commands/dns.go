package commands

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDnsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Check DNS entry",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				ui.Failf("Please pass domain name")
			}
			domain := args[0]
			analyzer := NewDNSAnalyzer(domain)
			results := analyzer.LookupAll(context.Background())
			analyzer.PrintResults(results)
		},
	}

	return cmd
}

type DNSResult struct {
	RecordType string
	Records    []string
	Duration   time.Duration
	Error      error
}

type DNSAnalyzer struct {
	domain   string
	resolver *net.Resolver
}

func NewDNSAnalyzer(domain string) *DNSAnalyzer {
	return &DNSAnalyzer{
		domain:   domain,
		resolver: net.DefaultResolver,
	}
}

func (da *DNSAnalyzer) LookupAll(ctx context.Context) map[string]DNSResult {
	results := make(map[string]DNSResult)

	// Perform A record lookup
	results["A"] = da.lookupA(ctx)

	// Perform NS record lookup
	results["NS"] = da.lookupNS(ctx)

	// Perform CNAME record lookup
	results["CNAME"] = da.lookupCNAME(ctx)

	return results
}

func (da *DNSAnalyzer) lookupA(ctx context.Context) DNSResult {
	start := time.Now()
	ips, err := da.resolver.LookupIP(ctx, da.domain, "ip4")
	duration := time.Since(start)

	var records []string
	for _, ip := range ips {
		records = append(records, ip.String())
	}

	return DNSResult{
		RecordType: "A",
		Records:    records,
		Duration:   duration,
		Error:      err,
	}
}

func (da *DNSAnalyzer) lookupNS(ctx context.Context) DNSResult {
	start := time.Now()
	nss, err := da.resolver.LookupNS(ctx, da.domain)
	duration := time.Since(start)

	var records []string
	for _, ns := range nss {
		records = append(records, ns.Host)
	}

	return DNSResult{
		RecordType: "NS",
		Records:    records,
		Duration:   duration,
		Error:      err,
	}
}

func (da *DNSAnalyzer) lookupCNAME(ctx context.Context) DNSResult {
	start := time.Now()
	cname, err := da.resolver.LookupCNAME(ctx, da.domain)
	duration := time.Since(start)

	var records []string
	if cname != "" {
		records = append(records, cname)
	}

	return DNSResult{
		RecordType: "CNAME",
		Records:    records,
		Duration:   duration,
		Error:      err,
	}
}

func (da *DNSAnalyzer) PrintResults(results map[string]DNSResult) {
	fmt.Printf("DNS Analysis Results for %s\n", da.domain)
	fmt.Println(strings.Repeat("-", 50))

	for recordType, result := range results {
		fmt.Printf("\n%s Records:\n", recordType)
		if result.Error != nil {
			fmt.Printf("  Error: %v\n", result.Error)
		} else if len(result.Records) == 0 {
			fmt.Println("  No records found")
		} else {
			for _, record := range result.Records {
				fmt.Printf("  %s\n", record)
			}
		}
		fmt.Printf("  Lookup Duration: %v\n", result.Duration)
	}
}
