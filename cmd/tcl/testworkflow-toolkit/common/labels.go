// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package common

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/pkg/ui"
)

func ServiceLabel(name string) string {
	return ui.LightCyan(name)
}

func InstanceLabel(name string, index int64, total int64) string {
	zeros := strings.Repeat("0", len(fmt.Sprintf("%d", total))-len(fmt.Sprintf("%d", index+1)))
	return ServiceLabel(name) + ui.Cyan(fmt.Sprintf("/%s%d", zeros, index+1))
}
