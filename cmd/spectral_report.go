package cmd

import (
	"encoding/json"
	"errors"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io/ioutil"
)

func GetSpectralReportCommand() *cobra.Command {

	return &cobra.Command{
		Use:   "report",
		Short: "Generate a Spectral compatible JSON report",
		Long: "Generate a JSON report using the same model as Spectral. Default output " +
			"filename is 'vacuum-spectral-report.json' located in the working directory.",
		Example: "vacuum report my-awesome-spec.yaml <vacuum-spectral-report.json>",
		RunE: func(cmd *cobra.Command, args []string) error {

			// check for file args
			if len(args) == 0 {
				errText := "please supply an OpenAPI specification to generate a report"
				pterm.Error.Println(errText)
				pterm.Println()
				return errors.New(errText)
			}

			//timeFlag, _ := cmd.Flags().GetBool("time")

			reportOutput := "vacuum-spectral-report.json"

			if len(args) > 1 {
				reportOutput = args[1]
			}

			//start := time.Now()

			// read file.
			b, ferr := ioutil.ReadFile(args[0])

			if ferr != nil {
				pterm.Error.Printf("Unable to read file '%s': %s\n", args[0], ferr.Error())
				pterm.Println()
				return ferr
			}

			pterm.Info.Printf("Running vacuum against spec '%s'\n", args[0])
			pterm.Println()

			rs := rulesets.BuildDefaultRuleSets()
			results, _ := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

			//duration := time.Since(start)

			resultSet := model.NewRuleResultSet(results)
			resultSet.SortResultsByLineNumber()

			//fi, _ := os.Stat(args[0])

			// serialize
			spectralReport := resultSet.GenerateSpectralReport(args[0]) // todo: convert to full path.

			data, err := json.MarshalIndent(spectralReport, "", "    ")

			if err != nil {
				pterm.Error.Printf("Unable to read marshal report into JSON '%s': %s\n", args[0], ferr.Error())
				pterm.Println()
				return err
			}

			err = ioutil.WriteFile(reportOutput, data, 0664)

			if err != nil {
				pterm.Error.Printf("Unable to write report file: '%s': %s\n", reportOutput, ferr.Error())
				pterm.Println()
				return err
			}

			pterm.Info.Printf("Report generated for '%s', written to '%s'\n", args[0], reportOutput)

			//renderTime(timeFlag, duration, fi)

			return nil
		},
	}

}
