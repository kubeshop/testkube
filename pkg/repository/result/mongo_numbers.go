package result

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type executionNumber struct {
	TestName string `json:"testName"`
	Number   int    `json:"number"`
}

func (r *MongoRepository) GetNextExecutionNumber(ctx context.Context, testName string) (number int32, err error) {

	execNmbr := executionNumber{TestName: testName}
	retry := false
	retryAttempts := 0
	maxRetries := 10

	opts := options.FindOneAndUpdate()
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	err = r.SequencesColl.FindOne(ctx, bson.M{"testname": testName}).Decode(&execNmbr)
	if err != nil {
		var execution testkube.Execution
		execution, err = r.GetLatestByTest(ctx, testName, "number")
		if err != nil {
			execNmbr.Number = 1
		} else {
			execNmbr.Number = int(execution.Number) + 1
		}
		_, err = r.SequencesColl.InsertOne(ctx, execNmbr)
	} else {
		err = r.SequencesColl.FindOneAndUpdate(ctx, bson.M{"testname": testName}, bson.M{"$inc": bson.M{"number": 1}}, opts).Decode(&execNmbr)
	}

	retry = err != nil

	for retry {
		retryAttempts++
		err = r.SequencesColl.FindOneAndUpdate(ctx, bson.M{"testname": testName}, bson.M{"$inc": bson.M{"number": 1}}, opts).Decode(&execNmbr)
		if err == nil || retryAttempts >= maxRetries {
			retry = false
		}
	}

	return int32(execNmbr.Number), nil
}
