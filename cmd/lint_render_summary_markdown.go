// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/dustin/go-humanize"
	"github.com/pterm/pterm"
)

func RenderMarkdownSummary(rso RenderSummaryOptions) {

	rs := rso.RuleResultSet
	cats := rso.RuleCategories
	totalFiles := rso.TotalFiles
	filename := rso.Filename
	silent := rso.Silent
	sev := rso.Severity

	// headers: a slice of column names
	headers := []string{"Category", "Errors", "Warnings", "Info"}

	// rows: each inner slice is one row of table data
	var rows [][]string
	for _, cat := range cats {
		errors := rs.GetErrorsByRuleCategory(cat.Id)
		warn := rs.GetWarningsByRuleCategory(cat.Id)
		info := rs.GetInfoByRuleCategory(cat.Id)

		if len(errors) > 0 || len(warn) > 0 || len(info) > 0 {
			rows = append(rows, []string{
				cat.Name,
				fmt.Sprintf("%v", humanize.Comma(int64(len(errors)))), // e.g. "1,234"
				fmt.Sprintf("%v", humanize.Comma(int64(len(warn)))),   // e.g. "56"
				fmt.Sprintf("%v", humanize.Comma(int64(len(info)))),   // e.g. "7"
			})
		}
	}

	errs := rs.GetErrorCount()
	warnings := rs.GetWarnCount()
	informs := rs.GetInfoCount()
	errorsHuman := humanize.Comma(int64(rs.GetErrorCount()))
	warningsHuman := humanize.Comma(int64(rs.GetWarnCount()))
	informsHuman := humanize.Comma(int64(rs.GetInfoCount()))
	ruleset := rso.RuleSet

	buf := strings.Builder{}

	if rso.PipelineOutput {

		errIcon := "🔴"

		violationHeaders := []string{"location", "JSONPath"}

		buf.WriteString("## [vacuum](https://quobix.com/vacuum/) OpenAPI quality report")
		buf.WriteString(fmt.Sprintf("\n> vacuum has graded this OpenAPI specification with a score of `%d` out of a possible 100\n\n", rso.ReportStats.OverallScore))

		if errs > 0 {
			buf.WriteString(fmt.Sprintf("### %s `%d` errors detected 🚨\n\n", errIcon, errs))
			buf.WriteString(fmt.Sprint("> vacuum detected **errors** in your OpenAPI specification, please review and address accordingly.\n\n"))

			if warnings > 0 {
				buf.WriteString(fmt.Sprintf("⚠️`%d` warnings were also detected\n\n", warnings))
			}
		}

		if warnings > 0 && errs == 0 {
			buf.WriteString(fmt.Sprintf("#### ⚠️`%d` warnings detected\n\n", warnings))
			buf.WriteString(fmt.Sprint("> vacuum detected warnings in your OpenAPI specification, please review and address accordingly.\n\n"))
		}
		if informs > 0 {
			buf.WriteString(fmt.Sprintf("ℹ️`%d` informs found\n\n", informs))
		}

		if rso.RenderRules {

			// sort the ruleset by severity
			type ruleSevMap struct {
				rule *model.Rule
				sev  int // 0 = info, 1 = warn, 2 = error
			}
			var rules []ruleSevMap
			if ruleset != nil && len(ruleset.Rules) > 0 {
				for _, r := range ruleset.Rules {
					s := 0
					switch r.Severity {
					case model.SeverityWarn:
						s = 1
					case model.SeverityError:
						s = 2
					}
					rules = append(rules, ruleSevMap{
						rule: r,
						sev:  s,
					})
				}
			}

			sort.Slice(rules, func(i, j int) bool {
				if rules[i].sev == rules[j].sev {
					return rules[i].rule.Id > rules[j].rule.Id
				}
				return rules[i].sev > rules[j].sev
			})

			buf.WriteString(fmt.Sprintf("<details><summary>vacuum ran against the following %d rules:</summary>\n\n", len(rules)))
			for _, r := range rules {
				sevIcon := "ℹ️"
				switch r.rule.Severity {
				case model.SeverityError:
					sevIcon = errIcon
				case model.SeverityWarn:
					sevIcon = "⚠️"
				}
				n := strings.ReplaceAll(r.rule.Name, "<", "&lt;")
				n = strings.ReplaceAll(n, ">", "&gt;")
				buf.WriteString(fmt.Sprintf("- %s `%s` (_%s_)\n", sevIcon, r.rule.Id, n))
			}
			buf.WriteString(fmt.Sprint("</details>\n\n"))
		}

		summaryTableMarkdown := utils.RenderMarkdownTable(headers, rows)
		buf.WriteString(fmt.Sprint("---\n\n"))

		for _, cat := range cats {
			catErrs := rs.GetErrorsByRuleCategory(cat.Id)
			warn := rs.GetWarningsByRuleCategory(cat.Id)
			info := rs.GetInfoByRuleCategory(cat.Id)

			errorRuleMap := make(map[string]int)
			warnRuleMap := make(map[string]int)
			infoRuleMap := make(map[string]int)

			checkMap := func(ruleId string, ruleMap map[string]int) {
				if _, ok := ruleMap[ruleId]; !ok {
					ruleMap[ruleId] = 1
				} else {
					ruleMap[ruleId]++
				}
			}

			for _, e := range catErrs {
				checkMap(e.Rule.Id, errorRuleMap)
			}
			for _, e := range warn {
				checkMap(e.Rule.Id, warnRuleMap)
			}
			for _, e := range info {
				checkMap(e.Rule.Id, infoRuleMap)
			}

			if len(catErrs) == 0 && len(warn) == 0 && len(info) == 0 {
				continue // no violations for this category
			}

			buf.WriteString(fmt.Sprintf("### `%s` violations\n", cat.Name))
			if len(catErrs) > 0 {
				buf.WriteString(fmt.Sprintf("<details><summary>%s Errors: %s</summary>\n", errIcon, humanize.Comma(int64(len(catErrs)))))
				for ruleId, count := range errorRuleMap {
					if count > 0 {
						buf.WriteString(fmt.Sprintf("%s %s : %d\n\n", errIcon, ruleId, count))
						var errData [][]string
						for _, v := range catErrs {
							if v.Rule.Id == ruleId {
								errData = append(errData, []string{fmt.Sprintf("`%d:%d`", v.StartNode.Line, v.StartNode.Column), v.Path})
							}
						}
						buf.WriteString(fmt.Sprintln(utils.RenderMarkdownTable(violationHeaders, errData)))
					}
				}
				buf.WriteString(fmt.Sprint("</details>\n\n"))
			}
			if len(warn) > 0 {
				buf.WriteString(fmt.Sprintf("<details><summary>⚠️️ Warnings: %s</summary>\n", humanize.Comma(int64(len(warn)))))
				for ruleId, count := range warnRuleMap {
					if count > 0 {
						buf.WriteString(fmt.Sprintf("⚠️️ %s: %d\n\n", ruleId, count))
						var warnData [][]string
						for _, v := range warn {
							if v.Rule.Id == ruleId {
								warnData = append(warnData, []string{fmt.Sprintf("`%d:%d`", v.StartNode.Line, v.StartNode.Column), v.Path})
							}
						}
						buf.WriteString(fmt.Sprintln(utils.RenderMarkdownTable(violationHeaders, warnData)))
					}
				}
				buf.WriteString(fmt.Sprint("</details>\n\n"))
			}
			if len(info) > 0 {
				buf.WriteString(fmt.Sprintf("<details><summary>ℹ️️ Informs: %s</summary>\n\n", humanize.Comma(int64(len(info)))))
				for ruleId, count := range infoRuleMap {
					if count > 0 {
						buf.WriteString(fmt.Sprintf("ℹ️️ %s: %d\n", ruleId, count))
						var infoData [][]string
						for _, v := range info {
							if v.Rule.Id == ruleId {
								infoData = append(infoData, []string{fmt.Sprintf("`%d:%d`", v.StartNode.Line, v.StartNode.Column), v.Path})
							}
						}
						buf.WriteString(fmt.Sprintln(utils.RenderMarkdownTable(violationHeaders, infoData)))
					}
				}
				buf.WriteString(fmt.Sprint("</details>\n\n"))
			}
			buf.WriteString(fmt.Sprint("---\n\n"))
		}
		total := rso.ReportStats.TotalErrors + rso.ReportStats.TotalWarnings + rso.ReportStats.TotalInfo

		if total > 0 {
			buf.WriteString(fmt.Sprintln(summaryTableMarkdown))
		} else {
			buf.WriteString(fmt.Sprint("✅ You have a perfect score! **Congratulations, you're doing it right.**\n\n"))
		}

		buf.WriteString(fmt.Sprintf("> learn more about vacuum at [quobix.com/vacuum](https://quobix/vacuum/)\n"))
		fmt.Print(buf.String())
		return
	}

	tableData := pterm.TableData{headers}
	for _, row := range rows {
		tableData = append(tableData, row)
	}

	if len(rs.Results) > 0 {
		if !silent {
			err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
			if err != nil {
				pterm.Error.Printf("error rendering table '%v'", err.Error())
			}
		}
	}

	errorHeader := pterm.HeaderPrinter{
		TextStyle:       pterm.NewStyle(pterm.FgWhite),
		BackgroundStyle: pterm.NewStyle(pterm.BgRed),
		Margin:          10,
	}

	successHeader := pterm.HeaderPrinter{
		TextStyle:       pterm.NewStyle(pterm.FgBlack),
		BackgroundStyle: pterm.NewStyle(pterm.BgGreen),
		Margin:          10,
	}

	warningHeader := pterm.HeaderPrinter{
		TextStyle:       pterm.NewStyle(pterm.FgBlack),
		BackgroundStyle: pterm.NewStyle(pterm.BgYellow),
		Margin:          10,
	}

	if totalFiles <= 1 {

		if errs > 0 {
			errorHeader.Printf(
				"Linting file '%s' failed with %v errors, %v warnings and %v informs", filename, errorsHuman, warningsHuman, informsHuman)
			return
		}
		if warnings > 0 {
			msg := "passed, but with"
			switch sev {
			case model.SeverityWarn:
				msg = "failed with"
			}

			warningHeader.Printf(
				"Linting %s %v warnings and %v informs", msg, warningsHuman, informsHuman)
			return
		}

		if informs > 0 {
			successHeader.Printf(
				"Linting passed, %v informs reported", informsHuman)
			return
		}

		if silent {
			return
		}

		successHeader.Println(
			"Linting passed, A perfect score! well done!")

	} else {

		if errs > 0 {
			pterm.Error.Printf("'%s' failed with %v errors, %v warnings and %v informs\n\n",
				filename, errorsHuman, warningsHuman, informsHuman)
			pterm.Println()
			return
		}
		if warnings > 0 {
			pterm.Warning.Printf(
				"'%s' passed, but with %v warnings and %v informs\n\n", filename, warningsHuman, informsHuman)
			pterm.Println()
			return
		}

		if informs > 0 {

			successHeader.Printf(
				"'%s' passed, %v informs reported\n\n", filename, informsHuman)

			pterm.Println()
			return
		}

		successHeader.Printf(
			"'%s' passed, A perfect score! well done!\n\n", filename)
		pterm.Println()

	}

}
