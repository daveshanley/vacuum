package model

type RuleFunctionContext struct {
}

type RuleFunctionResult struct {
	Message string
	Path    string
}

type RuleFunction interface {
	RunRule(input string, options map[string]interface{}, context RuleFunctionContext)
}

type RuleAction struct {
	Field           string `json:"field"`
	FunctionName    string `json:"description"`
	FunctionOptions map[string]interface{}
}

type Rule struct {
	Description string     `json:"description"`
	Given       []string   `json:"given"`
	Formats     []string   `json:"formats"`
	Resolved    bool       `json:"resolved"`
	Recommended bool       `json:"recommended"`
	Severity    int        `json:"severity"`
	Then        RuleAction `json:"then"`
}

type RuleSet struct {
	DocumentationURI string   `json:"documentationUri"`
	Formats          []string `json:"formats"`
}
