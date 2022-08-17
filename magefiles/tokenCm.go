package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type tokenSecret struct {
	APIVersion string `yaml:"apiVersion"`
	Data       struct {
		Username string `yaml:"username"`
		Token    string `yaml:"token"`
	} `yaml:"data"`
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Type string `yaml:"type"`
}

func (t *tokenSecret) TokenInit(user string, token string) {
	t.Data.Username = stringOrDefault(user, "anon")
	t.Data.Token = stringOrDefault(token, "noMoreSecrets")
}

func (t *tokenSecret) MetadataInit(name string, namespace string) {
	t.Metadata.Name = stringOrDefault(name, "cci-reader")
	t.Metadata.Namespace = stringOrDefault(namespace, "circleci-exporter")
}

func stringOrDefault(str string, def string) string {
	if str == "" {
		return def
	}

	return str
}

func (t *tokenSecret) WriteYaml(outFile string) error {
	_, err := os.Stat(outFile)
	if err == nil {
		return fmt.Errorf("file already exist")
	}

	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(outFile, data, 0666)

	if err != nil {
		return err
	}

	return nil
}

func NewTokenSecret() tokenSecret {
	return tokenSecret{
		APIVersion: "v1",
		Data: struct {
			Username string "yaml:\"username\""
			Token    string "yaml:\"token\""
		}{
			Token:    "noSecrets",
			Username: "anon",
		},
		Kind: "Secret",
		Metadata: struct {
			Name      string "yaml:\"name\""
			Namespace string "yaml:\"namespace\""
		}{
			Name:      "cci-reader",
			Namespace: "circleci-exporter",
		},
		Type: "Opaque",
	}
}
