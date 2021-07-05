package main

import v1API "github.com/kubeshop/kubetest/internal/app/api/v1"

func main() {
	v1API.NewServer().Run()
}
