.PHONY: test cover

CHART_NAME=api-server
BIN_DIR ?= $(HOME)/bin
USER ?= $(USER)
NAMESPACE ?= "testkube"
DATE ?= $(shell date '+%s')
COMMIT ?= $(shell git log -1 --pretty=format:"%h")
VERSION ?= 999.0.0-$(shell git log -1 --pretty=format:"%h")
DEBUG ?= ${DEBUG:-0}
DASHBOARD_URI ?= ${DASHBOARD_URI:-"https://demo.testkube.io"}
ANALYTICS_TRACKING_ID = ${ANALYTICS_TRACKING_ID:-""}
ANALYTICS_API_KEY = ${ANALYTICS_API_KEY:-""}
PROTOC := ${BIN_DIR}/protoc/bin/protoc
PROTOC_GEN_GO := ${BIN_DIR}/protoc-gen-go
PROTOC_GEN_GO_GRPC := ${BIN_DIR}/protoc-gen-go-grpc
LD_FLAGS += -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=$(SLACK_BOT_CLIENT_ID)
LD_FLAGS += -X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=$(SLACK_BOT_CLIENT_SECRET)
LD_FLAGS += -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=$(ANALYTICS_TRACKING_ID)
LD_FLAGS += -X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=$(ANALYTICS_API_KEY)
LD_FLAGS += -X github.com/kubeshop/testkube/internal/pkg/api.Version=$(VERSION)
LD_FLAGS += -X github.com/kubeshop/testkube/internal/pkg/api.Commit=$(COMMIT)

define setup_env
	$(eval include .env)
	$(eval export)
endef

use-env-file:
	$(call setup_env)

.PHONY: refresh-config
refresh-config:
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/executors.json" -O config/executors.json &
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/job-container-template.yml" -O config/job-container-template.yml &
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/job-scraper-template.yml" -O config/job-scraper-template.yml &
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/job-template.yml" -O config/job-template.yml &
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/pvc-template.yml" -O config/pvc-template.yml &
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/slack-config.json" -O config/slack-config.json &
	wget "https://raw.githubusercontent.com/kubeshop/helm-charts/develop/charts/testkube-api/slack-template.json" -O config/slack-template.json


generate-protobuf: use-env-file 
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/logs/pb/logs.proto

just-run-api: use-env-file 
	TESTKUBE_DASHBOARD_URI=$(DASHBOARD_URI) APISERVER_CONFIG=testkube-api-server-config-testkube TESTKUBE_ANALYTICS_ENABLED=$(TESTKUBE_ANALYTICS_ENABLED) TESTKUBE_NAMESPACE=$(NAMESPACE) SCRAPPERENABLED=true STORAGE_SSL=true DEBUG=$(DEBUG) APISERVER_PORT=8088 go run  -ldflags='$(LD_FLAGS)' cmd/api-server/main.go

run-api: use-env-file refresh-config
	TESTKUBE_DASHBOARD_URI=$(DASHBOARD_URI) APISERVER_CONFIG=testkube-api-server-config-testkube TESTKUBE_ANALYTICS_ENABLED=$(TESTKUBE_ANALYTICS_ENABLED) TESTKUBE_NAMESPACE=$(NAMESPACE) SCRAPPERENABLED=true STORAGE_SSL=true DEBUG=$(DEBUG) APISERVER_PORT=8088 go run  -ldflags='$(LD_FLAGS)' cmd/api-server/main.go

run-api-race-detector: use-env-file refresh-config
	TESTKUBE_DASHBOARD_URI=$(DASHBOARD_URI) APISERVER_CONFIG=testkube-api-server-config-testkube TESTKUBE_NAMESPACE=$(NAMESPACE) DEBUG=1 APISERVER_PORT=8088 go run -race -ldflags='$(LD_FLAGS)'  cmd/api-server/main.go

run-api-telepresence: use-env-file refresh-config
	TESTKUBE_DASHBOARD_URI=$(DASHBOARD_URI) APISERVER_CONFIG=testkube-api-server-config-testkube TESTKUBE_NAMESPACE=$(NAMESPACE) DEBUG=1 API_MONGO_DSN=mongodb://testkube-mongodb:27017 APISERVER_PORT=8088 go run cmd/api-server/main.go

run-mongo-dev:
	docker run --name mongodb -p 27017:27017 --rm mongo

build: build-api-server build-testkube-bin

build-api-server:
	go build -o $(BIN_DIR)/api-server -ldflags='$(LD_FLAGS)' cmd/api-server/main.go

build-testkube-bin:
	go build \
		-ldflags="-s -w -X main.version=999.0.0-$(COMMIT) \
			-X main.commit=$(COMMIT) \
			-X main.date=$(DATE) \
			-X main.builtBy=$(USER) \
			-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=$(SLACK_BOT_CLIENT_ID) \
			-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=$(SLACK_BOT_CLIENT_SECRET) \
			-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=$(ANALYTICS_TRACKING_ID)  \
			-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=$(ANALYTICS_API_KEY)" \
		-o "$(BIN_DIR)/kubectl-testkube" \
		cmd/kubectl-testkube/main.go

build-testkube-bin-intel:
	env GOARCH=amd64 \
	go build \
		-ldflags="-s -w -X main.version=999.0.0-$(COMMIT) \
			-X main.commit=$(COMMIT) \
			-X main.date=$(DATE) \
			-X main.builtBy=$(USER) \
			-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientID=$(SLACK_BOT_CLIENT_ID) \
			-X github.com/kubeshop/testkube/internal/app/api/v1.SlackBotClientSecret=$(SLACK_BOT_CLIENT_SECRET) \
			-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementID=$(ANALYTICS_TRACKING_ID)  \
			-X github.com/kubeshop/testkube/pkg/telemetry.TestkubeMeasurementSecret=$(ANALYTICS_API_KEY)" \
		-o "$(BIN_DIR)/kubectl-testkube" \
		cmd/kubectl-testkube/main.go

docker-build-api:
	env SLACK_BOT_CLIENT_ID=** SLACK_BOT_CLIENT_SECRET=** ANALYTICS_TRACKING_ID=** ANALYTICS_API_KEY=** SEGMENTIO_KEY=** CLOUD_SEGMENTIO_KEY=** DOCKER_BUILDX_CACHE_FROM=type=registry,ref=docker.io/kubeshop/testkube-api-server:latest ALPINE_IMAGE=alpine:3.18.0 goreleaser release -f goreleaser_files/.goreleaser-docker-build-api.yml --rm-dist --snapshot

docker-build-cli:
	env SLACK_BOT_CLIENT_ID=** SLACK_BOT_CLIENT_SECRET=** ANALYTICS_TRACKING_ID=** ANALYTICS_API_KEY=** SEGMENTIO_KEY=** CLOUD_SEGMENTIO_KEY=** DOCKER_BUILDX_CACHE_FROM=type=registry,ref=docker.io/kubeshop/testkube-cli:latest ALPINE_IMAGE=alpine:3.18.0 goreleaser release -f .builds-linux.goreleaser.yml --rm-dist --snapshot

#make docker-build-executor EXECUTOR=zap GITHUB_TOKEN=*** DOCKER_BUILDX_CACHE_FROM=type=registry,ref=docker.io/kubeshop/testkube-zap-executor:latest
#add ALPINE_IMAGE=alpine:3.18.0 env var for building of curl and scraper executor
docker-build-executor:
	goreleaser release -f goreleaser_files/.goreleaser-docker-build-executor.yml --clean --snapshot

dev-install-local-executors:
	kubectl apply --namespace testkube -f https://raw.githubusercontent.com/kubeshop/testkube-operator/main/config/samples/executor_v1_executor.yaml

install-swagger-codegen-mac:
	brew install swagger-codegen

openapi-generate-model: openapi-generate-model-testkube

openapi-generate-model-testkube:
	swagger-codegen generate --model-package testkube -i api/v1/testkube.yaml -l go -o tmp/api/testkube
	mv tmp/api/testkube/model_test.go tmp/api/testkube/model_test_base.go || true
	mv tmp/api/testkube/model_test_suite_step_execute_test.go tmp/api/testkube/model_test_suite_step_execute_test_base.go || true
	mv tmp/api/testkube/model_*.go pkg/api/v1/testkube/
	rm -rf tmp
	find ./pkg/api/v1/testkube -type f -exec sed -i '' -e "s/package swagger/package testkube/g" {} \;
	find ./pkg/api/v1/testkube -type f -exec sed -i '' -e "s/\*map\[string\]/map[string]/g" {} \;
	find ./pkg/api/v1/testkube -name "*.go" -type f -exec sed -i '' -e "s/ map\[string\]Object / map\[string\]interface\{\} /g" {} \; # support map with empty additional properties
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ map/ \*map/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ string/ \*string/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \[\]/ \*\[\]/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*TestContent/ \*\*TestContentUpdate/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*ExecutionRequest/ \*\*ExecutionUpdateRequest/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*Repository/ \*\*RepositoryUpdate/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*SecretRef/ \*\*SecretRef/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ int32/ \*int32/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ int64/ \*int64/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ bool/ \*bool/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*ArtifactRequest/ \*\*ArtifactUpdateRequest/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*TestSuiteExecutionRequest/ \*\*TestSuiteExecutionUpdateRequest/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*ExecutorMeta/ \*\*ExecutorMetaUpdate/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*PodRequest/ \*\*PodUpdateRequest/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*PodResourcesRequest/ \*\*PodResourcesUpdateRequest/g" {} \;
	find ./pkg/api/v1/testkube -name "*update*.go" -type f -exec sed -i '' -e "s/ \*ResourceRequest/ \*ResourceUpdateRequest/g" {} \;	
	find ./pkg/api/v1/testkube -type f -exec sed -i '' -e "s/ Deprecated/ \\n\/\/ Deprecated/g" {} \;
	go fmt pkg/api/v1/testkube/*.go

protobuf-generate:
	$(PROTOC) \
    --go_out=. --go-grpc_out=. proto/service.proto

install-protobuf: $(PROTOC) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
# Protoc and friends installation and generation
$(PROTOC):
	$(call install-protoc)

$(PROTOC_GEN_GO):
	@echo "[INFO]: Installing protobuf go generation plugin."
	GOBIN=${BIN_DIR} go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28

$(PROTOC_GEN_GO_GRPC):
	@echo "[INFO]: Installing protobuf GRPC go generation plugin."
	GOBIN=${BIN_DIR} go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

.PHONY: unit-tests
unit-tests:
	gotestsum --format pkgname -- -cover ./...

.PHONY: integration-tests
integration-tests:
	INTEGRATION="true" gotestsum --format pkgname -- -tags=integration -cover ./...

test-e2e:
	go test --tags=e2e -v ./test/e2e

test-e2e-namespace:
	NAMESPACE=$(NAMESPACE) go test --tags=e2e -v  ./test/e2e

create-examples:
	test/create.sh

execute-testkube-cli-test-suite:
	test/run.sh

test-reload-sanity-test:
	kubectl delete secrets sanity-secrets -ntestkube
	kubectl delete test sanity -ntestkube || true
	kubectl testkube create test -f test/postman/Testkube-Sanity.postman_collection.json --name sanity


# test local api server intance - need local-postman/collection type registered to local postman executor
test-api-local:
	newman run test/postman/Testkube-Sanity.postman_collection.json --env-var test_name=fill-me --env-var test_type=postman/collection  --env-var api_uri=http://localhost:8088 --env-var test_api_uri=http://localhost:8088 --env-var execution_name=fill --verbose

# run by newman but on top of port-forwarded cluster service to api-server
# e.g. kubectl port-forward svc/testkube-api-server 8088
test-api-port-forwarded:
	newman run test/postman/Testkube-Sanity.postman_collection.json --env-var test_name=fill-me --env-var test_type=postman/collection  --env-var api_uri=http://localhost:8088 --env-var execution_name=fill --env-var test_api_uri=http://testkube-api-server:8088 --verbose

# run test by testkube plugin
test-api-on-cluster:
	kubectl testkube run test sanity -f -p api_uri=http://testkube-api-server:8088 -p test_api_uri=http://testkube-api-server:8088 -p test_type=postman/collection -p test_name=fill-me -p execution_name=fill-me

cover:
	@go test -failfast -count=1 -v -tags test  -coverprofile=./testCoverage.txt ./... && go tool cover -html=./testCoverage.txt -o testCoverage.html && rm ./testCoverage.txt
	open testCoverage.html

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
	go run cmd/kubectl-testkube/main.go generate doc

.PHONY: docs
docs: commands-reference

prerelease:
	go run cmd/tools/main.go release -d -a $(CHART_NAME)

release:
	go run cmd/tools/main.go release -a $(CHART_NAME)

video:
	gource \
		-s .5 \
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
		--output-ppm-stream stream.out \
		--output-framerate 30

	ffmpeg -y -r 30 -f image2pipe -vcodec ppm -i stream.out -b 65536K movie.mp4

port-forward-minio:
	kubectl port-forward svc/testkube-minio-service-testkube 9090:9090 -ntestkube

port-forward-mongo:
	kubectl port-forward svc/testkube-mongodb 27017 -ntestkube

port-forward-api:
	kubectl port-forward svc/testkube-api-server 8088 -ntestkube

run-proxy:
	go run cmd/proxy/main.go --namespace $(NAMESPACE)

define install-protoc
@[ -f "${PROTOC}" ] || { \
set -e ;\
echo "[INFO] Installing protoc compiler to ${BIN_DIR}/protoc" ;\
mkdir -pv "${BIN_DIR}/" ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
PB_REL="https://github.com/protocolbuffers/protobuf/releases" ;\
VERSION=3.19.4 ;\
if [ "$$(uname)" == "Darwin" ];then FILENAME=protoc-$${VERSION}-osx-x86_64.zip ;fi ;\
if [ "$$(uname)" == "Linux" ];then FILENAME=protoc-$${VERSION}-linux-x86_64.zip;fi ;\
echo "Downloading $${FILENAME} to $${TMP_DIR}" ;\
curl -LO $${PB_REL}/download/v$${VERSION}/$${FILENAME} ; unzip $${FILENAME} -d ${BIN_DIR}/protoc ; \
rm -rf $${TMP_DIR} ;\
}
endef
