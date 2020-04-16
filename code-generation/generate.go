package codegen

import (
	"bufio"
	"html/template"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
)

type FileType int

const (
	FileTypeDataSource FileType = iota
	FileTypeResource
)

func (t FileType) String() string {
	switch t {
	case FileTypeDataSource:
		return "datasources"
	}
	return "resources"
}

var pathToGeneratedCodeDir = func() string {
	repoName := "terraform-provider-vault"
	wd, _ := os.Getwd()
	pathParts := strings.Split(wd, repoName)
	return pathParts[0] + repoName + "/generated/"
}()

func GenerateFile(logger hclog.Logger, fileType FileType, path string, pathItem *framework.OASPathItem) error {
	pathToFile := pathToGeneratedCodeDir + fileType.String() + path + ".go"
	pathToFile = strings.Replace(pathToFile, "{", "", -1)
	pathToFile = strings.Replace(pathToFile, "}", "", -1)
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
			os.Exit(1)
		}
	}()
	w := bufio.NewWriter(f)
	defer func() {
		if err := w.Flush(); err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}()
	if err := generateResource(w, fileType, path, parentDir, pathItem); err != nil {
		return err
	}
	return nil
}

// generateResource takes one pathItem and uses a template to generate code
// for it. This code is written to the given writer.
func generateResource(writer io.Writer, fileType FileType, path, dirName string, pathItem *framework.OASPathItem) error {
	tmpl, err := template.New(fileType.String()).Parse(templates[fileType])
	if err != nil {
		return err
	}
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

	// Only path parameters are included as the original params.
	// For the rest of the params, they're located in the post body
	// of the OpenAPI spec, so let's tack them together.
	postParams := getPostParams(pathItem)

	// There also can be dupes, so let's track all they keys we've
	// seen before putting new ones in.
	unique := make(map[string]bool)
	for _, param := range pathItem.Parameters {
		// We can assume these are already unique because they originated
		// from a map where the key was their name.
		unique[param.Name] = true
	}
	for _, param := range postParams {
		if found := unique[param.Name]; !found {
			pathItem.Parameters = append(pathItem.Parameters, param)
			unique[param.Name] = true
		}
	}

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

func getPostParams(pathItem *framework.OASPathItem) map[string]framework.OASParameter {
	if pathItem.Post == nil {
		return nil
	}
	if pathItem.Post.RequestBody == nil {
		return nil
	}
	if pathItem.Post.RequestBody.Content == nil {
		return nil
	}
	// Collect these in a map to de-duplicate them.
	postParams := make(map[string]framework.OASParameter)
	for _, mediaTypeObject := range pathItem.Post.RequestBody.Content {
		if mediaTypeObject.Schema == nil {
			continue
		}
		if mediaTypeObject.Schema.Properties == nil {
			continue
		}
		for propertyName, schema := range mediaTypeObject.Schema.Properties {
			postParams[propertyName] = framework.OASParameter{
				Name:        propertyName,
				Description: schema.Description,
				In:          "post",
				Schema:      schema,
			}
		}
	}
	return postParams
}
