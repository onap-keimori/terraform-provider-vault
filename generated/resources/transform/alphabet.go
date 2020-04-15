package transform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/vault/api"
)

func AlphabetResource() *schema.Resource {
	fields := map[string]*schema.Schema{
		"path": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			StateFunc: func(v interface{}) string {
				return strings.Trim(v.(string), "/")
			},
		},
	}
	return &schema.Resource{
		Read:   alphabetReadResource,
		Exists: alphabetResourceExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: fields,
	}
}
func alphabetReadResource(d *schema.ResourceData, meta interface{}) error {
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
	return nil
}

func alphabetResourceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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
