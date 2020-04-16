package codegen

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/strutil"
)

var (
	ErrUnsupported = errors.New("code and doc generation for this item is unsupported")

	supportedParamTypes = []string{
		"string",
		"boolean",
		"integer",
		"array", // We presently only support string arrays.
	}

	pathToHomeDir = func() string {
		repoName := "terraform-provider-vault"
		wd, _ := os.Getwd()
		pathParts := strings.Split(wd, repoName)
		return pathParts[0] + repoName
	}()
)

// GenerateFiles is used to generate the code and doc for one single resource
// or data source. For example, if you provided it with the path
// "/transform/transformation/{name}" and a fileType of Resource, it would
// generate both the Go code for the resource, and a starter doc for it.
// Tests are not generated at this time because we'd prefer human eyes and hands
// on the generated code before including it in the provider.
func GenerateFiles(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	if err := generateCode(logger, fileType, path, pathItem); err != nil {
		return err
	}
	if err := generateDoc(logger, fileType, path, pathItem); err != nil {
		return err
	}
	return nil
}

// generateCode generates the code (and file) for either one resource, or one data source.
func generateCode(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	pathToFile := stripCurlyBraces(fmt.Sprintf("%s/generated/%s%s.go", pathToHomeDir, fileType.String(), path))
	parentDir := pathToFile[:strings.LastIndex(pathToFile, "/")]
	if err := os.MkdirAll(parentDir, 0775); err != nil {
		return err
	}
	f, err := os.Create(pathToFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()
	w := bufio.NewWriter(f)
	defer func() {
		if err := w.Flush(); err != nil {
			logger.Error(err.Error())
		}
	}()
	if err := parseTemplate(logger, w, fileType, path, parentDir, pathItem); err != nil {
		return err
	}
	return nil
}

// generateDoc generates the doc for a resource or data source, and the file for it.
// The file is incomplete with a number of placeholders for the author to fill in
// additional information.
func generateDoc(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	pathToFile := stripCurlyBraces(fmt.Sprintf("%s/website/docs/generated/%s/%s.md", pathToHomeDir, fileType.String(), replaceSlashesWithDashes(path)))
	parentDir := pathToFile[:strings.LastIndex(pathToFile, "/")]
	if err := os.MkdirAll(parentDir, 0775); err != nil {
		return err
	}
	f, err := os.Create(pathToFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.Error(err.Error())
		}
	}()
	w := bufio.NewWriter(f)
	defer func() {
		if err := w.Flush(); err != nil {
			logger.Error(err.Error())
		}
	}()
	if err := parseTemplate(logger, w, FileTypeDoc, path, parentDir, pathItem); err != nil {
		return err
	}
	return nil
}

// parseTemplate takes one pathItem and uses a template to generate text
// for it. This test is written to the given writer.
func parseTemplate(logger hclog.Logger, writer io.Writer, fileType FileType, path, dirName string, pathItem *framework.OASPathItem) error {
	tmpl, err := template.New(fileType.String()).Parse(templates[fileType])
	if err != nil {
		return err
	}
	tmplFriendly, err := toTemplateFriendly(logger, path, dirName, pathItem)
	if err != nil {
		return err
	}
	return tmpl.Execute(writer, tmplFriendly)
}

// templateFriendly is a convenience struct that plays nicely with Go's
// template package.
type templateFriendly struct {
	Endpoint           string
	DirName            string
	ExportedFuncPrefix string
	PrivateFuncPrefix  string
	Parameters         []framework.OASParameter
	SupportsRead       bool
	SupportsWrite      bool
	SupportsDelete     bool
}

// TODO sensitive fields
// TODO what about ForceNew, Computed
// toTemplateFriendly does a bunch of work to format the given data into a
// struct that has fields that will be idiomatic to use with Go's templating
// language.
func toTemplateFriendly(logger hclog.Logger, path, dirName string, pathItem *framework.OASPathItem) (*templateFriendly, error) {
	// Isolate the last field in the path and use it to prefix functions
	// to prevent naming collisions.
	pathFields := strings.Split(path, "/")
	prefix := pathFields[0]
	if len(pathFields) > 1 {
		prefix = pathFields[len(pathFields)-1]
	}
	prefix = stripCurlyBraces(prefix)

	// We don't want snake case for the field name in Go code.
	prefix = strings.Replace(prefix, "_", "", -1)

	appendPostParamsToTopLevel(pathItem)

	// Validate that we don't have any unsupported parameters.
	for _, param := range pathItem.Parameters {
		if !strutil.StrListContains(supportedParamTypes, param.Schema.Type) {
			logger.Error(fmt.Sprintf(`can't generate %q because parameter type of %q for %s is unsupported'`, path, param.Schema.Type, param.Name))
			return nil, ErrUnsupported
		}
	}

	// Sort the parameters by name so they won't shift every time
	// new files are generated due to having originated in maps.
	sort.Slice(pathItem.Parameters, func(i, j int) bool {
		return pathItem.Parameters[i].Name < pathItem.Parameters[j].Name
	})
	return &templateFriendly{
		Endpoint:           path,
		DirName:            dirName[strings.LastIndex(dirName, "/")+1:],
		ExportedFuncPrefix: strings.Title(strings.ToLower(prefix)),
		PrivateFuncPrefix:  strings.ToLower(prefix),
		Parameters:         pathItem.Parameters,
		SupportsRead:       pathItem.Get != nil,
		SupportsWrite:      pathItem.Post != nil,
		SupportsDelete:     pathItem.Delete != nil,
	}, nil
}

// Parameters can be buried deep in the post request body. For
// convenience during templating, we dig down and grab those,
// and just put them at the top level with the rest.
func appendPostParamsToTopLevel(pathItem *framework.OASPathItem) {
	if pathItem.Post == nil {
		return
	}
	if pathItem.Post.RequestBody == nil {
		return
	}
	if pathItem.Post.RequestBody.Content == nil {
		return
	}
	// There also can be dupes, so let's track all they keys we've
	// seen before putting new ones in.
	unique := make(map[string]bool)
	for _, param := range pathItem.Parameters {
		// We can assume these are already unique because they originated
		// from a map where the key was their name.
		unique[param.Name] = true
	}
	for _, mediaTypeObject := range pathItem.Post.RequestBody.Content {
		if mediaTypeObject.Schema == nil {
			continue
		}
		if mediaTypeObject.Schema.Properties == nil {
			continue
		}
		for propertyName, schema := range mediaTypeObject.Schema.Properties {
			if ok := unique[propertyName]; ok {
				continue
			}
			pathItem.Parameters = append(pathItem.Parameters, framework.OASParameter{
				Name:        propertyName,
				Description: schema.Description,
				In:          "post",
				Schema:      schema,
			})
			unique[propertyName] = true
		}
	}
}

// replaceSlashesWithDashes converts a path like "/transform/transformation/{name}"
// to "transform-transformation-{name}". Note that it trims leading slashes.
func replaceSlashesWithDashes(s string) string {
	if strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	return strings.Replace(s, "/", "-", -1)
}

// stripCurlyBraces converts a path like
// "generated/resources/transform-transformation-{name}.go"
// to "generated/resources/transform-transformation-name.go".
func stripCurlyBraces(path string) string {
	path = strings.Replace(path, "{", "", -1)
	path = strings.Replace(path, "}", "", -1)
	return path
}
