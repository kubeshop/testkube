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
	Output *mongo.Collection
}

func NewMongoOutputRepository(db *mongo.Database) *MongoOutputRepository {
	return &MongoOutputRepository{
		Output: db.Collection(CollectionOutput),
	}
}

func (r *MongoOutputRepository) GetOutput(ctx context.Context, id string) (output string, err error) {
	var eOutput ExecutionOutput
	err = r.Output.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&eOutput)
	return eOutput.Output, err
}

func (m *MongoOutputRepository) InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	_, err := m.Output.InsertOne(ctx, ExecutionOutput{Id: id, Name: id, TestName: testName, TestSuiteName: testSuiteName, Output: output})
	return err
}

func (m *MongoOutputRepository) UpdateOutput(ctx context.Context, id, output string) error {
	_, err := m.Output.UpdateOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}, bson.M{"$set": bson.M{"output": output}})
	return err
}

func (m *MongoOutputRepository) DeleteOutput(ctx context.Context, id string) error {
	_, err := m.Output.DeleteOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}})
	return err
}

func (m *MongoOutputRepository) DeleteOutputByTest(ctx context.Context, testName string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testname": testName})
	return err
}

func (m *MongoOutputRepository) DeleteOutputForTests(ctx context.Context, testNames []string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testname": bson.M{"$in": testNames}})
	return err
}

func (m *MongoOutputRepository) DeleteOutputByTestSuite(ctx context.Context, testSuiteName string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testsuitename": testSuiteName})
	return err
}

func (m *MongoOutputRepository) DeleteOutputForTestSuites(ctx context.Context, testSuiteNames []string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$in": testSuiteNames}})
	return err
}

func (m *MongoOutputRepository) DeleteOutputForAllTestSuite(ctx context.Context) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testsuitename": bson.M{"$ne": ""}})
	return err
}

func (m *MongoOutputRepository) DeleteAllOutput(ctx context.Context) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{})
	return err
}
