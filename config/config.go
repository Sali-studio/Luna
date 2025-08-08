package config

import (
	"github.com/spf13/viper"
	"luna/interfaces"
)

// Config はアプリケーションの設定を保持します。
type Config struct {
	Discord struct {
		Token string `mapstructure:"token"`
	}
	Google struct {
		ProjectID       string `mapstructure:"project_id"`
		CredentialsPath string `mapstructure:"credentials_path"`
	}
	Web struct {
		ClientID      string `mapstructure:"client_id"`
		ClientSecret  string `mapstructure:"client_secret"`
		RedirectURI   string `mapstructure:"redirect_uri"`
		SessionSecret string `mapstructure:"session_secret"`
	}
}

var Cfg *Config

// LoadConfig は設定ファイルから設定を読み込みます。
func LoadConfig(log interfaces.Logger) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return err
	}

	log.Info("設定ファイルを正常に読み込みました。")
	return nil
}