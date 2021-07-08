module github.com/kubeshop/kubetest

go 1.16

require (
	github.com/gofiber/adaptor/v2 v2.1.7
	github.com/gofiber/fiber/v2 v2.14.0
	github.com/kubeshop/kubetest/internal/app/operator v0.0.0-20210706105244-914c8d666591
	github.com/moogar0880/problems v0.1.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	go.mongodb.org/mongo-driver v1.5.4
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)
