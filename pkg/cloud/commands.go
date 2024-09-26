package cloud

// Command is a type for cloud commands
// passed in centralized anvironments instead of HTTP agetn API Proxy requests
type Command string

const (
	ExecuteCommand Command = "execute"
	// TODO check if this is even needed anymore
	HealthcheckCommand Command = "healthcheck"
	LogsCommand        Command = "logs"
)
