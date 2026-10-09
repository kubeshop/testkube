package localinstall

import (
	"fmt"
	"strings"
)

// The chart labels its pods with this as their instance.
const ReleaseName = "testkube"

// The chart can't pin nodePorts for ui, api and ai.
var browserServices = map[string]struct {
	app, target string
	port        int
}{
	"dashboard": {"testkube-cloud-ui", "http", 8080},
	"api":       {"testkube-cloud-api", "http", 8090},
	"login":     {"dex", "http", 5556},
	"storage":   {"minio", "minio-api", 9000},
	"ai":        {"testkube-ai-service", "http", 9090},
}

func ServicesManifest() string {
	var docs []string
	for _, cp := range clusterPorts {
		s := browserServices[cp.name]
		docs = append(docs, fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: testkube-local-%s
  namespace: %s
spec:
  type: NodePort
  selector:
    app.kubernetes.io/name: %s
    app.kubernetes.io/instance: %s
  ports:
  - port: %d
    targetPort: %s
    nodePort: %d
`, cp.name, Namespace, s.app, ReleaseName, s.port, s.target, cp.inNodes))
	}
	return strings.Join(docs, "---\n")
}

// Browser URLs follow the host ports, which move when busy.
func PortValues(p Ports) string {
	api := fmt.Sprintf("http://localhost:%d", p["api"])
	return fmt.Sprintf(`global:
  dex:
    issuer: http://localhost:%[2]d
  storage:
    public:
      endpoint: localhost:%[3]d
testkube-cloud-api:
  additionalEnv:
    OAUTH_AUTH_URL: http://localhost:%[2]d/auth
  api:
    oauth:
      redirectUri: %[1]s/auth/callback
    apiAddress: %[1]s
    dashboardAddress: http://localhost:%[4]d
testkube-cloud-ui:
  ui:
    apiServerEndpoint: %[1]s
    wsServerEndpoint: ws://localhost:%[5]d
  ai:
    aiServiceApiUri: http://localhost:%[6]d
testkube-ai-service:
  dashboardUrl: http://localhost:%[4]d
postgresql:
  primary:
    service:
      # Its default NodePort could take one of our fixed ports.
      type: ClusterIP
dex:
  configTemplate:
    additionalStaticClients:
      - id: testkube-enterprise
        name: Testkube
        secret: QWkVzs3nct6HZM5hxsPzwaZtq
        redirectURIs:
          - %[1]s/auth/callback
          - %[1]s/mcp/auth/callback
      - id: testkube-cloud-cli
        name: Testkube Enterprise CLI
        public: true
        redirectURIs:
          - http://127.0.0.1:8090/callback
          - http://127.0.0.1:38090/callback
`, api, p["login"], p["storage"], p["dashboard"], p["api"], p["ai"])
}
