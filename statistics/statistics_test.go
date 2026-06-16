package statistics

import (
	"context"
	asyncapi_context "github.com/daveshanley/vacuum/asyncapi"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/libasyncapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"os"
	"testing"
	"time"
)

func TestCreateReportStatistics(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := os.ReadFile("../model/test_files/petstorev3.json")

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	assert.Equal(t, 30, stats.FilesizeKB)
	assert.Equal(t, 7, stats.References)
	assert.Equal(t, 9, stats.Parameters)
}

func TestCreateReportStatistics_AlmostPerfect(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet: selectedRS,
		Spec:    specBytes,
	})

	resultSet := model.NewRuleResultSet(ruleset.Results)
	stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

	//assert.Equal(t, 100, stats.OverallScore)
	// new missing examples function is now strict / correct
	assert.GreaterOrEqual(t, stats.OverallScore, 98)

}

func TestCreateReportStatistics_BigLoadOfIssues(t *testing.T) {

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	specBytes, _ := os.ReadFile("../model/test_files/api.github.com.yaml")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	d := make(chan bool)
	go func(f chan bool) {

		ruleset := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:     selectedRS,
			Spec:        specBytes,
			AllowLookup: true,
		})
		resultSet := model.NewRuleResultSet(ruleset.Results)
		stats := CreateReportStatistics(ruleset.Index, ruleset.SpecInfo, resultSet)

		assert.Equal(t, 10, stats.OverallScore)
		f <- true
	}(d)

	select {
	case <-ctx.Done():
		assert.Fail(t, "Timed out, we have an issue that needs fixing")
	case <-d:
		break
	}
}

func TestCreateReportStatistics_AsyncAPI(t *testing.T) {
	specBytes := []byte(`asyncapi: 3.1.0
info:
  title: Test
  version: 1.0.0
servers:
  production:
    host: api.example.com
    protocol: mqtt
channels:
  userChannel:
    address: users/{userId}
    parameters:
      userId: {}
    messages:
      - $ref: '#/components/messages/UserCreated'
operations:
  userCreated:
    action: receive
    channel:
      $ref: '#/channels/userChannel'
    messages:
      - $ref: '#/components/messages/UserCreated'
components:
  channels:
    componentChannel:
      address: component/{componentId}
      parameters:
        componentId: {}
  schemas:
    User:
      type: object
      properties:
        parameters:
          type: string
  messages:
    UserCreated:
      payload:
        $ref: '#/components/schemas/User'
  securitySchemes:
    oauth:
      type: oauth2
  replies:
    userReply:
      channel:
        $ref: '#/channels/userChannel'
`)
	asyncCtx, err := asyncapi_context.NewContext(specBytes, "asyncapi.yaml", libasyncapi.NewDocumentConfiguration())
	require.NoError(t, err)

	stats := CreateReportStatistics(asyncCtx.Index, asyncCtx.SpecInfo, model.NewRuleResultSet(nil))

	require.NotNil(t, stats)
	assert.Equal(t, "asyncapi", stats.SpecType)
	assert.Equal(t, 1, stats.Servers)
	assert.Equal(t, 2, stats.Channels)
	assert.Equal(t, 1, stats.Operations)
	assert.Equal(t, 2, stats.Messages)
	assert.Equal(t, 1, stats.Schemas)
	assert.Equal(t, 2, stats.Parameters)
	assert.Equal(t, 1, stats.Security)
	assert.Equal(t, 1, stats.Replies)
	assert.Equal(t, 3, stats.References)
	assert.Equal(t, 0, stats.Paths)
}
