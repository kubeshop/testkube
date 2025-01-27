package config

type ProContextMode string

const (
	ProContextModeUnknown ProContextMode = ""

	ProContextModeEnterprise ProContextMode = "enterprise"
	// TODO: Use "pro" in the future when refactoring TK Pro API server to use "pro" instead of "cloud"
	ProContextModePro ProContextMode = "cloud"
)

// Ref: #/components/schemas/PlanStatus
type ProContextStatus string

const (
	ProContextStatusUnknown           ProContextStatus = ""
	ProContextStatusActive            ProContextStatus = "Active"
	ProContextStatusCanceled          ProContextStatus = "Canceled"
	ProContextStatusIncomplete        ProContextStatus = "Incomplete"
	ProContextStatusIncompleteExpired ProContextStatus = "IncompleteExpired"
	ProContextStatusPastDue           ProContextStatus = "PastDue"
	ProContextStatusTrailing          ProContextStatus = "Trailing"
	ProContextStatusUnpaid            ProContextStatus = "Unpaid"
	ProContextStatusDeleted           ProContextStatus = "Deleted"
	ProContextStatusLocked            ProContextStatus = "Locked"
	ProContextStatusBlocked           ProContextStatus = "Blocked"
)

type ProContext struct {
	APIKey                              string
	URL                                 string
	TLSInsecure                         bool
	WorkerCount                         int
	LogStreamWorkerCount                int
	SkipVerify                          bool
	AgentID                             string
	EnvID                               string
	OrgID                               string
	Migrate                             string
	ConnectionTimeout                   int
	DashboardURI                        string
	NewArchitecture                     bool
	CloudStorage                        bool
	CloudStorageSupportedInControlPlane bool
	IsTrial                             bool
	Mode                                ProContextMode
	Status                              ProContextStatus
}
