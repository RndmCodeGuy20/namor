package config

import (
	"namor/pkg/errors"

	"github.com/spf13/viper"
)

type ValidationError struct {
	*errors.BaseError
	Value   interface{}
	Message string
	Field   string
}

func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		BaseError: &errors.BaseError{
			Message: message,
			Code:    "CONFIG_VALIDATION_ERROR",
			Details: map[string]interface{}{"field": field, "value": value},
		},
		Field:   field,
		Value:   value,
		Message: message,
	}
}

func LoadConfig(path string) (*NamorConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config NamorConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func ValidateConfig(config *NamorConfig) error {
	if config.Port == 0 {
		config.Port = 6060
	} else if config.Port < 1 || config.Port > 65535 {
		return NewValidationError("port", config.Port, "Port must be between 1 and 65535")
	}

	if config.Host == "" {
		dockerHost := viper.GetString("DOCKER_HOST")
		if dockerHost == "" {
			return NewValidationError("host", config.Host, "Docker host must be provided via config or DOCKER_HOST environment variable")
		}
		config.Host = dockerHost
	}

	if config.Timeout == 0 {
		config.Timeout = 30
	}

	if config.Webhook.Url == "" {
		return NewValidationError("webhook.url", config.Webhook.Url, "Webhook URL must be provided")
	}

	if len(config.Services) == 0 {
		return NewValidationError("services", config.Services, "At least one service must be defined")
	}

	for name, service := range config.Services {
		if service.Image == "" {
			return NewValidationError("services."+name+".image", service.Image, "Service image must be provided")
		}
		if service.Tag == "" {
			service.Tag = "latest"
			config.Services[name] = service
		}
	}

	return nil
}
