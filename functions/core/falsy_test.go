package core

import (
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFalsy_RunRule_Fail(t *testing.T) {

	sampleYaml := `
tags:
  - name: "bad tag 1"
    description: false
  - name: "bad tag 2"
    description: 0
  - name: "bad tag 3"
    description: ""
  - name: "bad tag 4"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, severityError, "falsy", "description", nil)
	ctx := buildCoreTestContext(rule.Then, nil)

	tru := Falsy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 3)
}

func TestFalsy_RunRule_Fail_NoNodes(t *testing.T) {

	sampleYaml := `
notTags:
 - name: "bad tag 1"
   description: false
 - name: "bad tag 2"
   description: 0
 - name: "bad tag 3"
   description: ""
 - name: "bad tag 4"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 0)

	rule := buildCoreTestRule(path, severityError, "falsy", "description", nil)
	ctx := buildCoreTestContext(rule.Then, nil)

	tru := Falsy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestFalsy_RunRule_Pass(t *testing.T) {

	sampleYaml := `
tags:
 - name: "good tag 1"
 - name: "bad tag 2"
 - name: "bad tag 3"
   description: ""
 - name: "good Tag 2"
   description: "a nice description"`

	path := "$.tags[*]"

	nodes, _ := utils.FindNodes([]byte(sampleYaml), path)
	assert.Len(t, nodes, 4)

	rule := buildCoreTestRule(path, severityError, "Falsy", "description", nil)
	ctx := buildCoreTestContext(rule.Then, nil)

	tru := Falsy{}
	res := tru.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}
