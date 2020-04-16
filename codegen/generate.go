package codegen

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
)

var pathToHomeDir = func() string {
	repoName := "terraform-provider-vault"
	wd, _ := os.Getwd()
	pathParts := strings.Split(wd, repoName)
	return pathParts[0] + repoName
}()

func GenerateFiles(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	if err := generateCode(logger, fileType, path, pathItem); err != nil {
		return err
	}
	if err := generateDoc(logger, fileType, path, pathItem); err != nil {
		return err
	}
	return nil
}

func generateCode(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	pathToFile := cleanFilePath(fmt.Sprintf("%s/generated/%s%s.go", pathToHomeDir, fileType.String(), path))
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
	if err := parseTemplate(w, fileType, path, parentDir, pathItem); err != nil {
		return err
	}
	return nil
}

func cleanFilePath(path string) string {
	path = strings.Replace(path, "{", "", -1)
	path = strings.Replace(path, "}", "", -1)
	return path
}

func toDocName(s string) string {
	if strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	return strings.Replace(s, "/", "-", -1)
}

func generateDoc(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	pathToFile := cleanFilePath(fmt.Sprintf("%s/website/docs/generated/%s/%s.md", pathToHomeDir, fileType.String(), toDocName(path)))
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
	if err := parseTemplate(w, FileTypeDoc, path, parentDir, pathItem); err != nil {
		return err
	}
	return nil
}

// parseTemplate takes one pathItem and uses a template to generate code
// for it. This code is written to the given writer.
func parseTemplate(writer io.Writer, fileType FileType, path, dirName string, pathItem *framework.OASPathItem) error {
	tmpl, err := template.New(fileType.String()).Parse(templates[fileType])
	if err != nil {
		return err
	}
	// TODO toTemplateable could only be called once.
	return tmpl.Execute(writer, toTemplateable(path, dirName, pathItem))
}

// templatable is a convenience struct that plays easily with Go's
// template package.
type templatable struct {
	Endpoint           string
	DirName            string
	ExportedFuncPrefix string
	PrivateFuncPrefix  string
	Parameters         []framework.OASParameter
	SupportsRead       bool
	SupportsWrite      bool
	SupportsDelete     bool
}

// TODO need to generate docs too
// TODO sensitive fields
// TODO what about ForceNew, Computed
// TODO doesn't yet support field types of "object", "array"
func toTemplateable(path, dirName string, pathItem *framework.OASPathItem) *templatable {
	// Isolate the last field in the path and use it to prefix functions
	// to prevent naming collisions.
	pathFields := strings.Split(path, "/")
	lastField := pathFields[0]
	if len(pathFields) > 1 {
		lastField = pathFields[len(pathFields)-1]
	}
	lastField = strings.Replace(lastField, "{", "", -1)
	lastField = strings.Replace(lastField, "}", "", -1)
	lastField = strings.Replace(lastField, "_", "", -1)

	appendPostParamsToTopLevel(pathItem)

	// Sort the parameters by name so they won't shift every time
	// new files are generated.
	sort.Slice(pathItem.Parameters, func(i, j int) bool {
		return pathItem.Parameters[i].Name < pathItem.Parameters[j].Name
	})
	return &templatable{
		Endpoint:           path,
		DirName:            dirName[strings.LastIndex(dirName, "/")+1:],
		ExportedFuncPrefix: strings.Title(strings.ToLower(lastField)),
		PrivateFuncPrefix:  strings.ToLower(lastField),
		Parameters:         pathItem.Parameters,
		SupportsRead:       pathItem.Get != nil,
		SupportsWrite:      pathItem.Post != nil,
		SupportsDelete:     pathItem.Delete != nil,
	}
}

// parameters can be buried deep in the post request body. For
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
