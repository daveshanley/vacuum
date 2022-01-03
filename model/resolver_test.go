package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"strings"
	"testing"
)

func Benchmark_ResolveDocument(b *testing.B) {
	burgershop, _ := ioutil.ReadFile("test_files/burgershop.openapi.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)
	for n := 0; n < b.N; n++ {
		ResolveOpenAPIDocument(&rootNode)
	}
}

func TestResolveDocument(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("test_files/burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}

	// should be x bytes larger after resolving.
	assert.Len(t, burgershop, 10079)
	assert.Len(t, b, 23357)

}

func TestResolveDocument_Remote(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("test_files/remote-burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}
	// should be x bytes larger after resolving.
	assert.Len(t, burgershop, 8889)
	assert.Len(t, b, 18205)

}

func TestResolveDocument_File(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("test_files/localfile-burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}

	// should be x bytes larger after resolving, should also match the same byte size as a remote test.
	assert.Len(t, burgershop, 7993)
	assert.Len(t, b, 18205)

}

func TestResolveDocument_ValidRef(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Puppy'
components:
  schemas:
    Puppy:
      type: string`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return no result, which means it was correctly resolved
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.Nil(t, result)

}

func TestResolveDocument_InvalidRef(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'nowhere#/components/schemas/Buppy'
components:
  schemas:
    Puppy:
      type: string`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_MalformedRef(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/[]..\.(@)..components/schemas/Buppy'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_ValidButMissingRef(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_RemoteBorkedRef(t *testing.T) {

	// TODO, technically this is allowed in JSONSchema, but it's really not helpful.
	// so for now, I am going to ignore it, until the need arises to support it.

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'http://nowhere/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_Remote_HTTP_Fail(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'http://nowhere/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_Remote_HTTP_Fail_InvalidDocType(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'https://quobix.com/images/quobix-logo.png#/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_Remote_HTTP_InvalidHostFail(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'https://kajhsdjlahsdkjah981238712.com#/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_File_Fail(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'nice-but-missing.json#/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_File_Fail_InvalidPath(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: 'nice-but-missing.json/components/schemas/DoesNotExist'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
}

func TestResolveDocument_CircularReferences(t *testing.T) {

	yml := `paths:
  /naughty/{puppy}:
    parameters:
      - in: path
        name: puppy
    get:
      responses:
      "200":
        description: The naughty pup
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Puppy'
components:
  schemas:
    Puppy:
      $ref: '#/components/schemas/Kitty'
    Kitty:
      $ref: '#/components/schemas/Puppy'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, errs := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)
	assert.Len(t, errs, 3)

}

func TestResolveDocument_Parameter_Resolving(t *testing.T) {

	yml := `parameters:
    Louie:
        in: query
        name: louie
    Chewy:
        $ref: '#/parameters/Louie'
paths:
    /naughty/{puppy}:
        parameters:
            - $ref: '#/parameters/Chewy'
        get:
            responses:
            "200":
                description: The naughty pup
                content:
                    application/json:
                        schema:
                            $ref: '#/components/schemas/Puppy'
components:
    schemas:
        Puppy:
            type: string`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, errs := ResolveOpenAPIDocument(&rootNode)

	b, _ := yaml.Marshal(resolved)
	assert.NotNil(t, b)
	assert.NotNil(t, resolved)
	assert.Len(t, errs, 0)

}
