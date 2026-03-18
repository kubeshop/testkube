package github

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/skratchdot/open-golang/open"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
	"github.com/kubeshop/testkube/pkg/utils"
)

const (
	BaseURL  = "https://github.com/kubeshop/testkube/issues/new"
	BugType  = "bug 🐛"
	Template = `
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Run '...'
2. Specify '...'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Version / Cluster**
- Testkube CLI version: {{ .ClientVersion }}
- Testkube API server version: {{ .ServerVersion }}
- Kubernetes cluster version: {{ .ClusterVersion }}

**Screenshots**
If applicable, add CLI commands/output to help explain your problem.

**Additional context**
Add any other context about the problem here.

Attach the output of the **testkube debug info** command to provide more details.
`
)

// OpenTicket opens up a browser to create a Bug issue in the Testkube GitHub repository
func OpenTicket(d testkube.DebugInfo) error {
	title, body, err := buildTicket(d)
	if err != nil {
		return fmt.Errorf("could not build issue: %w", err)
	}

	openURL, err := buildIssueURL(BaseURL, title, body, []string{BugType})
	if err != nil {
		return err
	}

	if len(openURL) >= 8192 {
		return fmt.Errorf("cannot open in browser: maximum URL length exceeded")
	}

	ui.Info(fmt.Sprintf("Opening %s in your browser.\n", BaseURL))

	return open.Start(openURL)
}

// buildIssueURL constructs a GitHub new issue URL with the provided parameters.
func buildIssueURL(baseURL, title, body string, labels []string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if title != "" {
		q.Set("title", title)
	}
	q.Set("body", body)
	if len(labels) > 0 {
		q.Set("labels", strings.Join(labels, ","))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// buildTicket builds up the title and the body of the ticket, completing the version numbers with data from the environment
func buildTicket(d testkube.DebugInfo) (string, string, error) {
	if d.ClientVersion == "" || d.ClusterVersion == "" {
		return "", "", errors.New("client version and cluster version must be populated to create debug message")
	}
	t, err := utils.NewTemplate("debug").Parse(Template)
	if err != nil {
		return "", "", fmt.Errorf("cannot create template: %w", err)
	}

	var result bytes.Buffer
	err = t.Execute(&result, d)
	if err != nil {
		return "", "", fmt.Errorf("cannot parse template: %w", err)
	}

	return "New bug report", result.String(), nil
}
