package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"terraform-provider-ionosdim/pkg/dim"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"gopkg.in/yaml.v3"
)

func toYaml(v interface{}) (string, error) {
	bytes, err := yaml.Marshal(v)
	return string(bytes), err
}

func toJson(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	return string(bytes), err
}

func main() {

	dimEndpoint := os.Getenv("IONOSDIM_ENDPOINT")
	dimUsername := os.Getenv("IONOSDIM_USERNAME")
	dimPassword := os.Getenv("IONOSDIM_PASSWORD")
	dimToken := os.Getenv("IONOSDIM_TOKEN")

	dimTokenFile := flag.String("token", "", "name of the file with session token (cookie) for a DIM account")
	dimEndpointCla := flag.String("endpoint", "", "DIM endpoint URL")
	dimFunc := flag.String("func", "server_info", "dim function")
	dimFuncArgs := flag.String("args", "[]", "dim function args (as json array)")
	outJson := flag.Bool("j", false, "output as json instead of yaml")
	flag.Parse()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.AllowError())
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	level.Info(logger).Log("msg", "starting dim cli")

	var dimTokenCla string
	if *dimTokenFile != "" {
		_token, err := os.ReadFile(*dimTokenFile)
		if err != nil {
			level.Error(logger).Log("msg", "could not read dim token file", "file", *dimTokenFile, "err", err)
			os.Exit(1)
		}
		dimTokenCla = strings.TrimSpace(string(_token))
	}

	var _dimFuncArgs interface{}

	err := json.Unmarshal([]byte(*dimFuncArgs), &_dimFuncArgs)
	if err != nil {
		level.Error(logger).Log("msg", "could not unmarshal args", "err", err)
		os.Exit(1)
	}

	if *dimEndpointCla != "" {
		dimEndpoint = *dimEndpointCla
	}

	if dimTokenCla != "" {
		dimToken = dimTokenCla
	}

	if dimEndpoint == "" {
		level.Error(logger).Log("msg", "DIM endpoint must be specified. Set the endpoint value as command line argument or use the IONOSDIM_ENDPOINT environment variable.", "err", err)
		os.Exit(1)
	}

	if dimToken == "" {
		if dimUsername == "" {
			level.Error(logger).Log("msg", "DIM username must be specified. Use the IONOSDIM_USERNAME environment variable. Alternatively you may specify the token value in IONOSDIM_TOKEN environment variable or in the file spicified with token command line argument.", "err", err)
			os.Exit(1)
		}

		if dimPassword == "" {
			level.Error(logger).Log("msg", "DIM password must be specified. Use the IONOSDIM_PASSWORD environment variable. Alternatively you may specify the token value in IONOSDIM_TOKEN environment variable or in the file spicified with token command line argument.", "err", err)
			os.Exit(1)
		}
	}

	dimC, err := dim.NewClient(&dimEndpoint, &dimToken, &dimUsername, &dimPassword, nil)
	resp, err := dimC.RawCall(*dimFunc, _dimFuncArgs)
	if err != nil {
		level.Error(logger).Log("msg", "dim request failed", "err", err)
		os.Exit(1)
	}

	var out string
	if *outJson {
		out, err = toJson(resp)
	} else {
		out, err = toYaml(resp)
	}
	if err != nil {
		level.Error(logger).Log("msg", "could not marshal response output", "err", err)
		os.Exit(1)
	}

	fmt.Println(out)
}
