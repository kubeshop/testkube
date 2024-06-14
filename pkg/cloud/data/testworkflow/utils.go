package testworkflow

import (
	"context"
	"encoding/json"

	"github.com/kubeshop/testkube/pkg/cloud/data/executor"
	testworkflow2 "github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func passWithErr[T any, U any](e executor.Executor, ctx context.Context, req interface{}, fn func(u T) (U, error)) (v U, err error) {
	response, err := e.Execute(ctx, command(req), req)
	if err != nil {
		return v, err
	}
	var commandResponse T
	if err = json.Unmarshal(response, &commandResponse); err != nil {
		return v, err
	}
	return fn(commandResponse)
}

func pass[T any, U any](e executor.Executor, ctx context.Context, req interface{}, fn func(u T) U) (v U, err error) {
	return passWithErr(e, ctx, req, func(u T) (U, error) {
		return fn(u), nil
	})
}

func passNoContentProcess[T any](e executor.Executor, ctx context.Context, req interface{}, fn func(u T) error) (err error) {
	_, err = passWithErr(e, ctx, req, func(u T) (interface{}, error) {
		return nil, fn(u)
	})
	return err
}

func passNoContent(e executor.Executor, ctx context.Context, req interface{}) (err error) {
	return passNoContentProcess(e, ctx, req, func(u interface{}) error {
		return nil
	})
}

func mapFilters(s []testworkflow2.Filter) []*testworkflow2.FilterImpl {
	v := make([]*testworkflow2.FilterImpl, len(s))
	for i := range s {
		if vv, ok := s[i].(testworkflow2.FilterImpl); ok {
			v[i] = &vv
		} else {
			v[i] = s[i].(*testworkflow2.FilterImpl)
		}
	}
	return v
}
