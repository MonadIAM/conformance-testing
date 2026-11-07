package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"time"
)

type API interface {
	Request(context.Context, string, string, url.Values, []byte) ([]byte, error)
}

type Options struct {
	Plan         string
	Module       string
	Output       string
	BaseURL      string
	Variant      map[string]string
	Timeout      time.Duration
	PollInterval time.Duration
}

type module struct {
	Name    string            `json:"testModule"`
	Variant map[string]string `json:"variant"`
}

func runPlan(ctx context.Context, api API, config json.RawMessage, opts Options) (summary Summary, err error) {
	defer func() {
		if err != nil {
			summary.Passed = false
			summary.Error = err.Error()
		}

		err = errors.Join(err, writeJSON(opts.Output, "summary.json", summary))
	}()

	if !json.Valid(config) {
		return summary, errors.New("invalid configuration JSON")
	}

	if opts.Timeout <= 0 || opts.PollInterval <= 0 {
		return summary, errors.New("timeout and poll interval must be positive")
	}

	if err = writeArtifact(opts.Output, "config.json", config); err != nil {
		return summary, err
	}

	query := url.Values{"planName": {opts.Plan}}
	if len(opts.Variant) > 0 {
		variant, _ := json.Marshal(opts.Variant)
		query.Set("variant", string(variant))
	}

	var plan struct {
		ID      string   `json:"id"`
		Modules []module `json:"modules"`
	}

	if err = request(ctx, api, opts.Output, "plan-created.json", "POST", "/api/plan", query, config, &plan); err != nil {
		return summary, err
	}
	summary.Plan = plan.ID
	if plan.ID == "" {
		return summary, errors.New("suite returned no plan ID")
	}

	for _, mod := range plan.Modules {
		if opts.Module != "" && opts.Module != mod.Name {
			continue
		}

		dir := filepath.Join(opts.Output, fmt.Sprintf("%03d", len(summary.Modules)+1))
		result, moduleErr := runModule(ctx, api, opts, plan.ID, mod, dir)
		summary.Modules = append(summary.Modules, result)

		if moduleErr != nil {
			return summary, moduleErr
		}
	}

	if len(summary.Modules) == 0 {
		return summary, errors.New("no modules matched the selected plan/module")
	}
	summary.Passed = true
	for _, mod := range summary.Modules {
		if mod.Status != "FINISHED" || mod.Result != "PASSED" {
			summary.Passed = false
		}
	}

	return summary, nil
}

func runModule(ctx context.Context, api API, opts Options, planID string, mod module, dir string) (result Module, err error) {
	result = Module{Name: mod.Name, Variant: mod.Variant}

	defer func() {
		if err != nil {
			result.Error = err.Error()
		}
	}()

	variant, _ := json.Marshal(mod.Variant)
	query := url.Values{"test": {mod.Name}, "plan": {planID}}
	if len(mod.Variant) > 0 {
		query.Set("variant", string(variant))
	}

	var test struct {
		ID string `json:"id"`
	}

	if err = request(ctx, api, dir, "test-created.json", "POST", "/api/runner", query, nil, &test); err != nil {
		return result, err
	}

	if test.ID == "" {
		return result, errors.New("suite returned no test ID")
	}

	result.ID = test.ID
	result.LogURL = opts.BaseURL + "/log-detail.html?log=" + url.QueryEscape(test.ID)

	testPath := url.PathEscape(test.ID)
	waitCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	started := false

	for {
		var info struct {
			Status string `json:"status"`
			Result string `json:"result"`
		}

		err = request(waitCtx, api, dir, "info.json", "GET", "/api/info/"+testPath, nil, nil, &info)
		if err != nil {
			break
		}

		result.Status, result.Result = info.Status, info.Result

		if info.Status == "FINISHED" || info.Status == "INTERRUPTED" {
			break
		}

		if info.Status == "CONFIGURED" && !started {
			err = request(waitCtx, api, dir, "started.json", "POST", "/api/runner/"+testPath, nil, nil, nil)
			if err != nil {
				break
			}

			started = true
		}

		select {
		case <-waitCtx.Done():
			err = waitCtx.Err()
		case <-time.After(opts.PollInterval):
		}

		if err != nil {
			break
		}
	}

	var log []struct {
		Result string `json:"result"`
	}

	logErr := request(ctx, api, dir, "log.json", "GET", "/api/log/"+testPath, nil, nil, &log)

	result.EventCounts = make(map[string]int)
	for _, event := range log {
		if event.Result != "" {
			result.EventCounts[event.Result]++
		}
	}

	archiveErr := request(ctx, api, dir, "test-log.zip", "GET", "/api/log/export/"+testPath, nil, nil, nil)
	err = errors.Join(err, logErr, archiveErr)

	return result, err
}

func request(ctx context.Context, api API, dir, file, method, path string, query url.Values, body []byte, target any) error {
	data, err := api.Request(ctx, method, path, query, body)
	if len(data) > 0 {
		err = errors.Join(err, writeArtifact(dir, file, data))
	}

	if err != nil {
		return err
	}

	if target != nil {
		return json.Unmarshal(data, target)
	}

	return nil
}
