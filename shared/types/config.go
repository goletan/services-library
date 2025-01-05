package types

type ServicesConfig struct {
	Discovery DiscoveryConfig `yaml:"discovery"`
}

type DiscoveryConfig struct {
	Strategies      []StrategyConfig `yaml:"strategies"`
	DefaultStrategy string           `yaml:"default_strategy"`
}

type StrategyConfig struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"` // For Kubernetes
	Network   string `yaml:"network,omitempty"`   // For Docker
	Domain    string `yaml:"domain,omitempty"`    // For DNS
}
