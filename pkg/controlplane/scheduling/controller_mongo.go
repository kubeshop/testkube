//nolint:govet // Allows pragmatic use of bson.D without Key, Value which makes it unreadable. It allows MongoDB Compass code generation.
package scheduling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

type MongoExecutionController struct {
	executionsCollection *mongo.Collection
}

func NewMongoExecutionController(col *mongo.Collection) Controller {
	return &MongoExecutionController{executionsCollection: col}
}

// StartExecution marks an execution that is currently assigned that it should be started.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, then no error will be emitted and no action will have been taken.
func (a MongoExecutionController) StartExecution(ctx context.Context, executionId string) error {
	res := a.executionsCollection.FindOneAndUpdate(ctx,
		bson.M{"$and": bson.A{
			bson.M{"id": executionId},
			bson.M{"result.status": testkube.ASSIGNED_TestWorkflowStatus},
		}},
		bson.M{"$set": bson.M{
			"statusat":      time.Now(),
			"result.status": testkube.STARTING_TestWorkflowStatus,
		}},
	)
	switch {
	case errors.Is(res.Err(), mongo.ErrNoDocuments):
	case res.Err() != nil:
		return fmt.Errorf("unable to update test workflow status: %s", res.Err())
	}
	return nil
}

// PauseExecution marks an execution that is currently running that it should be paused.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is currently running, then no error will be emitted and no action will
// have been taken.
func (a MongoExecutionController) PauseExecution(ctx context.Context, executionId string) error {
	res := a.executionsCollection.FindOneAndUpdate(ctx,
		bson.M{"$and": bson.A{
			bson.M{"id": executionId},
			bson.M{"result.status": testkube.RUNNING_TestWorkflowStatus},
		}},
		bson.M{"$set": bson.M{
			"statusat":      time.Now(),
			"result.status": testkube.PAUSING_TestWorkflowStatus,
		}},
	)
	switch {
	case errors.Is(res.Err(), mongo.ErrNoDocuments):
	case res.Err() != nil:
		return fmt.Errorf("unable to update test workflow status: %s", res.Err())
	}
	return nil
}

// ResumeExecution marks an execution that is currently paused that it should be resumed.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is currently paused, then no error will be emitted and no action will
// have been taken.
func (a MongoExecutionController) ResumeExecution(ctx context.Context, executionId string) error {
	res := a.executionsCollection.FindOneAndUpdate(ctx,
		bson.M{"$and": bson.A{
			bson.M{"id": executionId},
			bson.M{"result.status": testkube.PAUSED_TestWorkflowStatus},
		}},
		bson.M{"$set": bson.M{
			"statusat":      time.Now(),
			"result.status": testkube.RESUMING_TestWorkflowStatus,
		}},
	)
	switch {
	case errors.Is(res.Err(), mongo.ErrNoDocuments):
	case res.Err() != nil:
		return fmt.Errorf("unable to update test workflow status: %s", res.Err())
	}
	return nil
}

// AbortExecution marks an execution that is currently in an executing state that it
// should be aborted.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can only be aborted if they are currently in a Starting, Running, Paused, or
// Resuming state.
func (a MongoExecutionController) AbortExecution(ctx context.Context, executionId string) error {
	now := time.Now()
	res := a.executionsCollection.FindOneAndUpdate(ctx,
		bson.M{"$and": bson.A{
			bson.M{"id": executionId},
			bson.M{"result.status": bson.M{"$in": bson.A{
				testkube.STARTING_TestWorkflowStatus,
				testkube.SCHEDULING_TestWorkflowStatus,
				testkube.RUNNING_TestWorkflowStatus,
				testkube.PAUSED_TestWorkflowStatus,
				testkube.RESUMING_TestWorkflowStatus,
			}}},
		}},
		bson.M{"$set": bson.M{
			"statusat":               now,
			"result.status":          testkube.STOPPING_TestWorkflowStatus,
			"result.predictedstatus": testkube.ABORTED_TestWorkflowStatus,
		}},
	)
	switch {
	case errors.Is(res.Err(), mongo.ErrNoDocuments):
		// It is possible to jump directly to an aborted state in some circumstances.
		res := a.executionsCollection.FindOneAndUpdate(ctx,
			bson.M{"$and": bson.A{
				bson.M{"id": executionId},
				bson.M{"result.status": bson.M{"$in": bson.A{
					testkube.QUEUED_TestWorkflowStatus,
					testkube.ASSIGNED_TestWorkflowStatus,
				}}},
			}},
			bson.M{"$set": bson.M{
				"statusat":               now,
				"result.finishedat":      now,
				"result.status":          testkube.ABORTED_TestWorkflowStatus,
				"result.predictedstatus": testkube.ABORTED_TestWorkflowStatus,
			}},
		)
		switch {
		case errors.Is(res.Err(), mongo.ErrNoDocuments):
		case res.Err() != nil:
			return fmt.Errorf("unable to update test workflow status: %s", res.Err())
		}
	case res.Err() != nil:
		return fmt.Errorf("unable to update test workflow status: %s", res.Err())
	}
	return nil
}

// CancelExecution marks an execution that is currently in an executing state that it
// should be cancelled.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can only be cancelled if they are currently in a Starting, Running, Paused, or
// Resuming state.
func (a MongoExecutionController) CancelExecution(ctx context.Context, executionId string) error {
	now := time.Now()
	res := a.executionsCollection.FindOneAndUpdate(ctx,
		bson.M{"$and": bson.A{
			bson.M{"id": executionId},
			bson.M{"result.status": bson.M{"$in": bson.A{
				testkube.STARTING_TestWorkflowStatus,
				testkube.SCHEDULING_TestWorkflowStatus,
				testkube.RUNNING_TestWorkflowStatus,
				testkube.PAUSED_TestWorkflowStatus,
				testkube.RESUMING_TestWorkflowStatus,
			}}},
		}},
		bson.M{"$set": bson.M{
			"statusat":               now,
			"result.status":          testkube.STOPPING_TestWorkflowStatus,
			"result.predictedstatus": testkube.CANCELED_TestWorkflowStatus,
		}},
	)
	switch {
	case errors.Is(res.Err(), mongo.ErrNoDocuments):
		// It is possible to jump directly to a cancelled state in some circumstances.
		res := a.executionsCollection.FindOneAndUpdate(ctx,
			bson.M{"$and": bson.A{
				bson.M{"id": executionId},
				bson.M{"result.status": bson.M{"$in": bson.A{
					testkube.QUEUED_TestWorkflowStatus,
					testkube.ASSIGNED_TestWorkflowStatus,
				}}},
			}},
			bson.M{"$set": bson.M{
				"statusat":               now,
				"result.finishedat":      now,
				"result.status":          testkube.CANCELED_TestWorkflowStatus,
				"result.predictedstatus": testkube.CANCELED_TestWorkflowStatus,
			}},
		)
		switch {
		case errors.Is(res.Err(), mongo.ErrNoDocuments):
		case res.Err() != nil:
			return fmt.Errorf("unable to update test workflow status: %s", res.Err())
		}
	case res.Err() != nil:
		return fmt.Errorf("unable to update test workflow status: %s", res.Err())
	}
	return nil
}

// ForceCancelExecution marks an execution that is currently in an executing state as
// immediately cancelled.
// If no execution can be found that matches the passed ID, and is assigned to the passed
// runner ID, and is in an appropriate state, then no error will be emitted and no action will
// have been taken.
// Executions can be force cancelled if they are in any non-terminal state.
func (a MongoExecutionController) ForceCancelExecution(ctx context.Context, executionId string) error {
	now := time.Now()
	pipeline := bson.A{
		bson.M{"$set": bson.M{
			"statusat":               now,
			"result.finishedat":      now,
			"result.status":          testkube.CANCELED_TestWorkflowStatus,
			"result.predictedstatus": testkube.CANCELED_TestWorkflowStatus,
		}},
	}

	cancelExecutionSteps := createCancelExecutionStepsSteps(now)
	for _, step := range cancelExecutionSteps {
		pipeline = append(pipeline, step)
	}

	res := a.executionsCollection.FindOneAndUpdate(ctx,
		bson.M{"$and": bson.A{
			bson.M{"id": executionId},
			bson.M{"result.status": bson.M{"$in": bson.A{
				testkube.QUEUED_TestWorkflowStatus,
				testkube.ASSIGNED_TestWorkflowStatus,
				testkube.STARTING_TestWorkflowStatus,
				testkube.SCHEDULING_TestWorkflowStatus,
				testkube.RUNNING_TestWorkflowStatus,
				testkube.PAUSING_TestWorkflowStatus,
				testkube.PAUSED_TestWorkflowStatus,
				testkube.RESUMING_TestWorkflowStatus,
				testkube.STOPPING_TestWorkflowStatus,
			}}},
		}},
		pipeline,
	)
	if res.Err() != nil {
		return fmt.Errorf("unable to update test workflow status: %s", res.Err())
	}
	return nil
}

// createCancelExecutionStepsSteps creates steps for a Mongo Aggregation pipeline to cancel an execution's steps.
//
// For each step it will:
// - Cancel the step if it's not terminated (i.e. not passed or failed).
// - Set the queuedat, startedat and finishedat to `t` if missing.
//
// Note: This pipeline is pretty terrible, but it does the trick for now.
func createCancelExecutionStepsSteps(t time.Time) []bson.D {
	return []bson.D{
		bson.D{{"$addFields", bson.D{{"result.steps", bson.D{{"$objectToArray", "$result.steps"}}}}}},
		bson.D{
			{"$set",
				bson.D{
					{"result.steps",
						bson.D{
							{"$map",
								bson.D{
									{"input", "$result.steps"},
									{"in",
										bson.D{
											{"$cond",
												bson.D{
													{"if",
														bson.D{
															{"$in",
																bson.A{
																	"$$this.v.status",
																	bson.A{
																		"passed",
																		"failed",
																	},
																},
															},
														},
													},
													{"then",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"status", "$$this.v.status"}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													{"else",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"status", "canceled"}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"result.steps",
						bson.D{
							{"$map",
								bson.D{
									{"input", "$result.steps"},
									{"in",
										bson.D{
											{"$cond",
												bson.D{
													{"if",
														bson.D{
															{"$in",
																bson.A{
																	"$$this.v.finishedat",
																	bson.A{
																		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
																		primitive.Null{},
																		"",
																	},
																},
															},
														},
													},
													{"then",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"finishedat", t}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													{"else",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"finishedat", "$$this.v.finishedat"}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"result.steps",
						bson.D{
							{"$map",
								bson.D{
									{"input", "$result.steps"},
									{"in",
										bson.D{
											{"$cond",
												bson.D{
													{"if",
														bson.D{
															{"$in",
																bson.A{
																	"$$this.v.startedat",
																	bson.A{
																		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
																		primitive.Null{},
																		"",
																	},
																},
															},
														},
													},
													{"then",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"startedat", t}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													{"else",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"startedat", "$$this.v.startedat"}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{
			{"$set",
				bson.D{
					{"result.steps",
						bson.D{
							{"$map",
								bson.D{
									{"input", "$result.steps"},
									{"in",
										bson.D{
											{"$cond",
												bson.D{
													{"if",
														bson.D{
															{"$in",
																bson.A{
																	"$$this.v.queuedat",
																	bson.A{
																		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
																		primitive.Null{},
																		"",
																	},
																},
															},
														},
													},
													{"then",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"queuedat", t}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													{"else",
														bson.D{
															{"$mergeObjects",
																bson.A{
																	"$$this",
																	bson.D{
																		{"v",
																			bson.D{
																				{"$mergeObjects",
																					bson.A{
																						"$$this.v",
																						bson.D{{"queuedat", "$$this.v.queuedat"}},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		bson.D{{"$addFields", bson.D{{"result.steps", bson.D{{"$arrayToObject", "$result.steps"}}}}}},
	}
}
