package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Rounds    int        `json:"rounds"`
	Targets   []string   `json:"targets"`
	Providers []Provider `json:"providers"`
}

type Provider struct {
	Name  string   `json:"name"`
	Hosts []string `json:"hosts"`
}

func (c *Config) Load(filename string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(contents, c)
}
