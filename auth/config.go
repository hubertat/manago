package auth

import (
	"encoding/json"
	"os"
)

type Config struct {
	Host   string `json:"host"`
	Port   uint   `json:"port"`
	Domain string `json:"domain"`
	BaseDn string `json:"base_dn"`
}

func ConfigFromFile(filename string) (cfg Config, err error) {
	cfg = Config{}

	b, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return
	}
	return
}
