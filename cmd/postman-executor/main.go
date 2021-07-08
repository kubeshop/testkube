package main

import "github.com/kubeshop/kubetest/internal/app/postman"

func main() {

	executor := postman.NewPostmanExecutor()
	executor.Init()
	panic(executor.Run())

}
