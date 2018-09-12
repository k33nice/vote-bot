package vote

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

// Config - telegram bot configuration.
type Config struct {
	APIToken string
	Appeals  []string
	Place    struct {
		URL      string
		Location struct {
			Latitude  float64
			Longitude float64
		}
	}
}

// NewConfig - return new config instance
func NewConfig() *Config {
	var c Config

	apiToken := os.Getenv("API_TOKEN")

	viper.SetConfigName("config")
	viper.AddConfigPath(os.Getenv("CONFIG_PATH"))
	viper.ReadInConfig()

	err := viper.Unmarshal(&c)

	if err != nil {
		log.Print(err)
	}

	c.APIToken = apiToken

	return &c
}
