package config

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"strings"
	"vqlite/logging"
)

var GlobalConfig *Config

type ServiceConfig struct {
	Host                 string `mapstructure:"host"`
	Port                 int    `mapstructure:"port"`
	RunMode              string `mapstructure:"runMode"`
	DataPath             string `mapstructure:"dataPath"`
	SegmentVectorMaxSize int64  `mapstructure:"segmentVectorMaxSize"`
}

type Config struct {
	ServiceConfig ServiceConfig `mapstructure:"serviceConfig"`
}

func init() {
	viper.AddConfigPath(".")
	viper.SetConfigName("vqlite")
	viper.SetConfigType("yaml")  // set config type to yaml
	viper.AutomaticEnv()         // read in environment variables that match
	viper.SetEnvPrefix("VQLITE") // set the enviroment variable prefix
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	logging.InitLogger()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("read config file error")
	}
	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		log.Fatal().Err(err).Msg("unmarshal config file error")
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if GlobalConfig.ServiceConfig.RunMode == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if GlobalConfig.ServiceConfig.SegmentVectorMaxSize < 10000 {
		GlobalConfig.ServiceConfig.SegmentVectorMaxSize = 10000
		log.Info().Msgf("segmentVectorMaxSize is too small, set to default value 10000")
	}

}
