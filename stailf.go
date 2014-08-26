package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"stailf/splunk"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	URL     string
	Session string
}

// ProcessEnv processes environment variables
func ProcessEnv(c *Config) error {

	if err := envconfig.Process("splunk", c); err != nil {
		return err
	}

	if c.URL == "" {
		return errors.New("set env SPLUNK_URL to continue")
	}

	if c.Session == "" {
		return errors.New("set env SPLUNK_SESSION to continue")
	}

	return nil
}

// usage displays usage information
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] query\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options are: \n")
	flag.PrintDefaults()
}

func main() {

	// Process enviroment variables
	var c Config
	if err := ProcessEnv(&c); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
		os.Exit(2)
	}

	// Process commandline flags
	delay := flag.Int("delay", 10, "Output delay (ms) for each event")
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	query := flag.Args()[0]

	client := splunk.NewClient("", "", c.URL)
	// Set session key
	client.SetSessionKey(c.Session)

	// Submit realtime search job
	params := splunk.Param{
		"search_mode": "realtime",
		"timeout":     "60",
		"auto_cancel": "60",
	}
	sid, err := client.SubmitSearch(query, "stailf", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
		os.Exit(2)
	}

	// Handle signals
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-ch
		fmt.Fprintf(os.Stdout, "%s received, stopping...\n", sig)
		// If we received a signal, try to cancel the current search
		_ = client.CancelSearch(sid)
		// Abort streaming of events
		client.Shutdown()
	}()

	// Read events from search job
	eventCh := client.StreamEvents(sid)
	for e := range eventCh {
		if err, isErr := e.(error); isErr {
			fmt.Fprintf(os.Stderr, "[ERROR] %s\n", err)
			break
		}

		if e, isEvent := e.(*splunk.Events); isEvent {
			for i := range e.Results {
				fmt.Fprintf(os.Stdout, "\x1b[1;32m>\x1b[0m %s \n", e.Results[i].Raw)
				time.Sleep(time.Duration(*delay) * time.Millisecond)
			}
		}
	}

	os.Exit(0)
}
