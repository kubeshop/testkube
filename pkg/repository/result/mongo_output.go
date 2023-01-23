package result

import (
	"context"

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
	OutputColl *mongo.Collection
}

func NewMongoOutputRepository(db *mongo.Database, opts ...MongoOutputRepositoryOpt) *MongoOutputRepository {
	r := &MongoOutputRepository{
		OutputColl: db.Collection(CollectionOutput),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

type MongoOutputRepositoryOpt func(*MongoOutputRepository)

func WithMongoOutputRepositoryCollection(db *mongo.Database, collection string) MongoOutputRepositoryOpt {
	return func(r *MongoOutputRepository) {
		r.OutputColl = db.Collection(collection)
	}
}

func (r *MongoOutputRepository) GetOutput(ctx context.Context, id string) (output string, err error) {
	var eOutput ExecutionOutput
	err = r.OutputColl.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&eOutput)
	return eOutput.Output, err
}

func (m *MongoOutputRepository) InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	_, err := m.OutputColl.InsertOne(ctx, ExecutionOutput{Id: id, Name: id, TestName: testName, TestSuiteName: testSuiteName, Output: output})
	return err
}

func (m *MongoOutputRepository) UpdateOutput(ctx context.Context, id, output string) error {
	_, err := m.OutputColl.UpdateOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}, bson.M{"$set": bson.M{"output": output}})
	return err
}

func (m *MongoOutputRepository) DeleteOutput(ctx context.Context, id string) error {
	_, err := m.OutputColl.DeleteOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}})
	return err
}

func (m *MongoOutputRepository) DeleteOutputByTest(ctx context.Context, testName string) error {
	_, err := m.OutputColl.DeleteMany(ctx, bson.M{"testname": testName})
	return err
}

func (m *MongoOutputRepository) DeleteOutputForTests(ctx context.Context, testNames []string) error {
	_, err := m.OutputColl.DeleteMany(ctx, bson.M{"testname": bson.M{"$in": testNames}})
	return err
}

func (m *MongoOutputRepository) DeleteOutputByTestSuite(ctx context.Context, testSuiteName string) error {
	_, err := m.OutputColl.DeleteMany(ctx, bson.M{"testsuitename": testSuiteName})
	return err
}

func (m *MongoOutputRepository) DeleteOutputForTestSuites(ctx context.Context, testSuiteNames []string) error {
	_, err := m.OutputColl.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$in": testSuiteNames}})
	return err
}

func (m *MongoOutputRepository) DeleteOutputForAllTestSuite(ctx context.Context) error {
	_, err := m.OutputColl.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$ne": ""}})
	return err
}

func (m *MongoOutputRepository) DeleteAllOutput(ctx context.Context) error {
	_, err := m.OutputColl.DeleteMany(ctx, bson.M{})
	return err
}
