package config

import (
	config "github.com/goletan/config-library/pkg"
	logger "github.com/goletan/logger-library/pkg"
	"github.com/goletan/services-library/shared/types"
	"go.uber.org/zap"
)

func LoadServicesConfig(log *logger.ZapLogger) (*types.ServicesConfig, error) {
	var cfg types.ServicesConfig

	if err := config.LoadConfig("Services", &cfg, log); err != nil {
		log.Error("Failed to load services-library configuration", zap.Error(err))
		return nil, err
	}

	return &cfg, nil
}
