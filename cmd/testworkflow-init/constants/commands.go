package constants

const (
	EnvGroupActions   = "01"
	EnvGroupDebug     = "00"
	EnvGroupSecrets   = "02"
	EnvGroupInternal  = "03"
	EnvGroupResources = "04"
	EnvGroupRuntime   = "05"

	EnvNodeName               = "TKI_N"
	EnvPodName                = "TKI_P"
	EnvNamespaceName          = "TKI_S"
	EnvServiceAccountName     = "TKI_A"
	EnvActions                = "TKI_I"
	EnvInternalConfig         = "TKI_C"
	EnvSignature              = "TKI_G"
	EnvContainerName          = "TKI_O"
	EnvResourceRequestsCPU    = "TKI_R_R_C"
	EnvResourceLimitsCPU      = "TKI_R_L_C"
	EnvResourceRequestsMemory = "TKI_R_R_M"
	EnvResourceLimitsMemory   = "TKI_R_L_M"
)
