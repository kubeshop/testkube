// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package constants

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
	ArgRetryUntil     = "--retryUntil" // TODO: Replace when multi-level retry will be there
	ArgRetryCount     = "--retryCount" // TODO: Replace when multi-level retry will be there
)
