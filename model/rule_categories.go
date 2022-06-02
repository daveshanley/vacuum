// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package model

var RuleCategories = make(map[string]*RuleCategory)
var RuleCategoriesOrdered []*RuleCategory

func init() {
	RuleCategories[CategoryExamples] = &RuleCategory{
		Id:   CategoryExamples,
		Name: "Examples",
		Description: "Examples help consumers understand how API calls should look. They are really important for" +
			"automated tooling for mocking and testing. These rules check examples have been added to component schemas, " +
			"parameters and operations. These rules also check that examples match the schema and types provided.",
	}
	RuleCategories[CategoryOperations] = &RuleCategory{
		Id:   CategoryOperations,
		Name: "Operations",
		Description: "Operations are the core of the contract, they define paths and HTTP methods. These rules check" +
			" operations have been well constructed, looks for operationId, parameter, schema and return types in depth.",
	}
	RuleCategories[CategoryInfo] = &RuleCategory{
		Id:   CategoryInfo,
		Name: "Contract Information",
		Description: "The info object contains licencing, contact, authorship details and more. Checks to confirm " +
			"required details have been completed.",
	}
	RuleCategories[CategoryDescriptions] = &RuleCategory{
		Id:   CategoryDescriptions,
		Name: "Descriptions",
		Description: "Documentation is really important, in OpenAPI, just about everything can and should have a " +
			"description. This set of rules checks for absent descriptions, poor quality descriptions (copy/paste)," +
			" or short descriptions.",
	}
	RuleCategories[CategorySchemas] = &RuleCategory{
		Id:   CategorySchemas,
		Name: "Schemas",
		Description: "Schemas are how request bodies and response payloads are defined. They define the data going in " +
			"and the data flowing out of an operation. These rules check for structural validity, checking types, checking" +
			"required fields and validating correct use of structures.",
	}
	RuleCategories[CategorySecurity] = &RuleCategory{
		Id:   CategorySecurity,
		Name: "Security",
		Description: "Security plays a central role in RESTful APIs. These rules make sure that the correct definitions" +
			"have been used and put in the right places.",
	}
	RuleCategories[CategoryTags] = &RuleCategory{
		Id:   CategoryTags,
		Name: "Tags",
		Description: "Tags are used as meta-data for operations. They are mainly used by tooling as a taxonomy mechanism" +
			" to build navigation, search and more. Tags are important as they help consumers navigate the contract when " +
			"using documentation, testing, code generation or analysis tools.",
	}
	RuleCategories[CategoryValidation] = &RuleCategory{
		Id:   CategoryValidation,
		Name: "Validation",
		Description: "Validation rules make sure that certain characters or patterns have not been used that may cause" +
			"issues when rendering in different types of applications.",
	}
	RuleCategories[CategoryAll] = &RuleCategory{
		Id:          CategoryAll,
		Name:        "All Categories",
		Description: "All the categories, for those who like a party.",
	}

	RuleCategoriesOrdered = append(RuleCategoriesOrdered,
		RuleCategories[CategoryInfo],
		RuleCategories[CategoryOperations],
		RuleCategories[CategoryTags],
		RuleCategories[CategorySchemas],
		RuleCategories[CategoryValidation],
		RuleCategories[CategoryDescriptions],
		RuleCategories[CategorySecurity],
		RuleCategories[CategoryExamples],
	)
}
