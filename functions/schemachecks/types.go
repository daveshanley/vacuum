// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

const (
	// SanityCheckType checks type-specific constraints and scalar bounds.
	SanityCheckType = "type"
	// SanityCheckRequired checks required values against declared object properties.
	SanityCheckRequired = "required"
	// SanityCheckDependent checks dependentRequired values against declared object properties.
	SanityCheckDependent = "dependent"
	// SanityCheckEnumConst checks enum and const value compatibility.
	SanityCheckEnumConst = "enumConst"
	// SanityCheckPatterns checks ECMA-262 regular expression keywords.
	SanityCheckPatterns = "patterns"
	// SanityCheckComposition checks shallow composition contradictions.
	SanityCheckComposition = "composition"
	// SanityCheckQuality checks annotation and shape quality warnings.
	SanityCheckQuality = "quality"
	// SanityCheckExamples checks default and examples values against their schema.
	SanityCheckExamples = "examples"
)

// TypeCheckOptions configures schema type validation for a dialect or host format.
type TypeCheckOptions struct {
	AllowOAS30Nullable          bool
	ValidateDependentRequired   bool
	ValidateDiscriminator       bool
	ValidateEnumConstRedundancy bool
	ValidatePatterns            bool
	ValidateValueCompatibility  bool
}

// RequiredPropertyLookup captures whether a required property can be checked and whether it exists.
type RequiredPropertyLookup struct {
	PropertiesFound bool
	PropertyDefined bool
}

func (l RequiredPropertyLookup) merge(other RequiredPropertyLookup) RequiredPropertyLookup {
	return RequiredPropertyLookup{
		PropertiesFound: l.PropertiesFound || other.PropertiesFound,
		PropertyDefined: l.PropertyDefined || other.PropertyDefined,
	}
}
