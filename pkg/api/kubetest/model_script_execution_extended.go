package kubetest

import "go.mongodb.org/mongo-driver/bson/primitive"

func NewScriptExecution(scriptName, name string, execution Execution) ScriptExecution {
	return ScriptExecution{
		Id:         primitive.NewObjectID().Hex(),
		Name:       name,
		ScriptName: scriptName,
		Execution:  &execution,
		ScriptType: "postman/collection", // TODO need to be passed from CRD type
	}
}
