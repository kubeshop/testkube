package resolvers

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

var (
	sample = testkube.ExecutorDetails{
		Name: "sample",
		Executor: &testkube.Executor{
			ExecutorType:         "job",
			Image:                "",
			ImagePullSecrets:     nil,
			Command:              nil,
			Args:                 nil,
			Types:                []string{"curl/test"},
			Uri:                  "",
			ContentTypes:         nil,
			JobTemplate:          "",
			JobTemplateReference: "",
			Labels:               map[string]string{"label-name": "label-value"},
			Features:             nil,
			Meta:                 nil,
		},
	}
)

func TestExecutorsResolver_Query_Executors(t *testing.T) {
	t.Run("should default selector to empty string", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		q := `query { executors { name } }`
		c.ExecutorsService.EXPECT().List("").Return([]testkube.ExecutorDetails{}, nil)
		var resp interface{}
		c.Client.MustPost(q, &resp)
	})

	t.Run("should pass the selector as an argument", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		q := `query { executors(selector: "label=value") { name } }`
		c.ExecutorsService.EXPECT().List("label=value").Return([]testkube.ExecutorDetails{}, nil)
		var resp interface{}
		c.Client.MustPost(q, &resp)
	})

	t.Run("should throw client error", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		q := `query { executors { name } }`
		c.ExecutorsService.EXPECT().List("").Return(nil, errors.New("some error"))

		var resp interface{}
		e := c.Client.Post(q, &resp)
		assert.Error(t, e)
	})

	t.Run("should return back the executors", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		q := `query { executors { name } }`
		c.ExecutorsService.EXPECT().List("").Return([]testkube.ExecutorDetails{sample}, nil)
		var resp struct{ Executors []struct{ Name string } }
		c.Client.MustPost(q, &resp)
		assert.Len(t, resp.Executors, 1)
		assert.Equal(t, resp.Executors[0].Name, sample.Name)
	})
}

func TestExecutorsResolver_Subscription_Executors(t *testing.T) {
	t.Run("should default selector to empty string", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		input := make(chan []testkube.ExecutorDetails, 1)
		input <- []testkube.ExecutorDetails{}
		close(input)

		q := `subscription { executors { name } }`
		c.ExecutorsService.EXPECT().SubscribeList(gomock.Any(), "").Return(input, nil)
		var resp interface{}
		c.Client.WebsocketOnce(q, &resp)
	})

	t.Run("should pass the selector as an argument", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		input := make(chan []testkube.ExecutorDetails, 1)
		input <- []testkube.ExecutorDetails{}
		close(input)

		q := `subscription { executors(selector: "label=value") { name } }`
		c.ExecutorsService.EXPECT().SubscribeList(gomock.Any(), "label=value").Return(input, nil)
		var resp interface{}
		assert.NoError(t, c.Client.WebsocketOnce(q, &resp))
	})

	t.Run("should throw client error", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		q := `subscription { executors { name } }`
		c.ExecutorsService.EXPECT().SubscribeList(gomock.Any(), "").Return(nil, errors.New("some error"))

		var resp interface{}
		e := c.Client.WebsocketOnce(q, &resp)
		assert.Error(t, e)
	})

	t.Run("should return back the executors", func(t *testing.T) {
		c := NewTestClient(t)
		defer c.Finish()

		input := make(chan []testkube.ExecutorDetails, 1)
		input <- []testkube.ExecutorDetails{}
		go func() {
			time.Sleep(10 * time.Millisecond)
			input <- []testkube.ExecutorDetails{sample}
			close(input)
		}()

		q := `subscription { executors { name } }`
		c.ExecutorsService.EXPECT().SubscribeList(gomock.Any(), "").Return(input, nil)
		var resp struct{ Executors []struct{ Name string } }
		s := c.Client.Websocket(q)

		assert.NoError(t, s.Next(&resp))
		assert.Len(t, resp.Executors, 0)

		assert.NoError(t, s.Next(&resp))
		assert.Len(t, resp.Executors, 1)
		assert.Equal(t, resp.Executors[0].Name, sample.Name)
	})
}
