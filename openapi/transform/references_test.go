package transform

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestValidateBundled(t *testing.T) {
	valid := parseTestYAML(t, `openapi: 3.1.0
paths: {}
components:
  examples:
    Remote:
      externalValue: https://example.com/value.json
  schemas:
    Local:
      $ref: '#/components/schemas/Other'
    Other:
      discriminator:
        mapping:
          local: Local
        defaultMapping: '#/components/schemas/Local'
`)
	require.NoError(t, ValidateBundled(valid))

	for _, source := range []string{
		"openapi: 3.0.0\npaths: {}\nx: {$ref: 'schemas.yaml#/Thing'}\n",
		"openapi: 3.1.0\npaths: {}\nx: {$dynamicRef: 'https://example.com/schema#thing'}\n",
		"openapi: 3.0.0\npaths: {}\nx: {operationRef: '../operations.yaml#/get'}\n",
		"openapi: 3.0.0\npaths: {}\nx: {discriminator: {mapping: {x: 'schemas.yaml#/X'}}}\n",
		"openapi: 3.2.0\npaths: {}\nx: {discriminator: {defaultMapping: 'schemas.yaml#/X'}}\n",
	} {
		err := ValidateBundled(parseTestYAML(t, source))
		assert.ErrorContains(t, err, "require a bundled OpenAPI document")
		assert.Contains(t, err.Error(), "$")
	}
	assert.Error(t, ValidateBundled(&yaml.Node{}))
}

func TestValidateBundledIgnoresReferencesInArbitraryData(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /things:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              example:
                $ref: https://example.com/payload-data
components:
  examples:
    Payload:
      value:
        operationRef: ../also-payload-data.yaml
  schemas:
    ExampleProperty:
      properties:
        example:
          $ref: '#/components/schemas/Target'
    SchemaExamples:
      examples:
        - {$ref: https://example.com/schema-example-data}
    Target: {type: string}
x-publication-metadata:
  $ref: https://example.com/extension-data
`)
	require.NoError(t, ValidateBundled(root))
}

func TestLocalPointerParsing(t *testing.T) {
	tokens, err := localPointerTokens("#/components/schemas/Foo%20Bar")
	require.NoError(t, err)
	assert.Equal(t, []string{"components", "schemas", "Foo Bar"}, tokens)
	tokens, err = localPointerTokens("#/components/schemas/Foo~1Bar~0Baz")
	require.NoError(t, err)
	assert.Equal(t, []string{"components", "schemas", "Foo/Bar~Baz"}, tokens)
	tokens, err = localPointerTokens("#anchor")
	require.NoError(t, err)
	assert.Equal(t, []string{"anchor"}, tokens)
	tokens, err = localPointerTokens("#")
	require.NoError(t, err)
	assert.Empty(t, tokens)
	_, err = localPointerTokens("other.yaml#/x")
	assert.ErrorContains(t, err, "not local")
	_, err = localPointerTokens("#/%zz")
	assert.ErrorContains(t, err, "invalid URI fragment")
	_, err = localPointerTokens("#/Bad~")
	assert.ErrorContains(t, err, "invalid escape")
	_, err = localPointerTokens("#/Bad~2")
	assert.ErrorContains(t, err, "invalid escape")
}

func TestYAMLHelpers(t *testing.T) {
	assert.Nil(t, documentRoot(nil))
	scalar := &yaml.Node{Kind: yaml.ScalarNode}
	assert.Same(t, scalar, documentRoot(scalar))
	assert.Nil(t, mapValue(scalar, "x"))
	assert.Equal(t, "$['a']['b\\'c']", jsonPath([]string{"a", "b'c"}))
	assert.False(t, isExternalReference(""))
	assert.False(t, isExternalReference("#/x"))
	assert.True(t, isExternalReference("other.yaml#/x"))
	assert.False(t, isExternalDiscriminatorReference("Pet"))
	assert.False(t, isExternalDiscriminatorReference("Pet.v2"))
	assert.True(t, isExternalDiscriminatorReference("../pet.yaml#/Pet"))
	assert.True(t, isExternalDiscriminatorReference("pet.yaml"))
}
