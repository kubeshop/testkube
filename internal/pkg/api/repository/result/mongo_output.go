package result

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type ExecutionOutput struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	TestName      string `json:"testName,omitempty"`
	TestSuiteName string `json:"testSuiteName,omitempty"`
	Output        string `json:"output"`
}

func (r *MongoRepository) GetOutput(ctx context.Context, id string) (output string, err error) {
	var eOutput ExecutionOutput
	err = r.Output.FindOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}).Decode(&eOutput)
	return eOutput.Output, err
}

func (m *MongoRepository) GetOutputByTest(ctx context.Context, testName string) (output string, err error) {
	var eOutput ExecutionOutput
	err = m.Output.FindOne(ctx, bson.M{"testName": testName}).Decode(&eOutput)
	return eOutput.Output, err
}

func (m *MongoRepository) GetOutputByTestSuite(ctx context.Context, testSuiteName string) (output string, err error) {
	var eOutput ExecutionOutput
	err = m.Output.FindOne(ctx, bson.M{"testSuiteName": testSuiteName}).Decode(&eOutput)
	return eOutput.Output, err
}

func (m *MongoRepository) InsertOutput(ctx context.Context, id, testName, testSuiteName, output string) error {
	_, err := m.Output.InsertOne(ctx, ExecutionOutput{Id: id, Name: id, TestName: testName, TestSuiteName: testSuiteName, Output: output})
	return err
}

func (m *MongoRepository) UpdateOutput(ctx context.Context, id, output string) error {
	_, err := m.Output.UpdateOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}}, bson.M{"$set": bson.M{"output": output}})
	return err
}

func (m *MongoRepository) DeleteOutput(ctx context.Context, id string) error {
	_, err := m.Output.DeleteOne(ctx, bson.M{"$or": bson.A{bson.M{"id": id}, bson.M{"name": id}}})
	return err
}

func (m *MongoRepository) DeleteOutputByTest(ctx context.Context, testName string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testName": testName})
	return err
}

func (m *MongoRepository) DeleteOutputForTests(ctx context.Context, testNames []string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testName": bson.M{"$in": testNames}})
	return err
}

func (m *MongoRepository) DeleteOutputByTestSuite(ctx context.Context, testSuiteName string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testSuiteName": testSuiteName})
	return err
}

func (m *MongoRepository) DeleteOutputForTestSuites(ctx context.Context, testSuiteNames []string) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testSuiteName": bson.M{"$in": testSuiteNames}})
	return err
}

func (m *MongoRepository) DeleteOutputForAllTestSuite(ctx context.Context) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{"testSuiteName": bson.M{"$ne": ""}})
	return err
}

func (m *MongoRepository) DeleteAllOutput(ctx context.Context) error {
	_, err := m.Output.DeleteMany(ctx, bson.M{})
	return err
}
