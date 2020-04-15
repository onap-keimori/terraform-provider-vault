package codegen

import (
	"bufio"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
)

var pathToGeneratedCodeDir = func() string {
	repoName := "terraform-provider-vault"
	wd, _ := os.Getwd()
	pathParts := strings.Split(wd, repoName)
	return pathParts[0] + repoName + "/generated/"
}()

func GenerateResourceFile(logger hclog.Logger, path string, pathItem *framework.OASPathItem) error {
	pathToFile := pathToGeneratedCodeDir + "resources" + path + ".go"
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
	if err := generateResource(w, path, parentDir, pathItem); err != nil {
		return err
	}
	return nil
}

// generateResource takes one pathItem and uses a template to generate code
// for it. This code is written to the given writer.
func generateResource(writer io.Writer, path, dirName string, pathItem *framework.OASPathItem) error {
	tmpl, err := template.New("resourceTemplate").Parse(resourceTemplate)
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
	for _, postParam := range postParams {
		pathItem.Parameters = append(pathItem.Parameters, postParam)
	}
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

func getPostParams(pathItem *framework.OASPathItem) []framework.OASParameter {
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
	var toReturn []framework.OASParameter
	for _, param := range postParams {
		toReturn = append(toReturn, param)
	}
	return toReturn
}

const resourceTemplate = `package {{.DirName}}

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/vault/api"
	{{- if .SupportsWrite }}
	"github.com/terraform-providers/terraform-provider-vault/util"
	{{- end }}
)

{{- if .SupportsWrite }}
const {{.PrivateFuncPrefix}}Endpoint = "{{ .Endpoint }}"
{{ end }}

func {{.ExportedFuncPrefix}}Resource() *schema.Resource {
	fields := map[string]*schema.Schema{
		"path": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			StateFunc: func(v interface{}) string {
				return strings.Trim(v.(string), "/")
			},
		},
		{{- range .Parameters }}
		"{{.Name}}": {
			{{- if (eq .Schema.Type "string") }}
			Type:        schema.TypeString,
			{{- end }}
			{{- if (eq .Schema.Type "boolean") }}
			Type:        schema.TypeBool,
			{{- end }}
			{{- if (eq .Schema.Type "integer") }}
			Type:        schema.TypeInt,
			{{- end }}
			{{- if (eq .Schema.Type "object") }}
			Type:        schema.TypeInt,
			{{- end }}
			{{- if .Required }}
			Required:    true,
			{{- else }}
			Optional:    true,
			{{- end }}
			Description: "{{.Description}}",
		},
		{{- end }}
	}
	return &schema.Resource{
		{{- if .SupportsWrite }}
		Create: {{.PrivateFuncPrefix}}CreateResource,
		Update: {{.PrivateFuncPrefix}}UpdateResource,
		{{- end }}
		{{- if .SupportsRead }}
		Read:   {{.PrivateFuncPrefix}}ReadResource,
		Exists: {{.PrivateFuncPrefix}}ResourceExists,
		{{- end }}
		{{- if .SupportsDelete }}
		Delete: {{.PrivateFuncPrefix}}DeleteResource,
		{{- end }}
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: fields,
	}
}

{{- if .SupportsWrite }}
func {{.PrivateFuncPrefix}}CreateResource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	backend := d.Get("path").(string)

	data := map[string]interface{}{}
	{{- range .Parameters }}
	if v, ok := d.GetOkExists("{{.Name}}"); ok {
		data["{{.Name}}"] = v
	}
	{{- end }}

	path := util.ReplacePathParameters(backend + {{.PrivateFuncPrefix}}Endpoint, d)
	log.Printf("[DEBUG] Writing %q", path)
	_, err := client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("error writing %q: %s", path, err)
	}
	d.SetId(path)
	log.Printf("[DEBUG] Wrote %q", path)
	return {{.PrivateFuncPrefix}}ReadResource(d, meta)
}
{{ end }}

{{- if .SupportsRead }}
func {{.PrivateFuncPrefix}}ReadResource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	path := d.Id()

	log.Printf("[DEBUG] Reading %q", path)
	resp, err := client.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("error reading %q: %s", path, err)
	}
	log.Printf("[DEBUG] Read %q", path)
	if resp == nil {
		log.Printf("[WARN] %q not found, removing from state", path)
		d.SetId("")
		return nil
	}
	{{- range .Parameters }}
	if err := d.Set("{{.Name}}", resp.Data["{{.Name}}"]); err != nil {
		return fmt.Errorf("error setting state key '{{.Name}}': %s", err)
	}
	{{- end }}
	return nil
}
{{ end }}

{{- if .SupportsWrite }}
func {{.PrivateFuncPrefix}}UpdateResource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	path := d.Id()

	log.Printf("[DEBUG] Updating %q", path)

	data := map[string]interface{}{}
	{{- range .Parameters }}
	if d.HasChange("{{.Name}}") {
		data["{{.Name}}"] = d.Get("{{.Name}}")
	}
	{{- end }}
	defer func() {
		d.SetId(path)
	}()
	_, err := client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("error updating template auth backend role %q: %s", path, err)
	}
	log.Printf("[DEBUG] Updated %q", path)
	return {{.PrivateFuncPrefix}}ReadResource(d, meta)
}
{{ end }}

{{- if .SupportsDelete }}
func {{.PrivateFuncPrefix}}DeleteResource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	path := d.Id()

	log.Printf("[DEBUG] Deleting %q", path)
	_, err := client.Logical().Delete(path)
	if err != nil && !util.Is404(err) {
		return fmt.Errorf("error deleting %q", path)
	} else if err != nil {
		log.Printf("[DEBUG] %q not found, removing from state", path)
		d.SetId("")
		return nil
	}
	log.Printf("[DEBUG] Deleted template auth backend role %q", path)
	return nil
}
{{ end }}

{{- if .SupportsRead }}
func {{.PrivateFuncPrefix}}ResourceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*api.Client)

	path := d.Id()
	log.Printf("[DEBUG] Checking if %q exists", path)

	resp, err := client.Logical().Read(path)
	if err != nil {
		return true, fmt.Errorf("error checking if %q exists: %s", path, err)
	}
	log.Printf("[DEBUG] Checked if %q exists", path)
	return resp != nil, nil
}
{{ end }}
`
