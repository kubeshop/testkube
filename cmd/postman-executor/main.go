package main

import (
	"github.com/kubeshop/kubetest/internal/app/postman"
	"github.com/kubeshop/kubetest/internal/pkg/postman/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/storage"
)

func main() {

	db, err := storage.GetMongoDataBase()
	if err != nil {
		panic(err)
	}

	executor := postman.NewPostmanExecutor(result.NewMongoRespository(db))
	executor.Init()
	panic(executor.Run())

}
