package configs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"go.yaml.in/yaml/v4"
)

func LoadConfigFromFile(path string, rc *RexyConf, proxies *sync.Map) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	var conf RexyConf

	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &conf); err != nil {
			return fmt.Errorf("failed to parse config: %v", err)
		}
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(data, &conf); err != nil {
			return fmt.Errorf("failed to parse config: %v", err)
		}
	default:
		return fmt.Errorf("failed to parse config: unknown format based on file extension: '%s'", ext)
	}

	if ok, err := conf.IsValid(); !ok {
		return err
	}

	*rc = conf

	newProxies, err := buildProxies(rc)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	proxies = newProxies

	port := conf.HttpPort
	if conf.UseHTTPS() {
		port = conf.HttpsPort
	}
	log.Printf("reverse proxy page available on %s (%s:%d)\n", conf.Domain, conf.BindToIp, port)

	return nil
}
