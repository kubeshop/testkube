package logs

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/logs/pb"
	"github.com/kubeshop/testkube/pkg/logs/repository"
	"github.com/kubeshop/testkube/pkg/logs/state"
)

func NewLogsServer(repo repository.Factory, state state.Interface) *LogsServer {
	return &LogsServer{
		state:       state,
		repoFactory: repo,
		log:         log.DefaultLogger.With("service", "logs-grpc-server"),
	}
}

type LogsServer struct {
	pb.UnimplementedLogsServiceServer
	repoFactory repository.Factory
	state       state.Interface
	log         *zap.SugaredLogger
}

func (s LogsServer) Logs(req *pb.LogRequest, stream pb.LogsService_LogsServer) error {
	ctx := stream.Context()

	logger := s.log.With("execution_id", req.ExecutionId)

	// get state of current log stream (pending or finished)
	st, err := s.state.Get(ctx, req.ExecutionId)
	if err != nil {
		return err
	}

	// get valid repository based on state
	repo, err := s.repoFactory.GetRepository(st)
	if err != nil {
		return err
	}

	logger.Debugw("starting sending log stream", "repo", fmt.Sprintf("%T", repo), "state", st)

	// stream logs from repository through GRPC channel
	ch, err := repo.Get(ctx, req.ExecutionId)
	if err != nil {
		return err
	}

	for l := range ch {
		log.Tracew(logger, "sending log chunk", "log", l)
		if err := stream.Send(pb.MapResponseToPB(l)); err != nil {
			return err
		}
	}

	logger.Debugw("log stream finished")

	return nil
}
