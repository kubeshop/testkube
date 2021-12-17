package testkube

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewStartedTestExecution(name string) TestExecution {
	return TestExecution{
		Id:        primitive.NewObjectID().Hex(),
		StartTime: time.Now(),
		Name:      name,
		Status:    TestStatusQueued,
	}
}
