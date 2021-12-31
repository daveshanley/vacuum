package model

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"testing"
)

func TestResolveDocument(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("test_files/burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)

	resolved, _ := ResolveDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile("/tmp/piiiiiza.yaml", b, 777)
	assert.NotNil(t, b)

}

func TestResolveDocument_Remote(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("test_files/remote-burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)

	resolved, _ := ResolveDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile("/tmp/piiiiiza-remote.yaml", b, 775)
	assert.NotNil(t, b)

}

func TestResolveDocument_File(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("test_files/localfile-burgershop.openapi.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(burgershop, &rootNode)

	resolved, _ := ResolveDocument(&rootNode)
	assert.NotNil(t, resolved)

	b, err := yaml.Marshal(resolved)
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile("/tmp/piiiiiza-localfile.yaml", b, 775)
	assert.NotNil(t, b)

}
