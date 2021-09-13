module github.com/kubeshop/kubtest

go 1.16

// replace github.com/kubeshop/kubtest-operator v0.1.3 => ../kubtest-operator

require (
	github.com/Masterminds/semver v1.5.0
	github.com/bclicn/color v0.0.0-20180711051946-108f2023dc84
	github.com/davecgh/go-spew v1.1.1
	github.com/dustinkirkland/golang-petname v0.0.0-20191129215211-8e5a1ed0cff0
	github.com/gofiber/adaptor/v2 v2.1.7
	github.com/gofiber/fiber/v2 v2.14.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kubeshop/kubtest-operator v0.1.12
	github.com/moogar0880/problems v0.1.1
	github.com/olekukonko/tablewriter v0.0.0-20170122224234-a0225b3f23b5
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	go.mongodb.org/mongo-driver v1.5.4
	go.uber.org/zap v1.17.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
