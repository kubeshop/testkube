package result

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type oldExecutionNumber struct {
	TestName string `json:"testName"`
	Number   int    `json:"number"`
}

type executionNumber struct {
	Name        string `json:"name"`
	Number      int    `json:"number"`
	IsTestSuite bool   `json:"isTestSuite"`
}

func (r *MongoRepository) GetNextExecutionNumber(ctx context.Context, name string) (number int32, err error) {
	err = r.convertFromOldToNew()
	if err != nil {
		return 1, err
	}

	// TODO: modify this when we decide to update the interfaces for OSS and cloud
	isTestSuite := strings.HasPrefix(name, "ts-")

	execNmbr := executionNumber{Name: name, IsTestSuite: isTestSuite}
	retry := false
	retryAttempts := 0
	maxRetries := 10

	opts := options.FindOneAndUpdate()
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)

	err = r.SequencesColl.FindOne(ctx, bson.M{"name": name}).Decode(&execNmbr)
	if err != nil {
		var execution testkube.Execution
		number, _ = r.GetLatestTestNumber(ctx, name)
		if number == 0 {
			execNmbr.Number = 1
		} else {
			execNmbr.Number = int(execution.Number) + 1
		}
		_, err = r.SequencesColl.InsertOne(ctx, execNmbr)
	} else {
		err = r.SequencesColl.FindOneAndUpdate(ctx, bson.M{"name": name}, bson.M{"$inc": bson.M{"number": 1}}, opts).Decode(&execNmbr)
	}

	retry = err != nil

	for retry {
		retryAttempts++
		err = r.SequencesColl.FindOneAndUpdate(ctx, bson.M{"name": name}, bson.M{"$inc": bson.M{"number": 1}}, opts).Decode(&execNmbr)
		if err == nil || retryAttempts >= maxRetries {
			retry = false
		}
	}

	return int32(execNmbr.Number), nil
}

func (r *MongoRepository) DeleteExecutionNumber(ctx context.Context, name string) (err error) {
	err = r.convertFromOldToNew()
	if err != nil {
		return err
	}
	_, err = r.SequencesColl.DeleteOne(ctx, bson.M{"name": name})
	return err
}

func (r *MongoRepository) DeleteExecutionNumbers(ctx context.Context, names []string) (err error) {
	err = r.convertFromOldToNew()
	if err != nil {
		return err
	}
	_, err = r.SequencesColl.DeleteMany(ctx, bson.M{"name": bson.M{"$in": names}})
	return err
}

func (r *MongoRepository) DeleteAllExecutionNumbers(ctx context.Context, isTestSuite bool) (err error) {
	err = r.convertFromOldToNew()
	if err != nil {
		return err
	}
	_, err = r.SequencesColl.DeleteMany(ctx, bson.M{"istestsuite": isTestSuite})
	return err
}

func (r *MongoRepository) convertFromOldToNew() error {
	filter := bson.M{"testname": bson.M{"$exists": true}}

	cursor, err := r.SequencesColl.Find(context.Background(), filter)
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var entry oldExecutionNumber
		err := cursor.Decode(&entry)
		if err != nil {
			return err
		}

		isTestSuite := strings.HasPrefix(entry.TestName, "ts-")

		newEntry := executionNumber{
			Name:        entry.TestName,
			Number:      entry.Number,
			IsTestSuite: isTestSuite,
		}

		_, err = r.SequencesColl.InsertOne(context.Background(), newEntry)
		if err != nil {
			return err
		}
	}
	_, err = r.SequencesColl.DeleteMany(context.Background(), filter)
	return err
}
