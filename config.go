package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Cmd        string
	Dockerfile string
	Image      string
	Services   map[string]Service
}

type Service struct {
	Format string
	Image  string
	Port   string
	Hooks  map[string][]string
}

func ConfigFromFile() (*Config, error) {
	pwd, _ := os.Getwd()
	_, temakiYml, err := GetTemakiYml(pwd)
	if err != nil {
		return nil, err
	}

	config, err := ioutil.ReadAll(temakiYml)
	if err != nil {
		return nil, err
	}

	conf := Config{}
	if err := yaml.Unmarshal(config, &conf); err != nil {
		return nil, err
	}

	if len(os.Args) > 1 {
		conf.Cmd = strings.Join(os.Args[1:], " ")
	}

	return &conf, nil
}

func GetTemakiYml(basePath string) (string, *os.File, error) {
	fullPath := filepath.Join(basePath, "temaki.yml")
	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) && basePath != "/" {
			return GetTemakiYml(filepath.Dir(basePath))
		}
		return "", nil, err
	}
	if stat.IsDir() {
		return basePath, nil, errors.New(fmt.Sprintf("%s is a directory.", fullPath))
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return "", nil, err
	}

	return basePath, file, nil
}
