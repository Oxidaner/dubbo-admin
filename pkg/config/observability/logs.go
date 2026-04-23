package observability

type LogProviderType string

const (
	LogProviderLoki LogProviderType = "loki"
)

type LogsConfig struct {
	// DefaultProvider is the default log provider
	DefaultProvider string `json:"defaultProvider" yaml:"defaultProvider"`
	// Providers is the list of log providers
	Providers []LogProviderConfig `json:"providers" yaml:"providers"`
}

type LogProviderConfig struct {
	Name     string          `json:"name" yaml:"name"`
	Type     LogProviderType `json:"type" yaml:"type"`
	Endpoint string          `json:"endpoint" yaml:"endpoint"`
	Tenant   string          `json:"tenant,omitempty" yaml:"tenant,omitempty"`
}
