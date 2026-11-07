// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestReviewedOperationIDsReserveNamesAcrossProposals(t *testing.T) {
	for _, tc := range []struct{ rule, source, first, second string }{
		{"operation-operationId-valid-in-url", `{"paths":{"/a":{"get":{"operationId":"list.pets"}},"/b":{"get":{"operationId":"list:pets"}},"/existing":{"get":{"operationId":"list_pets_2"}}},"components":{"links":{"next":{"operationId":"list:pets"}}}}`, "list_pets", "list_pets_3"},
		{"operation-operationId-unique", `{"paths":{"/a.b":{"get":{"operationId":"same"}},"/a_b":{"get":{"operationId":"same"}}}}`, "get__a_b", "get__a_b_2"},
	} {
		t.Run(tc.rule, func(t *testing.T) {
			firstPath, secondPath := "/a", "/b"
			if tc.rule == "operation-operationId-unique" {
				firstPath, secondPath = "/a.b", "/a_b"
			}
			findings := []Finding{{RuleID: tc.rule, Pointer: "/paths/" + EscapeToken(firstPath) + "/get"}, {RuleID: tc.rule, Pointer: "/paths/" + EscapeToken(secondPath) + "/get"}}
			source := []byte(tc.source)
			plan, err := ReviewedPlan("source", source, findings)
			require.NoError(t, err)
			require.Len(t, plan.Targets, 2)
			reversed, err := ReviewedPlan("source", source, []Finding{findings[1], findings[0]})
			require.NoError(t, err)
			require.Equal(t, plan.Operations, reversed.Operations)
			got, err := Apply("source", source, plan)
			require.NoError(t, err)
			var document struct {
				Paths map[string]struct {
					Get struct {
						ID string `json:"operationId"`
					} `json:"get"`
				} `json:"paths"`
				Components struct {
					Links map[string]struct {
						ID string `json:"operationId"`
					} `json:"links"`
				} `json:"components"`
			}
			require.NoError(t, json.Unmarshal(got, &document))
			require.Equal(t, tc.first, document.Paths[firstPath].Get.ID)
			require.Equal(t, tc.second, document.Paths[secondPath].Get.ID)
			if tc.rule == "operation-operationId-valid-in-url" {
				require.Equal(t, tc.second, document.Components.Links["next"].ID)
			}
		})
	}
}

func TestReviewedPathRenameKeepsMappingOrderAndAcronyms(t *testing.T) {
	for _, tc := range []struct{ before, after string }{
		{"/userID", "/user-id"}, {"/HTTPServer", "/http-server"}, {"/USER_NAMES", "/user-names"}, {"/UserIDs.JSON", "/user-ids.json"},
	} {
		source := []byte("paths:\n  " + tc.before + ":\n    get:\n      description: |\n        Keep this block.\n  /zebras: {}\n")
		plan, err := ReviewedPlan("source", source, []Finding{{RuleID: "paths-kebab-case", Pointer: "/paths/" + EscapeToken(tc.before)}})
		require.NoError(t, err)
		got, err := Apply("source", source, plan)
		require.NoError(t, err)
		require.Equal(t, strings.Replace(string(source), tc.before, tc.after, 1), string(got))
	}
}

func TestReviewedPropertyRenameKeepsOrderAndRelatedNames(t *testing.T) {
	for _, tc := range []struct{ before, after string }{
		{"USER_NAME", "userName"}, {"HTTP_SERVER", "httpServer"}, {"first_name", "firstName"}, {"URLValue", "urlValue"},
	} {
		source := []byte("type: object\nproperties:\n  " + tc.before + ":\n    type: string\n    description: 'Keep quotes'\n  zebra:\n    type: string\nrequired: [" + tc.before + "]\nexample:\n  " + tc.before + ": person\n  zebra: animal\n")
		plan, err := ReviewedPlan("source", source, []Finding{{RuleID: "camel-case-properties", Pointer: "/properties/" + EscapeToken(tc.before)}})
		require.NoError(t, err)
		got, err := Apply("source", source, plan)
		require.NoError(t, err)
		require.Equal(t, strings.ReplaceAll(string(source), tc.before, tc.after), string(got))
	}
}
