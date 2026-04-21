package config

type ContextType string

const (
	ContextTypeCloud      ContextType = "cloud"
	ContextTypeKubeconfig ContextType = "kubeconfig"

	TokenTypeOIDC      = "oidc"
	TokenTypeAPI       = "api"
	TokenTypeEmailLink = "emailLink"

	CallbackPort            = 8090
	AlternativeCallbackPort = 38090

	// DatabaseTypeMongoDB is the identifier for MongoDB as the active database.
	DatabaseTypeMongoDB = "mongodb"
	// DatabaseTypePostgreSQL is the identifier for PostgreSQL as the active database.
	DatabaseTypePostgreSQL = "postgresql"
)

type CloudContext struct {
	EnvironmentId       string `json:"environment,omitempty"`
	EnvironmentName     string `json:"environmentName,omitempty"`
	OrganizationId      string `json:"organization,omitempty"`
	OrganizationName    string `json:"organizationName,omitempty"`
	ApiKey              string `json:"apiKey,omitempty"`
	RefreshToken        string `json:"refreshToken,omitempty"`
	ApiUri              string `json:"apiUri,omitempty"`
	AgentKey            string `json:"agentKey,omitempty"`
	AgentUri            string `json:"agentUri,omitempty"`
	RootDomain          string `json:"rootDomain,omitempty"`
	UiUri               string `json:"uiUri,omitempty"`
	AuthUri             string `json:"authUri,omitempty"`
	TokenType           string `json:"tokenType,omitempty"`
	DockerContainerName string `json:"dockerContainerName,omitempty"`
	CustomAuth          bool   `json:"customConnector,omitempty"`
	CallbackPort        int    `json:"callbackPort,omitempty"`
	// DatabaseType records which database (mongodb or postgresql) was active before
	// connecting to Pro, so it can be restored on disconnect.
	DatabaseType string `json:"databaseType,omitempty"`
	// AgentReleaseName is the Helm release name of the runner chart installed by "pro connect".
	AgentReleaseName string `json:"agentReleaseName,omitempty"`
	// AgentNamespace is the Kubernetes namespace where the runner chart was installed by "pro connect".
	AgentNamespace string `json:"agentNamespace,omitempty"`
	// AgentName is the name of the agent record created in the control plane by "pro connect".
	AgentName string `json:"agentName,omitempty"`
}

type Data struct {
	TelemetryEnabled bool              `json:"telemetryEnabled,omitempty"`
	Namespace        string            `json:"namespace,omitempty"`
	Initialized      bool              `json:"initialized,omitempty"`
	APIURI           string            `json:"apiURI,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	APIServerName    string            `json:"apiServerName,omitempty"`
	APIServerPort    int               `json:"apiServerPort,omitempty"`
	DashboardName    string            `json:"dashboardName,omitempty"`
	DashboardPort    int               `json:"dashboardPort,omitempty"`

	ContextType  ContextType  `json:"contextType,omitempty"`
	CloudContext CloudContext `json:"cloudContext,omitempty"`
	Master       Master       `json:"master,omitempty"`
}

func (c *Data) EnableAnalytics() {
	c.TelemetryEnabled = true
}

func (c *Data) DisableAnalytics() {
	c.TelemetryEnabled = false
}

func (c *Data) SetNamespace(ns string) {
	c.Namespace = ns
}

func (c *Data) SetInitialized() {
	c.Initialized = true
}
