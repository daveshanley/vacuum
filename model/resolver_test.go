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

func Benchmark_ResolveDocumentBurgerShop(b *testing.B) {
	burgershop, _ := ioutil.ReadFile("test_files/burgershop.openapi.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)
	for n := 0; n < b.N; n++ {
		ResolveOpenAPIDocument(&rootNode)
	}
}

func Benchmark_ResolveDocumentStripe(b *testing.B) {
	burgershop, _ := ioutil.ReadFile("test_files/stripe.yaml")
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
	assert.Len(t, burgershop, 10077)
	assert.Len(t, b, 23353)

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

func TestResolveDocument_Stripe(t *testing.T) {

	stripe, _ := ioutil.ReadFile("test_files/stripe.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(stripe, &rootNode)

	resolved, _ := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}

	//should be x bytes larger after resolving, should also match the same byte size as a remote test.
	before := len(stripe)
	after := len(b)

	assert.Greater(t, after, before)
	//assert.Equal(t, before, 3173977)
	//assert.Equal(t, after, 80745647)

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

	resolved, errs := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)

	// looking up the ref should return a result, which means it was ignored.
	pathValue := "$.paths..$ref"
	path, _ := yamlpath.NewPath(pathValue)
	result, _ := path.Find(resolved)
	assert.NotNil(t, result)
	assert.NotNil(t, errs)
	assert.Len(t, errs, 1)
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
      description: Puppy thing
      properties:
        kitty: 
          $ref: '#/components/schemas/Kitty'
    Kitty:
      properties:
        description: Kitty Thing
        puppy:  
          $ref: '#/components/schemas/Puppy'`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, errs := ResolveOpenAPIDocument(&rootNode)
	assert.NotNil(t, resolved)
	assert.Len(t, errs, 1)

	//d, _ := yaml.Marshal(&rootNode)
	//fmt.Print(d)

}

func TestResolveDocument_Parameter_Resolving(t *testing.T) {

	yml := `components: 
  schemas:
    Puppy:
      type: string
  parameters:
    Louie:
      in: query
      name: louie
    Chewy:
      $ref: '#/components/parameters/Louie'
paths:
  /naughty/{puppy}:
    parameters:
      - $ref: '#/components/parameters/Chewy'
    get:
      responses:
        "200":
          description: The naughty pup
          content:
            application/json:
              schema:
                  $ref: '#/components/schemas/Puppy'
`

	spec := strings.TrimSpace(yml)

	var rootNode yaml.Node
	yaml.Unmarshal([]byte(spec), &rootNode)

	resolved, errs := ResolveOpenAPIDocument(&rootNode)

	b, _ := yaml.Marshal(resolved)
	assert.NotNil(t, b)
	assert.NotNil(t, resolved)
	assert.Len(t, errs, 1)
	assert.Equal(t, "component '#/components/parameters/Chewy' cannot be resolved", errs[0].Error.Error())
}
