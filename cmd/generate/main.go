package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	codegen "github.com/terraform-providers/terraform-provider-vault/code-generation"
)

var pathToOpenAPIDoc = flag.String("openapi-doc", "", "path/to/openapi.json")

func main() {
	logger := hclog.Default()
	flag.Parse()
	if pathToOpenAPIDoc == nil || *pathToOpenAPIDoc == "" {
		logger.Info("'openapi-doc' is required")
		os.Exit(1)
	}
	doc, err := ioutil.ReadFile(*pathToOpenAPIDoc)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Read in Vault's description of all the supported endpoints, their methods, and more.
	oasDoc := &framework.OASDocument{}
	if err := json.NewDecoder(bytes.NewBuffer(doc)).Decode(oasDoc); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	for path, pathItem := range oasDoc.Paths {
		for _, allowedPath := range codegen.AllowedPaths {
			if !strings.HasPrefix(path, allowedPath) {
				continue
			}
			if err := codegen.GenerateResourceFile(logger, path, pathItem); err != nil {
				logger.Error(err.Error())
				os.Exit(1)
			}
		}
	}
}
