package openapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

func TestAllOfConflicts_GetSchema(t *testing.T) {
	def := AllOfConflicts{}
	assert.Equal(t, "allOfConflicts", def.GetSchema().Name)
}

func TestAllOfConflicts_GetCategory(t *testing.T) {
	def := AllOfConflicts{}
	assert.Equal(t, model.FunctionCategoryOpenAPI, def.GetCategory())
}

func TestAllOfConflicts_RunRule_NoDocument(t *testing.T) {
	def := AllOfConflicts{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Nil(t, res)
}

func TestAllOfConflicts_UntypedParentDetectsConflict(t *testing.T) {
	yml := `openapi: 3.0.4
components:
  schemas:
    Conflict:
      allOf:
        - type: object
          properties:
            kind:
              type: string
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)

	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "kind")
}

func TestAllOfConflicts_ExactUserCase_StringVsNumber(t *testing.T) {
	yml := `openapi: 3.2.0
info:
  title: hey
  version: "1.5"
paths:
  /conflict:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Conflict'
components:
  schemas:
    Base:
      type: object
      properties:
        name:
          type: string
        kind:
          type: string
    Conflict:
      type: object
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)

	assert.Len(t, res, 1)
	assert.Equal(t, "$.components.schemas['Conflict'].allOf", res[0].Path)
	assert.Contains(t, res[0].Message, "property `kind`")
	assert.Contains(t, res[0].Message, "string")
	assert.Contains(t, res[0].Message, "number")
}

func TestAllOfConflicts_UsesCachedAliasProperties(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: cached alias allOf
  version: 1.0.0
paths:
  /first:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Conflict'
  /second:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Conflict'
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Conflict:
      type: object
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            kind:
              type: number`

	ctx, def := buildAllOfConflictsContextWithConfig(t, yml, &drModel.DrConfig{
		UseSchemaCache:     true,
		DeterministicPaths: true,
	})
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "$.components.schemas['Conflict'].allOf", res[0].Path)
	assert.Contains(t, res[0].Message, "property `kind`")
}

func TestAllOfConflicts_ParentPropertyVsAllOfArm(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      properties:
        kind:
          type: string
      allOf:
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)

	assert.Len(t, res, 1)
	assert.Equal(t, "$.components.schemas['Conflict'].allOf", res[0].Path)
}

func TestAllOfConflicts_IntegerAndNumberCompatible(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: integer
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 0)
}

func TestAllOfConflicts_MultiTypeOverlapCompatible(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: [string, "null"]
        - type: object
          properties:
            kind:
              type: ["null"]`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 0)
}

func TestAllOfConflicts_MultiTypeNoOverlapFails(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: [string, "null"]
        - type: object
          properties:
            kind:
              type: [number]`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "kind")
}

func TestAllOfConflicts_OAS30NullableNoFalsePositive(t *testing.T) {
	yml := `openapi: 3.0.3
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: string
              nullable: true
        - type: object
          properties:
            kind:
              type: string`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 0)
}

func TestAllOfConflicts_NestedAllOfConflict(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Mid:
      type: object
      allOf:
        - $ref: '#/components/schemas/Base'
    Conflict:
      type: object
      allOf:
        - $ref: '#/components/schemas/Mid'
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)

	assert.Len(t, res, 1)
	assert.Equal(t, "$.components.schemas['Conflict'].allOf", res[0].Path)
}

func TestAllOfConflicts_RefExpandedConflict(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Conflict:
      type: object
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 1)
}

func TestAllOfConflicts_UntypedPropertySkippedInV1(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Base:
      type: object
      properties:
        kind: {}
    Conflict:
      type: object
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 0)
}

func TestAllOfConflicts_SharedRefTwiceNoDuplicate(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Conflict:
      type: object
      allOf:
        - $ref: '#/components/schemas/Base'
        - $ref: '#/components/schemas/Base'`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 0)
}

func TestAllOfConflicts_SelfCircularAllOfDoesNotLoop(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Self:
      type: object
      properties:
        kind:
          type: string
      allOf:
        - $ref: '#/components/schemas/Self'`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 0)
}

func TestAllOfConflicts_MutualCircularAllOfConflict(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    A:
      type: object
      properties:
        kind:
          type: string
      allOf:
        - $ref: '#/components/schemas/B'
    B:
      type: object
      properties:
        kind:
          type: number
      allOf:
        - $ref: '#/components/schemas/A'`

	res := runAllOfConflicts(t, yml)

	assert.Len(t, res, 2)
	assert.Equal(t, "$.components.schemas['A'].allOf", res[0].Path)
	assert.Equal(t, "$.components.schemas['B'].allOf", res[1].Path)
}

func TestAllOfConflicts_MultiplePropertiesReportSeparately(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: string
            status:
              type: boolean
        - type: object
          properties:
            kind:
              type: number
            status:
              type: object`

	res := runAllOfConflicts(t, yml)

	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Message, "kind")
	assert.Contains(t, res[1].Message, "status")
}

func TestAllOfConflicts_PathsPopulatedFromSharedSchema(t *testing.T) {
	yml := `openapi: 3.1.0
paths:
  /a:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Conflict'
  /b:
    post:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Conflict'
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: string
        - type: object
          properties:
            kind:
              type: number`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 1)
	assert.GreaterOrEqual(t, len(res[0].Paths), 2)
}

func TestAllOfConflicts_ThreeWayConflict(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: string
        - type: object
          properties:
            kind:
              type: number
        - type: object
          properties:
            kind:
              type: boolean`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "string")
	assert.Contains(t, res[0].Message, "number")
	assert.Contains(t, res[0].Message, "boolean")
}

func TestAllOfConflicts_ConflictMixedWithCompatibleProps(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Conflict:
      type: object
      allOf:
        - type: object
          properties:
            kind:
              type: string
            name:
              type: string
        - type: object
          properties:
            kind:
              type: number
            name:
              type: string`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "kind")
	assert.NotContains(t, res[0].Message, "name")
}

func TestAllOfConflicts_DeeplyNestedChainConflict(t *testing.T) {
	yml := `openapi: 3.1.0
components:
  schemas:
    Leaf:
      type: object
      properties:
        kind:
          type: number
    Mid:
      type: object
      allOf:
        - $ref: '#/components/schemas/Leaf'
    Root:
      type: object
      properties:
        kind:
          type: string
      allOf:
        - $ref: '#/components/schemas/Mid'`

	res := runAllOfConflicts(t, yml)
	assert.Len(t, res, 1)
	assert.Equal(t, "$.components.schemas['Root'].allOf", res[0].Path)
	assert.Contains(t, res[0].Message, "kind")
}

func BenchmarkAllOfConflicts_LargeValidAcyclic(b *testing.B) {
	benchmarkAllOfConflictsAnalysis(b, generateValidAllOfChainSpec(200))
}

func BenchmarkAllOfConflicts_LargeValidCircular(b *testing.B) {
	benchmarkAllOfConflictsAnalysis(b, generateValidCircularAllOfSpec(200))
}

func BenchmarkAllOfConflicts_LargeSparseConflicts(b *testing.B) {
	benchmarkAllOfConflictsAnalysis(b, generateSparseConflictAllOfSpec(200, 25))
}

func runAllOfConflicts(t *testing.T, yml string) []model.RuleFunctionResult {
	t.Helper()

	ctx, def := buildAllOfConflictsContext(t, yml)
	return def.RunRule(nil, ctx)
}

type allOfConflictsFixture struct {
	document   libopenapi.Document
	drDocument *drModel.DrDocument
	specInfo   *datamodel.SpecInfo
	rule       model.Rule
}

func buildAllOfConflictsFixture(tb testing.TB, yml string) allOfConflictsFixture {
	tb.Helper()
	return buildAllOfConflictsFixtureWithConfig(tb, yml, nil)
}

func buildAllOfConflictsFixtureWithConfig(tb testing.TB, yml string, config *drModel.DrConfig) allOfConflictsFixture {
	tb.Helper()

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		tb.Fatalf("cannot create new document: %v", err)
	}

	m, err := document.BuildV3Model()
	if err != nil {
		tb.Fatalf("cannot build v3 model: %v", err)
	}

	specInfo, err := datamodel.ExtractSpecInfo([]byte(yml))
	if err != nil {
		tb.Fatalf("cannot extract spec info: %v", err)
	}

	rule := buildOpenApiTestRuleAction("$", "allOfConflicts", "", nil)
	drDocument := drModel.NewDrDocument(m)
	if config != nil {
		drDocument = drModel.NewDrDocumentWithConfig(m, config)
	}

	return allOfConflictsFixture{
		document:   document,
		drDocument: drDocument,
		specInfo:   specInfo,
		rule:       rule,
	}
}

func buildAllOfConflictsContext(t *testing.T, yml string) (model.RuleFunctionContext, AllOfConflicts) {
	t.Helper()

	fixture := buildAllOfConflictsFixture(t, yml)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(fixture.rule.Then), nil)
	ctx.Document = fixture.document
	ctx.DrDocument = fixture.drDocument
	ctx.SpecInfo = fixture.specInfo
	ctx.Rule = &fixture.rule

	return ctx, AllOfConflicts{}
}

func buildAllOfConflictsContextWithConfig(t *testing.T, yml string, config *drModel.DrConfig) (model.RuleFunctionContext, AllOfConflicts) {
	t.Helper()

	fixture := buildAllOfConflictsFixtureWithConfig(t, yml, config)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(fixture.rule.Then), nil)
	ctx.Document = fixture.document
	ctx.DrDocument = fixture.drDocument
	ctx.SpecInfo = fixture.specInfo
	ctx.Rule = &fixture.rule

	return ctx, AllOfConflicts{}
}

var benchmarkAllOfConflictsSink int

// benchmarkAllOfConflictsAnalysis isolates the core graph analysis cost from doctor/model creation.
func benchmarkAllOfConflictsAnalysis(b *testing.B, yml string) {
	fixture := buildAllOfConflictsFixture(b, yml)
	schemas := fixture.drDocument.Schemas
	specInfo := fixture.specInfo

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		graph := newAllOfConflictGraph(schemas, specInfo)
		graph.computeSCCs()
		graph.computeAggregates()
		benchmarkAllOfConflictsSink = len(graph.effectiveAggBySCC)
	}
}

func generateValidAllOfChainSpec(schemaCount int) string {
	var builder strings.Builder
	builder.WriteString("openapi: 3.1.0\ncomponents:\n  schemas:\n")

	for i := 0; i < schemaCount; i++ {
		fmt.Fprintf(&builder, "    Node%d:\n", i)
		builder.WriteString("      type: object\n")
		builder.WriteString("      properties:\n")
		builder.WriteString("        kind:\n          type: string\n")
		fmt.Fprintf(&builder, "        marker%d:\n          type: string\n", i)
		if i == 0 {
			continue
		}
		builder.WriteString("      allOf:\n")
		fmt.Fprintf(&builder, "        - $ref: '#/components/schemas/Node%d'\n", i-1)
	}

	return builder.String()
}

func generateValidCircularAllOfSpec(schemaCount int) string {
	var builder strings.Builder
	builder.WriteString(generateValidAllOfChainSpec(schemaCount))

	if schemaCount > 1 {
		builder.WriteString("    Loop:\n")
		builder.WriteString("      type: object\n")
		builder.WriteString("      properties:\n")
		builder.WriteString("        kind:\n          type: string\n")
		builder.WriteString("      allOf:\n")
		fmt.Fprintf(&builder, "        - $ref: '#/components/schemas/Node%d'\n", schemaCount-1)
		builder.WriteString("        - $ref: '#/components/schemas/LoopBack'\n")
		builder.WriteString("    LoopBack:\n")
		builder.WriteString("      type: object\n")
		builder.WriteString("      properties:\n")
		builder.WriteString("        kind:\n          type: string\n")
		builder.WriteString("      allOf:\n")
		builder.WriteString("        - $ref: '#/components/schemas/Loop'\n")
	}

	return builder.String()
}

func generateSparseConflictAllOfSpec(schemaCount, conflictEvery int) string {
	var builder strings.Builder
	builder.WriteString("openapi: 3.1.0\ncomponents:\n  schemas:\n")

	for i := 0; i < schemaCount; i++ {
		fmt.Fprintf(&builder, "    Base%d:\n", i)
		builder.WriteString("      type: object\n")
		builder.WriteString("      properties:\n")
		builder.WriteString("        kind:\n          type: string\n")
		fmt.Fprintf(&builder, "    Conflict%d:\n", i)
		builder.WriteString("      type: object\n")
		builder.WriteString("      allOf:\n")
		fmt.Fprintf(&builder, "        - $ref: '#/components/schemas/Base%d'\n", i)
		builder.WriteString("        - type: object\n")
		builder.WriteString("          properties:\n")
		if conflictEvery > 0 && i > 0 && i%conflictEvery == 0 {
			builder.WriteString("            kind:\n              type: number\n")
		} else {
			builder.WriteString("            kind:\n              type: string\n")
		}
	}

	return builder.String()
}
