.PHONY: test cover 

CHART_NAME=api-server
BIN_DIR ?= $(HOME)/bin
USER ?= $(USER)
NAMESPACE ?= "testkube"
DATE ?= $(shell date -u --iso-8601=seconds)
COMMIT ?= $(shell git log -1 --pretty=format:"%h")
VERSION ?= 0.0.0-$(shell git log -1 --pretty=format:"%h")
LD_FLAGS += -X github.com/kubeshop/testkube/pkg/telemetry.telemetryToken=$(TELEMETRY_TOKEN)

run-api: 
	SCRAPPERENABLED=true STORAGE_SSL=true DEBUG=1 APISERVER_PORT=8088 go run -ldflags "-X github.com/kubeshop/testkube/internal/pkg/api.Version=$(VERSION) -X github.com/kubeshop/testkube/internal/pkg/api.Commit=$(COMMIT)"  cmd/api-server/main.go 

run-api-race-detector: 
	DEBUG=1 APISERVER_PORT=8088 go run -race -ldflags "-X github.com/kubeshop/testkube/internal/pkg/api.Version=$(VERSION) -X github.com/kubeshop/testkube/internal/pkg/api.Commit=$(COMMIT)"  cmd/api-server/main.go 

run-api-telepresence: 
	DEBUG=1 API_MONGO_DSN=mongodb://testkube-mongodb:27017 APISERVER_PORT=8088 go run cmd/api-server/main.go

run-mongo-dev: 
	docker run --name mongodb -p 27017:27017 --rm mongo


build: build-api-server build-testkube-bin

build-api-server:
	go build -o $(BIN_DIR)/api-server -ldflags='$(LD_FLAGS)' cmd/api-server/main.go 

build-testkube-bin: 
	go build -ldflags="-s -w -X main.version=0.0.0-$(COMMIT) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(USER) -X github.com/kubeshop/testkube/pkg/telemetry.telemetryToken=$(TELEMETRY_TOKEN)" -o "$(BIN_DIR)/kubectl-testkube" cmd/kubectl-testkube/main.go

build-testkube-bin-intel: 
	env GOARCH=amd64 go build -ldflags="-s -w -X main.version=0.0.0-$(COMMIT) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(USER) -X github.com/kubeshop/testkube/pkg/telemetry.telemetryToken=$(TELEMETRY_TOKEN)" -o "$(BIN_DIR)/kubectl-testkube" cmd/kubectl-testkube/main.go

docker-build-api:
	docker build -t api-server -f build/api-server/Dockerfile .

dev-install-local-executors:
	kubectl apply --namespace testkube -f https://raw.githubusercontent.com/kubeshop/testkube-operator/main/config/samples/executor_v1_executor.yaml

install-swagger-codegen-mac: 
	brew install swagger-codegen

openapi-generate-model: openapi-generate-model-testkube 

# we need to remove model_test_step to mimic inheritance 
# look at https://github.com/swagger-api/swagger-codegen/issues/11292
openapi-generate-model-testkube:
	swagger-codegen generate -i api/v1/testkube.yaml -l go -o tmp/api/testkube
	mv tmp/api/testkube/model_test.go tmp/api/testkube/model_test_base.go
	mv tmp/api/testkube/model_*.go pkg/api/v1/testkube/
	rm -rf tmp
	find ./pkg/api/v1/testkube -type f -exec sed -i '' -e "s/package swagger/package testkube/g" {} \;
	go fmt pkg/api/v1/testkube/*.go
	

test: 
	go test ./... -cover -failfast

test-e2e:
	go test --tags=e2e -v ./test/e2e

test-integration:
	go test -failfast --tags=integration -v ./...


test-e2e-namespace:
	NAMESPACE=$(NAMESPACE) go test --tags=e2e -v  ./test/e2e 

create-examples:
	kubectl delete script testkube-dashboard -ntestkube || true
	kubectl testkube scripts create --uri https://github.com/kubeshop/testkube-dashboard.git --git-path test --git-branch main --name testkube-dashboard  --type cypress/project
	kubectl delete script testkube-todo-frontend -ntestkube || true
	kubectl testkube scripts create --git-branch main --uri https://github.com/kubeshop/testkube-example-cypress-project.git --git-path "cypress" --name testkube-todo-frontend --type cypress/project
	kubectl delete script testkube-todo-api -ntestkube || true
	kubectl testkube scripts create --file test/e2e/TODO.postman_collection.json --name testkube-todo-api
	kubectl delete script kubeshop-site -ntestkube || true
	kubectl testkube scripts create --file test/e2e/Kubeshop.postman_collection.json --name kubeshop-site 
	kubectl delete test testkube-global-test -ntestkube || true
	cat test/e2e/test-example-1.json | kubectl testkube tests create --name testkube-global-test
	kubectl delete test kubeshop-sites-test -ntestkube || true
	cat test/e2e/test-example-2.json | kubectl testkube tests create --name kubeshop-sites-test


test-reload-sanity-script:
	kubectl delete script sanity -ntestkube || true
	kubectl testkube scripts create -f test/e2e/TestKube-Sanity.postman_collection.json --name sanity


# test local api server intance - need local-postman/collection type registered to local postman executor
test-api-local:
	newman run test/e2e/TestKube-Sanity.postman_collection.json --env-var script_name=fill-me --env-var script_type=postman/collection  --env-var api_uri=http://localhost:8088 --env-var script_api_uri=http://localhost:8088 --env-var execution_name=fill --verbose

# run by newman but on top of port-forwarded cluster service to api-server 
# e.g. kubectl port-forward svc/testkube-api-server 8088
test-api-port-forwarded:
	newman run test/e2e/TestKube-Sanity.postman_collection.json --env-var script_name=fill-me --env-var script_type=postman/collection  --env-var api_uri=http://localhost:8088 --env-var execution_name=fill --env-var script_api_uri=http://testkube-api-server:8088 --verbose

# run script by testkube plugin
test-api-on-cluster: 
	kubectl testkube scripts start sanity -f -p api_uri=http://testkube-api-server:8088 -p script_api_uri=http://testkube-api-server:8088 -p script_type=postman/collection -p script_name=fill-me -p execution_name=fill-me


cover: 
	@go test -failfast -count=1 -v -tags test  -coverprofile=./testCoverage.txt ./... && go tool cover -html=./testCoverage.txt -o testCoverage.html && rm ./testCoverage.txt 
	open testCoverage.html

diagrams: 
	plantuml ./docs/puml/*.puml -o ../img/

version-bump: version-bump-patch

version-bump-patch:
	go run cmd/tools/main.go bump -k patch

version-bump-minor:
	go run cmd/tools/main.go bump -k minor

version-bump-major:
	go run cmd/tools/main.go bump -k major

version-bump-dev:
	go run cmd/tools/main.go bump --dev

commands-reference: 
	go run cmd/kubectl-testkube/main.go doc > ./docs/reference.md

prerelease: 
	go run cmd/tools/main.go release -d -a $(CHART_NAME)

release: 
	go run cmd/tools/main.go release -a $(CHART_NAME)

video: 
	gource \    -s .5 \
		-1280x720 \
		--auto-skip-seconds .1 \
		--multi-sampling \
		--stop-at-end \
		--key \
		--highlight-users \
		--date-format "%d/%m/%y" \
		--hide mouse,filenames \
		--file-idle-time 0 \
		--max-files 0  \
		--background-colour 000000 \
		--font-size 25 \
		--output-ppm-stream - \
		--output-framerate 30 \
		| ffmpeg -y -r 30 -f image2pipe -vcodec ppm -i - -b 65536K movie.mp4


port-forward-minio:
	kubectl port-forward svc/testkube-minio-service-testkube 9090:9090 -ntestkube 

port-forward-mongo: 
	kubectl port-forward svc/testkube-mongodb 27017 -ntestkube

port-forward-api: 
	kubectl port-forward svc/testkube-api-server 8088 -ntestkube