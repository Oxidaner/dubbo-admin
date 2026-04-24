package logs

import (
	"context"
	"math"

	consolemodel "github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

const ComponentType runtime.ComponentType = "log provider"

func init() {
	runtime.RegisterComponent(&logProviderComponent{})
}

type Provider interface {
	Search(ctx context.Context, req *consolemodel.SearchLogsReq) (*consolemodel.SearchLogsResp, error)
}

type ProviderComponent interface {
	runtime.Component
	LogProvider() Provider
}

var _ ProviderComponent = &logProviderComponent{}

type logProviderComponent struct {
	provider Provider
}

func (c *logProviderComponent) RequiredDependencies() []runtime.ComponentType {
	return nil
}

func (c *logProviderComponent) Type() runtime.ComponentType {
	return ComponentType
}

func (c *logProviderComponent) Order() int {
	return math.MaxInt - 10
}

func (c *logProviderComponent) Init(ctx runtime.BuilderContext) error {
	c.provider = NewMockProvider()
	return nil
}

func (c *logProviderComponent) Start(runtime.Runtime, <-chan struct{}) error {
	return nil
}

func (c *logProviderComponent) LogProvider() Provider {
	return c.provider
}

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) Search(ctx context.Context, req *consolemodel.SearchLogsReq) (*consolemodel.SearchLogsResp, error) {
	return &consolemodel.SearchLogsResp{
		Logs: []consolemodel.LogItem{
			{
				Timestamp:    "2024-01-01T00:00:00Z",
				AppName:      req.AppName,
				ServiceName:  req.ServiceName,
				InstanceName: req.InstanceName,
				Severity:     "INFO",
				Message:      "Mock log message for testing",
			},
		},
	}, nil
}
