package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	ConfigFilename = "config.json"
)

type Result struct {
	name    string
	host    string
	average time.Duration
}

var config *Config

func main() {
	chanBench := make(chan Result, 0)
	config = &Config{}
	if err := config.Load(ConfigFilename); err != nil {
		fmt.Printf("Error loading configuration file: %v\n", err)
		return
	}

	hostCount := 0
	maxNameLength := 0
	for _, p := range config.Providers {
		for _, h := range p.Hosts {
			go runBench(p.Name, h, chanBench)

			hostCount++
			if len(p.Name) > maxNameLength {
				maxNameLength = len(p.Name)
			}
		}
	}
	nameFormat := "%" + strconv.Itoa(maxNameLength) + "s"

	for i := 0; i < hostCount; i++ {
		result := <-chanBench

		average := fmt.Sprintf("%s", result.average)
		if result.average.Nanoseconds() < 1 {
			average = "error"
		}

		fmt.Printf(nameFormat+" (%15s) average: %s\n",
			result.name, result.host, average)
	}
}

func runBench(name, host string, result chan Result) {
	var latencySum int64
	var skipped int64
	chanTarget := make(chan int64, 0)
	targetCount := len(config.Targets)

	for _, target := range config.Targets {
		go runTarget(host, target, chanTarget)
	}

	for i := 0; i < targetCount; i++ {
		queryResult := <-chanTarget
		if queryResult > 0 {
			latencySum += queryResult
		} else {
			skipped++
		}
	}

	result <- Result{name, host,
		time.Duration(latencySum / (int64(targetCount) - skipped))}
}

func runTarget(host, target string, result chan int64) {
	const TimeoutErrorText = `i/o timeout`
	const TooManyOpenFilesErrorText = `too many open files`
	c := dns.Client{}
	m := dns.Msg{}
	rounds := config.Rounds
	var latencySum int64
	var skipped int64

	target = target + "."
	host = host + ":53"
	c.DialTimeout = time.Millisecond * 500
	c.ReadTimeout = c.DialTimeout
	c.WriteTimeout = c.DialTimeout

	m.SetQuestion(target, dns.TypeA)
	for i := 0; i < rounds; i++ {
		_, t, err := c.Exchange(&m, host)
		if err != nil {
			/*if strings.Index(err.Error(), TimeoutErrorText) > 0 {
				fmt.Printf("Timeout: %s -> %s\n", host, target)
			} else */if strings.Index(err.Error(), TooManyOpenFilesErrorText) > 0 {
				fmt.Printf("Overflow: %s -> %s\n", host, target)
			}
			skipped++
		} else {
			latencySum += t.Nanoseconds()
		}
	}

	result <- latencySum / (int64(rounds) - skipped)
}
