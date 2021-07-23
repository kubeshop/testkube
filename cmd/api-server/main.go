package main

import (
	v1API "github.com/kubeshop/kubetest/internal/app/api/v1"
	"github.com/kubeshop/kubetest/internal/pkg/api/repository/result"
	"github.com/kubeshop/kubetest/internal/pkg/postman/storage"
)

func main() {
	db, err := storage.GetMongoDataBase()
	if err != nil {
		panic(err)
	}

	repository := result.NewMongoRespository(db)
	v1API.NewServer(repository).Run()
}
