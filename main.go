package main

import (
	"fmt"
	"time"

	"github.com/miekg/dns"
)

const (
	CONFIG_FILENAME = "config.json"
)

type Result struct {
	name    string
	host    string
	average time.Duration
}

var config *Config

func main() {
	chanBench := make(chan Result)
	config = &Config{}
	config.Load(CONFIG_FILENAME)

	hostCount := 0
	for _, p := range config.Providers {
		for _, h := range p.Hosts {
			go runBench(p.Name, h, chanBench)
			hostCount++
		}
	}

	for i := 0; i < hostCount; i++ {
		result := <-chanBench

		average := fmt.Sprintf("%s", result.average)
		if result.average.Nanoseconds() == -1 {
			average = "error"
		}

		fmt.Printf("%s (%s) average: %s\n",
			result.name, result.host, average)
	}
}

func runBench(name, host string, result chan Result) {
	var latencySum int64 = 0

	c := dns.Client{}
	m := dns.Msg{}

	for _, target := range config.Targets {
		m.SetQuestion(target+".", dns.TypeA)
		for i := 0; i < config.Rounds; i++ {
			_, t, err := c.Exchange(&m, host+":53")
			if err != nil {
				result <- Result{name, host, -1}
				return
			}
			latencySum += t.Nanoseconds()
		}
	}

	result <- Result{name, host, time.Duration(
		latencySum / (int64(config.Rounds) + int64(len(config.Targets))))}
}
