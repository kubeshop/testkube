.PHONY: test cover 

BIN_DIR ?= $(HOME)/bin


run-api-serer: 
	go run cmd/api-server/main.go

run-executor: 
	go run cmd/postman-executor/main.go

run-mongo-dev: 
	docker run -p 27017:27017 mongo

# build done by vendoring to bypass private go repo problems
build-executor: 
	go mod vendor
	docker build -t postman-executor -f build/postman-executor/Dockerfile .

build-kubetest-bin: 
	go mod vendor
	go build -o "$(BIN_DIR)/kubectl-kubetest" cmd/kubectl-kubetest/main.go


install-swagger-codegen-mac: 
	brew install swagger-codegen

openapi-generate-model: openapi-generate-model-kubetest openapi-generate-model-executor

openapi-generate-model-kubetest:
	swagger-codegen generate -i api/v1/kubetest.yaml -l go -o tmp/api/kubetest
	mv tmp/api/kubetest/model_*.go pkg/api/kubetest
	rm -rf tmp
	find ./pkg/api/kubetest -type f -exec sed -i '' -e "s/package swagger/package kubetest/g" {} \;
	

openapi-generate-model-executor:
	swagger-codegen generate -i api/v1/executor.yaml -l go -o tmp/api/executor
	mv tmp/api/executor/model_*.go pkg/api/executor
	rm -rf tmp
	find ./pkg/api/executor -type f -exec sed -i '' -e "s/package swagger/package executor/g" {} \;

test: 
	go test ./... -cover

cover: 
	@go test -mod=vendor -failfast -count=1 -v -tags test  -coverprofile=./testCoverage.txt ./... && go tool cover -html=./testCoverage.txt -o testCoverage.html && rm ./testCoverage.txt 
	open testCoverage.html
