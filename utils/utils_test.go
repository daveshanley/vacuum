package utils

import (
	_ "embed"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"sync"
	"testing"
)

type petstore []byte

var once sync.Once

var (
	psBytes petstore
)

func getPetstore() petstore {
	once.Do(func() {
		psBytes, _ = ioutil.ReadFile("../model/test_files/petstorev3.json")
	})
	return psBytes
}

func TestRenderCodeSnippet(t *testing.T) {
	code := []string{"hey", "ho", "let's", "go!"}
	startNode := &yaml.Node{
		Line: 1,
	}
	rendered := RenderCodeSnippet(startNode, code, 1, 3)
	assert.Equal(t, "hey\nho\nlet's\n", rendered)

}

func TestFindNodes(t *testing.T) {
	nodes, err := FindNodes(getPetstore(), "$.info.contact")
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.Len(t, nodes, 1)
}

func TestFindNodes_BadPath(t *testing.T) {
	nodes, err := FindNodes(getPetstore(), "I am not valid")
	assert.Error(t, err)
	assert.Nil(t, nodes)
}

func TestFindLastChildNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$.info")
	lastNode := FindLastChildNode(nodes[0])
	assert.Equal(t, "1.0.11", lastNode.Value) // should be the version.
}

func TestFindLastChildNode_WithKids(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$.paths./pet")
	lastNode := FindLastChildNode(nodes[0])
	assert.Equal(t, "read:pets", lastNode.Value)
}

func TestFindLastChildNode_NotFound(t *testing.T) {
	node := &yaml.Node{
		Value: "same",
	}
	lastNode := FindLastChildNode(node)
	assert.Equal(t, "same", lastNode.Value) // should be the same node
}

func TestBuildPath(t *testing.T) {

	assert.Equal(t, "$.fresh.fish.and.chicken.nuggets",
		BuildPath("$.fresh.fish", []string{"and", "chicken", "nuggets"}))
}

func TestBuildPath_WithTrailingPeriod(t *testing.T) {

	assert.Equal(t, "$.fresh.fish.and.chicken.nuggets",
		BuildPath("$.fresh.fish", []string{"and", "chicken", "nuggets", ""}))
}

func TestFindNodesWithoutDeserializing(t *testing.T) {
	root, err := FindNodes(getPetstore(), "$")
	nodes, err := FindNodesWithoutDeserializing(root[0], "$.info.contact")
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.Len(t, nodes, 1)
}

func TestFindNodesWithoutDeserializing_InvalidPath(t *testing.T) {
	root, err := FindNodes(getPetstore(), "$")
	nodes, err := FindNodesWithoutDeserializing(root[0], "I love a good curry")
	assert.Error(t, err)
	assert.Nil(t, nodes)
}

func TestConvertInterfaceIntoStringMap(t *testing.T) {
	var d interface{}
	n := make(map[string]string)
	n["melody"] = "baby girl"
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "baby girl", parsed["melody"])
}

func TestConvertInterfaceIntoStringMap_NoType(t *testing.T) {
	var d interface{}
	n := make(map[string]interface{})
	n["melody"] = "baby girl"
	d = n
	parsed := ConvertInterfaceIntoStringMap(d)
	assert.Equal(t, "baby girl", parsed["melody"])
}

func TestConvertInterfaceToStringArray(t *testing.T) {
	var d interface{}
	n := make(map[string][]string)
	n["melody"] = []string{"melody", "is", "my", "baby"}
	d = n
	parsed := ConvertInterfaceToStringArray(d)
	assert.Equal(t, "baby", parsed[3])
}

func TestConvertInterfaceToStringArray_NoType(t *testing.T) {
	var d interface{}
	m := make([]interface{}, 4)
	n := make(map[string]interface{})
	m[0] = "melody"
	m[1] = "is"
	m[2] = "my"
	m[3] = "baby"
	n["melody"] = m
	d = n
	parsed := ConvertInterfaceToStringArray(d)
	assert.Equal(t, "baby", parsed[3])
}

func TestConvertInterfaceToStringArray_Invalid(t *testing.T) {
	var d interface{}
	d = "I am a carrot"
	parsed := ConvertInterfaceToStringArray(d)
	assert.Nil(t, parsed)
}

func TestConvertInterfaceArrayToStringArray(t *testing.T) {
	var d interface{}
	m := []string{"maddox", "is", "my", "little", "champion"}
	d = m
	parsed := ConvertInterfaceArrayToStringArray(d)
	assert.Equal(t, "little", parsed[3])
}

func TestConvertInterfaceArrayToStringArray_NoType(t *testing.T) {
	var d interface{}
	m := make([]interface{}, 4)
	m[0] = "melody"
	m[1] = "is"
	m[2] = "my"
	m[3] = "baby"
	d = m
	parsed := ConvertInterfaceArrayToStringArray(d)
	assert.Equal(t, "baby", parsed[3])
}

func TestConvertInterfaceArrayToStringArray_Invalid(t *testing.T) {
	var d interface{}
	d = "weed is good"
	parsed := ConvertInterfaceArrayToStringArray(d)
	assert.Nil(t, parsed)
}

func TestExtractValueFromInterfaceMap(t *testing.T) {
	var d interface{}
	m := make(map[string][]string)
	m["melody"] = []string{"is", "my", "baby"}
	d = m
	parsed := ExtractValueFromInterfaceMap("melody", d)
	assert.Equal(t, "baby", parsed.([]string)[2])
}

func TestExtractValueFromInterfaceMap_NoType(t *testing.T) {
	var d interface{}
	m := make(map[string]interface{})
	n := make([]interface{}, 3)
	n[0] = "maddy"
	n[1] = "the"
	n[2] = "champion"
	m["maddy"] = n
	d = m
	parsed := ExtractValueFromInterfaceMap("maddy", d)
	assert.Equal(t, "champion", parsed.([]interface{})[2])
}

func TestExtractValueFromInterfaceMap_Flat(t *testing.T) {
	var d interface{}
	m := make(map[string]interface{})
	m["maddy"] = "niblet"
	d = m
	parsed := ExtractValueFromInterfaceMap("maddy", d)
	assert.Equal(t, "niblet", parsed.(interface{}))
}

func TestExtractValueFromInterfaceMap_NotFound(t *testing.T) {
	var d interface{}
	d = "not a map"
	parsed := ExtractValueFromInterfaceMap("melody", d)
	assert.Nil(t, parsed)
}

func TestFindFirstKeyNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	key, value := FindFirstKeyNode("operationId", nodes, 0)
	assert.NotNil(t, key)
	assert.NotNil(t, value)
	assert.Equal(t, 55, key.Line)
}

func TestFindFirstKeyNode_NotFound(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	key, value := FindFirstKeyNode("i-do-not-exist-in-the-doc", nodes, 0)
	assert.Nil(t, key)
	assert.Nil(t, value)
}

func TestFindFirstKeyNode_Map(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	key, value := FindFirstKeyNode("pet", nodes, 0)
	assert.NotNil(t, key)
	assert.NotNil(t, value)
	assert.Equal(t, 27, key.Line)
}

func TestFindKeyNodeTop(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNodeTop("info", nodes[0].Content)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, 3, k.Line)
}

func TestFindKeyNodeTop_NotFound(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNodeTop("i am a giant potato", nodes[0].Content)
	assert.Nil(t, k)
	assert.Nil(t, v)
}

func TestFindKeyNode(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNode("/pet", nodes[0].Content)
	assert.NotNil(t, k)
	assert.NotNil(t, v)
	assert.Equal(t, 47, k.Line)
}

func TestFindKeyNode_NotFound(t *testing.T) {
	nodes, _ := FindNodes(getPetstore(), "$")
	k, v := FindKeyNode("I am not anything at all", nodes[0].Content)
	assert.Nil(t, k)
	assert.Nil(t, v)
}

func TestMakeTagReadable(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!map",
	}
	assert.Equal(t, ObjectLabel, MakeTagReadable(n))
	n.Tag = "!!seq"
	assert.Equal(t, ArrayLabel, MakeTagReadable(n))
	n.Tag = "!!str"
	assert.Equal(t, StringLabel, MakeTagReadable(n))
	n.Tag = "!!int"
	assert.Equal(t, IntegerLabel, MakeTagReadable(n))
	n.Tag = "!!float"
	assert.Equal(t, NumberLabel, MakeTagReadable(n))
	n.Tag = "!!bool"
	assert.Equal(t, BooleanLabel, MakeTagReadable(n))
	n.Tag = "mr potato man is here"
	assert.Equal(t, "unknown", MakeTagReadable(n))
}

func TestIsNodeMap(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!map",
	}
	assert.True(t, IsNodeMap(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeMap(n))
}

func TestIsNodeMap_Nil(t *testing.T) {
	assert.False(t, IsNodeMap(nil))
}

func TestIsNodePolyMorphic(t *testing.T) {
	n := &yaml.Node{
		Content: []*yaml.Node{
			{
				Value: "anyOf",
			},
		},
	}
	assert.True(t, IsNodePolyMorphic(n))
	n.Content[0].Value = "cakes"
	assert.False(t, IsNodePolyMorphic(n))
}

func TestIsNodeArray(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!seq",
	}
	assert.True(t, IsNodeArray(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeArray(n))
}

func TestIsNodeArray_Nil(t *testing.T) {
	assert.False(t, IsNodeArray(nil))
}

func TestIsNodeStringValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!str",
	}
	assert.True(t, IsNodeStringValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeStringValue(n))
}

func TestIsNodeStringValue_Nil(t *testing.T) {
	assert.False(t, IsNodeStringValue(nil))
}

func TestIsNodeIntValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!int",
	}
	assert.True(t, IsNodeIntValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeIntValue(n))
}

func TestIsNodeIntValue_Nil(t *testing.T) {
	assert.False(t, IsNodeIntValue(nil))
}

func TestIsNodeFloatValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!float",
	}
	assert.True(t, IsNodeFloatValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeFloatValue(n))
}

func TestIsNodeFloatValue_Nil(t *testing.T) {
	assert.False(t, IsNodeFloatValue(nil))
}

func TestIsNodeBoolValue(t *testing.T) {
	n := &yaml.Node{
		Tag: "!!bool",
	}
	assert.True(t, IsNodeBoolValue(n))
	n.Tag = "!!pizza"
	assert.False(t, IsNodeBoolValue(n))
}

func TestIsNodeBoolValue_Nil(t *testing.T) {
	assert.False(t, IsNodeBoolValue(nil))
}

func TestFixContext(t *testing.T) {
	assert.Equal(t, "$.nuggets[12].name", FixContext("(root).nuggets.12.name"))
}

func TestFixContext_HttpCode(t *testing.T) {
	assert.Equal(t, "$.nuggets.404.name", FixContext("(root).nuggets.404.name"))
}
