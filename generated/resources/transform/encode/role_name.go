package encode

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/vault/api"
	"github.com/terraform-providers/terraform-provider-vault/util"
)

const rolenameEndpoint = "/transform/encode/{role_name}"

func RolenameResource() *schema.Resource {
	fields := map[string]*schema.Schema{
		"path": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			StateFunc: func(v interface{}) string {
				return strings.Trim(v.(string), "/")
			},
		},
		"role_name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name of the role.",
		},
		"tweak": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The tweak value to use. Only applicable for FPE transformations",
		},
		"value": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The value in which to encode.",
		},
		"batch_input": {
			Optional:    true,
			Description: "Specifies a list of items to be encoded in a single batch. If this parameter is set, the parameters &#39;value&#39;, &#39;transformation&#39; and &#39;tweak&#39; will be ignored. Each batch item within the list can specify these parameters instead.",
		},
		"transformation": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The transformation to perform. If no value is provided and the role contains a single transformation, this value will be inferred from the role.",
		},
	}
	return &schema.Resource{
		Create: rolenameCreateResource,
		Update: rolenameUpdateResource,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: fields,
	}
}
func rolenameCreateResource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	backend := d.Get("path").(string)

	data := map[string]interface{}{}
	if v, ok := d.GetOkExists("role_name"); ok {
		data["role_name"] = v
	}
	if v, ok := d.GetOkExists("tweak"); ok {
		data["tweak"] = v
	}
	if v, ok := d.GetOkExists("value"); ok {
		data["value"] = v
	}
	if v, ok := d.GetOkExists("batch_input"); ok {
		data["batch_input"] = v
	}
	if v, ok := d.GetOkExists("transformation"); ok {
		data["transformation"] = v
	}

	path := util.ReplacePathParameters(backend+rolenameEndpoint, d)
	log.Printf("[DEBUG] Writing %q", path)
	_, err := client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("error writing %q: %s", path, err)
	}
	d.SetId(path)
	log.Printf("[DEBUG] Wrote %q", path)
	return rolenameReadResource(d, meta)
}

func rolenameUpdateResource(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	path := d.Id()

	log.Printf("[DEBUG] Updating %q", path)

	data := map[string]interface{}{}
	if d.HasChange("role_name") {
		data["role_name"] = d.Get("role_name")
	}
	if d.HasChange("tweak") {
		data["tweak"] = d.Get("tweak")
	}
	if d.HasChange("value") {
		data["value"] = d.Get("value")
	}
	if d.HasChange("batch_input") {
		data["batch_input"] = d.Get("batch_input")
	}
	if d.HasChange("transformation") {
		data["transformation"] = d.Get("transformation")
	}
	defer func() {
		d.SetId(path)
	}()
	_, err := client.Logical().Write(path, data)
	if err != nil {
		return fmt.Errorf("error updating template auth backend role %q: %s", path, err)
	}
	log.Printf("[DEBUG] Updated %q", path)
	return rolenameReadResource(d, meta)
}
