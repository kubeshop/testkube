package result

import (
	"context"
	"io"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ExecutionOutput struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	TestName      string `json:"testname,omitempty"`
	TestSuiteName string `json:"testsuitename,omitempty"`
	Output        string `json:"output"`
}

const CollectionOutput = "output"

type MongoOutputRepository struct {
	Coll *mongo.Collection
}

var _ OutputRepository = (*MongoOutputRepository)(nil)

func NewMongoOutputRepository(db *mongo.Database, opts ...MongoOutputRepositoryOpt) *MongoOutputRepository {
	r := &MongoOutputRepository{
		Coll: db.Collection(CollectionOutput),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

type MongoOutputRepositoryOpt func(*MongoOutputRepository)

func WithMongoOutputRepositoryCollection(collection *mongo.Collection) MongoOutputRepositoryOpt {
	return func(r *MongoOutputRepository) {
		r.Coll = collection
	}
}

func (r *MongoOutputRepository) GetOutput(ctx context.Context, id, testName, testSuiteName string) (output string, err error) {
	var eOutput ExecutionOutput
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&eOutput)
	return eOutput.Output, err
}

func (r *MongoOutputRepository) InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	_, err := r.Coll.InsertOne(ctx, ExecutionOutput{Id: id, Name: id, TestName: testName, TestSuiteName: testSuiteName, Output: output})
	return err
}

func (r *MongoOutputRepository) UpdateOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	_, err := r.Coll.UpdateOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}, bson.M{"$set": bson.M{"output": output}})
	return err
}

func (r *MongoOutputRepository) DeleteOutput(ctx context.Context, id, testName, testSuiteName string) error {
	_, err := r.Coll.DeleteOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}})
	return err
}

func (r *MongoOutputRepository) DeleteOutputByTest(ctx context.Context, testName string) error {
	_, err := r.Coll.DeleteMany(ctx, bson.M{"testname": testName})
	return err
}

func (r *MongoOutputRepository) DeleteOutputForTests(ctx context.Context, testNames []string) error {
	_, err := r.Coll.DeleteMany(ctx, bson.M{"testname": bson.M{"$in": testNames}})
	return err
}

func (r *MongoOutputRepository) DeleteOutputByTestSuite(ctx context.Context, testSuiteName string) error {
	_, err := r.Coll.DeleteMany(ctx, bson.M{"testsuitename": testSuiteName})
	return err
}

func (r *MongoOutputRepository) DeleteOutputForTestSuites(ctx context.Context, testSuiteNames []string) error {
	_, err := r.Coll.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$in": testSuiteNames}})
	return err
}

func (r *MongoOutputRepository) DeleteOutputForAllTestSuite(ctx context.Context) error {
	_, err := r.Coll.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$ne": ""}})
	return err
}

func (r *MongoOutputRepository) DeleteAllOutput(ctx context.Context) error {
	_, err := r.Coll.DeleteMany(ctx, bson.M{})
	return err
}

func (r *MongoOutputRepository) StreamOutput(ctx context.Context, executionID, testName, testSuiteName string) (reader io.Reader, err error) {
	var eOutput ExecutionOutput
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": executionID}, bson.M{"name": executionID}}}).Decode(&eOutput)
	return strings.NewReader(eOutput.Output), err
}

func (r *MongoOutputRepository) GetOutputSize(ctx context.Context, executionID, testName, testSuiteName string) (size int, err error) {
	var eOutput ExecutionOutput
	err = r.Coll.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": executionID}, bson.M{"name": executionID}}}).Decode(&eOutput)
	return len(eOutput.Output), err
}
