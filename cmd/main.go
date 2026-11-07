package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	plan := flag.String("plan", "oidcc-config-certification-test-plan", "upstream test plan name")
	module := flag.String("module", "", "run only this module (all matching variants)")
	configFile := flag.String("config", "configs/local.example.json", "suite configuration JSON")
	variantJSON := flag.String("variant", "{}", "plan variant selection as a JSON object")
	baseURL := flag.String("suite-url", "https://localhost:8443", "local suite HTTPS origin")
	connect := flag.String("connect", "127.0.0.1:8443", "local dial address; empty uses DNS")
	caFile := flag.String("ca", ".local/certs/ca.crt", "trusted local CA PEM")
	results := flag.String("results", "results", "parent directory for run artifacts")
	timeout := flag.Duration("timeout", 60*time.Second, "maximum wait per module")

	flag.Parse()
	if flag.NArg() != 0 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments or non-positive timeout")
		return 2
	}

	var variants map[string]string
	if err := json.Unmarshal([]byte(*variantJSON), &variants); err != nil {
		fmt.Fprintln(os.Stderr, "variant:", err)
		return 2
	}

	config, err := os.ReadFile(*configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(config, &fields); err != nil || fields == nil {
		fmt.Fprintln(os.Stderr, "configuration must be a JSON object")
		return 2
	}

	client, err := newClient(*baseURL, *caFile, *connect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	defer client.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(*results, 0700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	dir, err := os.MkdirTemp(*results, "run-"+time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	abs, _ := filepath.Abs(dir)
	fmt.Println("Results:", abs)
	summary, err := runPlan(ctx, client, config, Options{
		Plan:         *plan,
		Module:       *module,
		Variant:      variants,
		Output:       dir,
		BaseURL:      strings.TrimRight(*baseURL, "/"),
		Timeout:      *timeout,
		PollInterval: time.Second,
	})

	if summary.Plan != "" {
		fmt.Printf("\nPlan: %s/plan-detail.html?plan=%s\n", strings.TrimRight(*baseURL, "/"), url.QueryEscape(summary.Plan))
	}

	for _, mod := range summary.Modules {
		fmt.Printf("\n%s: %s / %s\n", mod.Name, mod.Status, mod.Result)
		fmt.Printf("Events: %d success, %d failure, %d warning\n",
			mod.EventCounts["SUCCESS"], mod.EventCounts["FAILURE"], mod.EventCounts["WARNING"])
		fmt.Printf("Details: %s\n", mod.LogURL)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if !summary.Passed {
		return 1
	}

	return 0
}
