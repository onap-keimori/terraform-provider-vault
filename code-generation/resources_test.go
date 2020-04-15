package codegen

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/hashicorp/vault/sdk/framework"
)

const (
	examplePath     = `/transform/role/{name}`
	examplePathItem = `{
  "description": "Read, write, and delete roles.",
  "parameters": [
    {
      "name": "name",
      "description": "The name of the role.",
      "in": "path",
      "schema": {
        "type": "string"
      },
      "required": true
    }
  ],
  "x-vault-createSupported": true,
  "get": {
    "operationId": "getTransformRoleName",
    "tags": [
      "secrets"
    ],
    "responses": {
      "200": {
        "description": "OK"
      }
    }
  },
  "post": {
    "operationId": "postTransformRoleName",
    "tags": [
      "secrets"
    ],
    "requestBody": {
      "content": {
        "application/json": {
          "schema": {
            "type": "object",
            "properties": {
              "transformations": {
                "type": "array",
                "description": "A comma separated string or slice of transformations to use.",
                "items": {
                  "type": "string"
                }
              }
            }
          }
        }
      }
    },
    "responses": {
      "200": {
        "description": "OK"
      }
    }
  },
  "delete": {
    "operationId": "deleteTransformRoleName",
    "tags": [
      "secrets"
    ],
    "responses": {
      "204": {
        "description": "empty body"
      }
    }
  }
}
`
)

func TestGenerateResource(t *testing.T) {
	pathItem := &framework.OASPathItem{}
	if err := json.NewDecoder(bytes.NewReader([]byte(examplePathItem))).Decode(pathItem); err != nil {
		t.Fatal(err)
	}
	if err := generateResource(os.Stdout, examplePath, pathItem); err != nil {
		t.Fatal(err)
	}
}
