package config

import (
	"encoding/json"
	"os"

	"github.com/WangYihang/PrGoxy/lib/util/log"
)

type Config struct {
	Proxy struct {
		LHost string `json:"lhost"`
		LPort int16  `json:"lport"`
	} `json:"proxy"`
	Block struct {
		Hosts []string `hosts:"redirect"`
		Sites []string `sites:"redirect"`
	} `json:"block"`
	Redirect map[string]string `json:"redirect"`
	Cache    bool              `json:cache`
}

var Cfg Config

func init() {
	log.Info("Loading config")
	Cfg.Reload()
}

func (config *Config) Reload() {
	// Open config file
	file, err := os.Open("config.json")
	defer file.Close()
	if err != nil {
		log.Error("Can not open config file")
	}
	// Parse content
	err = json.NewDecoder(file).Decode(config)
	if err != nil {
		log.Error("Failed to parse config file: %s", err)
	}
}
