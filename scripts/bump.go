package main

import (
	"fmt"

	"github.com/kubeshop/kubetest/pkg/version"
)

func main() {
	fmt.Println(version.Next("0.0.2", version.Major))
	fmt.Println(version.Next("0.0.2", version.Minor))
	fmt.Println(version.Next("0.0.2", version.Patch))
}
