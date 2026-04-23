package service

import (
	"context"
	"strings"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolemodel "github.com/apache/dubbo-admin/pkg/console/model"
	logprovider "github.com/apache/dubbo-admin/pkg/observability/logs"
)

type LogService struct {
	provider logprovider.Provider
}

func NewLogService(provider logprovider.Provider) *LogService {
	return &LogService{provider: provider}
}

func (s *LogService) SearchLogs(ctx context.Context, req *consolemodel.SearchLogsReq) (*consolemodel.SearchLogsResp, error) {
	if req.Mesh == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "mesh is required")
	}
	if req.AppName == "" && req.ServiceName == "" && req.InstanceName == "" {
		return nil, bizerror.New(bizerror.InvalidArgument, "one of appName, serviceName, instanceName is required")
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}
	return s.provider.Search(ctx, req)
}

func (s *LogService) AnalyzeErrorLogs(ctx context.Context, req *consolemodel.AnalyzeErrorLogsReq) (*consolemodel.AnalyzeErrorLogsResp, error) {
	searchResp, err := s.SearchLogs(ctx, &req.SearchLogsReq)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, item := range searchResp.Logs {
		key := item.Severity
		if key == "" {
			key = "unknown"
		}
		if strings.Contains(strings.ToLower(item.Message), "error") {
			counts[key]++
		}
	}

	top := make([]consolemodel.ErrorBucket, 0, len(counts))
	for k, v := range counts {
		top = append(top, consolemodel.ErrorBucket{Key: k, Count: v})
	}

	return &consolemodel.AnalyzeErrorLogsResp{
		Summary:   "basic error analysis completed",
		TopErrors: top,
	}, nil
}
