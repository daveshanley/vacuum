package motor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	asyncfunc "github.com/daveshanley/vacuum/functions/asyncapi"
	openapifunc "github.com/daveshanley/vacuum/functions/openapi"
	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/go-yaml"
	schemaengine "github.com/pb33f/jsonschema/v6"
	arazzovalidation "github.com/pb33f/libopenapi-validator/arazzo"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	"github.com/pb33f/libopenapi/index"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type repairDefect struct{ kind, pointer, detail string }

// verifyRepairIntegrity compares native validation defects independently of the
// caller's lint rules and ignores. Existing defects can remain; new ones cannot.
func verifyRepairIntegrity(before, after *RuleSetExecutionResult, correspondence *repair.Correspondence, options *ExecutionOptions) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("repair integrity check failed: %v", p)
		}
	}()
	if err = repairCanceled(options); err != nil {
		return err
	}
	old, err := baselineRepairIntegrity(before, options)
	if err != nil {
		return fmt.Errorf("baseline integrity check: %w", err)
	}
	next, err := repairIntegrityDefects(after, options)
	if err != nil {
		return fmt.Errorf("candidate integrity check: %w", err)
	}
	counts := make(map[repairDefect]int, len(old))
	for _, d := range old {
		counts[d]++
	}
	for _, d := range next {
		original, ok := correspondence.OriginalPointer(d.pointer)
		if !ok {
			return fmt.Errorf("repair introduces %s at %s: %s", d.kind, d.pointer, d.detail)
		}
		d.pointer = original
		d.detail = originalDiagnosticLines(d.detail, correspondence)
		if counts[d] == 0 {
			return fmt.Errorf("repair introduces %s at %s: %s", d.kind, d.pointer, d.detail)
		}
		counts[d]--
	}
	return repairCanceled(options)
}

func repairIntegrityDefects(result *RuleSetExecutionResult, options ...*ExecutionOptions) ([]repairDefect, error) {
	if result == nil || result.RuleSetExecution == nil {
		return nil, fmt.Errorf("document model is unavailable")
	}
	e := result.RuleSetExecution
	document := result.ownedDocument
	if document == nil {
		document = e.Document
	}
	root := e.CanonicalDocument
	if root == nil {
		return nil, fmt.Errorf("canonical document is unavailable")
	}
	var defects []repairDefect
	// This also protects plugin candidates, which need not pass repair.Apply.
	seen := make(map[*yaml.Node]bool)
	var walk func(*yaml.Node, string) error
	walk = func(n *yaml.Node, p string) error {
		if n == nil {
			return fmt.Errorf("nil document node")
		}
		if seen[n] {
			return fmt.Errorf("aliased or cyclic document tree")
		}
		seen[n] = true
		if n.Kind == yaml.AliasNode {
			return fmt.Errorf("aliased document tree")
		}
		switch n.Kind {
		case yaml.DocumentNode:
			for _, c := range n.Content {
				if err := walk(c, p); err != nil {
					return err
				}
			}
		case yaml.MappingNode:
			if len(n.Content)%2 != 0 {
				return fmt.Errorf("invalid mapping")
			}
			keys := make(map[string]bool)
			for i := 0; i < len(n.Content); i += 2 {
				k := n.Content[i]
				path := p + "/" + repair.EscapeToken(k.Value)
				if keys[k.Value] {
					defects = append(defects, repairDefect{"duplicate key", path, k.Value})
				}
				keys[k.Value] = true
				if err := walk(n.Content[i+1], path); err != nil {
					return err
				}
			}
		case yaml.SequenceNode:
			for i, c := range n.Content {
				if err := walk(c, p+"/"+strconv.Itoa(i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, err
	}
	locations := repair.IndexLocations(root)
	add := func(kind string, node *yaml.Node, path, detail string) {
		if p, ok := locations.FindExact(node); ok {
			path = p
		}
		// Native opaque errors without source nodes remain document-wide defects.
		if path != "" && !strings.HasPrefix(path, "/") {
			path = ""
		}
		defects = append(defects, repairDefect{kind, path, detail})
	}
	// Semantic checks use the unresolved owner. Only exact authored nodes can
	// supply root pointers; external nodes must not match by line and column.
	addSemantic := func(kind string, node *yaml.Node, path, detail string) {
		if pointer, ok := locations.FindExact(node); ok {
			defects = append(defects, repairDefect{kind, pointer, detail})
			return
		}
		defects = append(defects, repairDefect{kind, "", path + ": " + detail})
	}
	switch {
	case e.Arazzo != nil:
		validation := e.Arazzo.Validation
		if validation == nil {
			ctx := context.Background()
			if len(options) > 0 && options[0] != nil && options[0].Context != nil {
				ctx = options[0].Context
			}
			timeout := e.Timeout
			if timeout <= 0 {
				timeout = 5 * time.Second
			}
			ctx, cancel := context.WithTimeout(ctx, timeout)
			var err error
			validation, err = arazzovalidation.Validate(ctx, arazzovalidation.Document{Root: root, URI: e.SpecFileName})
			cancel()
			if err != nil {
				return nil, err
			}
			if validation == nil {
				return nil, fmt.Errorf("Arazzo validation is unavailable")
			}
		}
		for _, d := range validation.Diagnostics {
			add("Arazzo "+string(d.Code), nil, d.Pointer, d.Message)
		}
	case e.AsyncAPI != nil:
		for _, err := range e.AsyncAPI.DocumentErrors() {
			if err != nil {
				add("AsyncAPI document", nil, "", err.Error())
			}
		}
		invoke := func(fn model.RuleFunction, paths ...string) {
			var nodes []*yaml.Node
			for _, path := range paths {
				owner := fixNodeAt(root, path)
				if owner != nil && owner.Kind == yaml.MappingNode {
					for i := 1; i < len(owner.Content); i += 2 {
						nodes = append(nodes, owner.Content[i])
					}
				}
			}
			ctx := model.RuleFunctionContext{AsyncAPI: e.AsyncAPI, Index: result.Index, Rule: &model.Rule{Id: fn.GetSchema().Name}}
			for _, d := range fn.RunRule(nodes, ctx) {
				add("AsyncAPI "+fn.GetSchema().Name, d.StartNode, "", d.Message)
			}
		}
		invoke(asyncfunc.OperationChannel{}, "/operations", "/components/operations")
		invoke(asyncfunc.OperationMessages{}, "/operations", "/components/operations")
		invoke(asyncfunc.OperationReply{}, "/operations", "/components/operations")
		invoke(asyncfunc.ChannelServers{}, "/channels", "/components/channels")
		invoke(asyncfunc.ChannelParameters{}, "/channels", "/components/channels")
		invoke(asyncfunc.ServerVariables{}, "/servers", "/components/servers")
	case isJSONSchemaFormat(e.SpecFormat):
		if !schemautil.IsSupportedDialect(schemautil.DetectDialect(root).Format) {
			return nil, fmt.Errorf("schema dialect has no integrity validator")
		}
		issues, err := schemautil.ValidateAgainstMetaschema(root)
		if err != nil {
			return nil, err
		}
		for _, d := range issues {
			add("schema", d.Node, strings.TrimPrefix(d.Pointer, "#"), d.Message)
		}
	case document != nil:
		if e.IndexUnresolved == nil {
			return nil, fmt.Errorf("OpenAPI integrity index is unavailable")
		}
		ctx := model.RuleFunctionContext{Index: e.IndexUnresolved, DrDocument: e.DrDocument}
		checks := []model.RuleFunction{openapifunc.OAS2OperationSecurityDefined{}}
		if document.GetSpecInfo().VersionNumeric >= 3 {
			if e.DrDocument == nil || e.DrDocument.V3Document == nil {
				return nil, fmt.Errorf("OpenAPI integrity Doctor model is unavailable; reference and path checks cannot complete")
			}
			checks = []model.RuleFunction{openapifunc.PathParameters{}, openapifunc.OperationSecurityDefined{}}
		}
		for _, check := range checks {
			ctx.Rule = &model.Rule{Id: check.GetSchema().Name}
			for _, d := range check.RunRule([]*yaml.Node{root}, ctx) {
				addSemantic("OpenAPI "+check.GetSchema().Name, d.StartNode, d.Path, d.Message)
			}
		}
		// The native index owns duplicate parameter detection. Read one graph,
		// rather than count the resolved and unresolved copies of each error.
		indexes := []*index.SpecIndex{e.IndexUnresolved}
		if rolo := e.IndexUnresolved.GetRolodex(); rolo != nil {
			indexes = append(indexes, rolo.GetIndexes()...)
		}
		for _, idx := range indexes {
			if idx == nil {
				continue
			}
			for _, err := range idx.GetOperationParametersIndexErrors() {
				var ie *index.IndexingError
				if errors.As(err, &ie) {
					addSemantic("operation parameter", ie.Node, idx.GetSpecAbsolutePath()+"#"+ie.Path, ie.Error())
				} else {
					addSemantic("operation parameter", nil, idx.GetSpecAbsolutePath(), err.Error())
				}
			}
		}
		defects = append(defects, repairLinkDefects(document, root)...)
		valid, validation := schema_validation.ValidateOpenAPIDocument(document)
		if !valid && len(validation) == 0 {
			return nil, fmt.Errorf("OpenAPI schema validation did not complete")
		}
		printer := message.NewPrinter(language.Tag{})
		var leaves func(*schemaengine.ValidationError)
		leaves = func(v *schemaengine.ValidationError) {
			if v == nil {
				return
			}
			if len(v.Causes) > 0 {
				for _, c := range v.Causes {
					leaves(c)
				}
				return
			}
			path := ""
			for _, part := range v.InstanceLocation {
				path += "/" + repair.EscapeToken(part)
			}
			detail := v.SchemaURL
			if v.ErrorKind != nil {
				detail += " " + v.ErrorKind.LocalizedString(printer)
			}
			add("schema", nil, path, detail)
		}
		for _, v := range validation {
			if len(v.SchemaValidationErrors) == 0 {
				add("schema", nil, "", v.Message+": "+v.Reason)
			}
			for _, f := range v.SchemaValidationErrors {
				if f.OriginalJsonSchemaError != nil {
					leaves(f.OriginalJsonSchemaError)
					continue
				}
				path := ""
				for _, part := range f.InstancePath {
					path += "/" + repair.EscapeToken(part)
				}
				add("schema", nil, path, f.KeywordLocation+" "+f.Reason)
			}
		}
		// OpenAPI 3.1 document metaschemas leave Schema Objects extensible.
		// Validate those objects against JSON Schema as a separate native check.
		if result.Index != nil && document.GetSpecInfo().VersionNumeric >= 3.1 {
			validated := make(map[*yaml.Node]bool)
			refs := result.Index.GetAllSchemas()
			sort.SliceStable(refs, func(i, j int) bool {
				a, _ := locations.Find(refs[i].Node)
				b, _ := locations.Find(refs[j].Node)
				return len(a) < len(b)
			})
			var mark func(*yaml.Node)
			mark = func(n *yaml.Node) {
				if n == nil || validated[n] {
					return
				}
				validated[n] = true
				for _, child := range n.Content {
					mark(child)
				}
			}
			for _, ref := range refs {
				if ref == nil || ref.Node == nil || validated[ref.Node] {
					continue
				}
				mark(ref.Node)
				if !schemautil.IsSupportedDialect(schemautil.DetectDialect(ref.Node).Format) {
					return nil, fmt.Errorf("embedded schema dialect has no integrity validator")
				}
				issues, err := schemautil.ValidateAgainstMetaschema(ref.Node)
				if err != nil {
					return nil, err
				}
				for _, issue := range issues {
					add("schema", issue.Node, "", issue.Message)
				}
			}
		}
		if result.Index != nil && document.GetSpecInfo().VersionNumeric < 3 {
			ctx := model.RuleFunctionContext{Index: result.Index, Rule: &model.Rule{Id: "operation-operationId-unique"}}
			for _, d := range (openapifunc.UniqueOperationId{}).RunRule([]*yaml.Node{root}, ctx) {
				add("duplicate operationId", d.StartNode, "", d.Message)
			}
		}
	default:
		return nil, fmt.Errorf("document format has no integrity validator")
	}
	indexes := []*index.SpecIndex{result.Index, e.IndexResolved, e.IndexUnresolved}
	visited := make(map[*index.SpecIndex]bool)
	for i := 0; i < len(indexes); i++ {
		idx := indexes[i]
		if idx == nil || visited[idx] {
			continue
		}
		visited[idx] = true
		if r := idx.GetRolodex(); r != nil {
			indexes = append(indexes, r.GetIndexes()...)
		}
		for _, er := range idx.GetReferenceIndexErrors() {
			var ie *index.IndexingError
			if isJSONSchemaFormat(e.SpecFormat) && errors.As(er, &ie) && schemautil.IsDynamicScopeIndexingError(ie) {
				continue
			}
			if errors.As(er, &ie) {
				add("reference", ie.Node, "", ie.Error())
			} else {
				add("reference", nil, "", er.Error())
			}
		}
		if resolver := idx.GetResolver(); resolver != nil {
			for _, er := range resolver.GetResolvingErrors() {
				if isJSONSchemaFormat(e.SpecFormat) && schemautil.IsDynamicScopeResolvingError(er) {
					continue
				}
				add("reference", er.Node, "", er.ErrorRef.Error())
			}
		}
	}
	return defects, nil
}
