package handlers

import "core_project/shared/sdk/config"

// AppConfig holds the application configuration instance for handlers
var AppConfig *config.Config

// InitConfig initializes the config for handlers package
func InitConfig(cfg *config.Config) {
	AppConfig = cfg
}
