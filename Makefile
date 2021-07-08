test-all: 
	go test -v -cover ./...

run-api-serer: 
	go run cmd/api-server/main.go



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
