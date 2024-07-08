package constants

// Internal variables

const (
	EnvNodeName           = "TKI_N"
	EnvPodName            = "TKI_P"
	EnvNamespaceName      = "TKI_S"
	EnvServiceAccountName = "TKI_A"
	EnvInstructions       = "TKI_I"
)

// Run arguments

const ()

const (
	ArgSeparator      = "--"
	ArgInit           = "-i"
	ArgInitLong       = "--init"
	ArgCondition      = "-c"
	ArgConditionLong  = "--cond"
	ArgResult         = "-r"
	ArgResultLong     = "--result"
	ArgTimeout        = "-t"
	ArgTimeoutLong    = "--timeout"
	ArgComputeEnv     = "-e"
	ArgComputeEnvLong = "--env"
	ArgNegative       = "-n"
	ArgNegativeLong   = "--negative"
	ArgPaused         = "-p"
	ArgPausedLong     = "--pause"
	ArgDebug          = "--debug"
	ArgWorkingDir     = "-w"
	ArgWorkingDirLong = "--workingDir"
	ArgToolkit        = "-k"
	ArgToolkitLong    = "--toolkit"
	ArgRetryUntil     = "--retryUntil" // TODO: Replace when multi-level retry will be there
	ArgRetryCount     = "--retryCount" // TODO: Replace when multi-level retry will be there
)
