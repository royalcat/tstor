package config

import (
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

func Load(path string) (*Config, error) {

	err := k.Load(structs.Provider(defaultConfig, "koanf"), nil)
	if err != nil {
		return nil, err
	}

	if path != "" {
		_ = k.Load(file.Provider(path), yaml.Parser()) // its ok if file doesnt exist
	}

	err = k.Load(env.Provider("TSTOR_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "TSTOR_")), "_", ".", -1)
	}), nil)
	if err != nil {
		return nil, err
	}

	data, err := k.Marshal(yaml.Parser())
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return nil, err
	}

	conf := Config{}
	k.Unmarshal("", &conf)

	return &conf, nil
}
