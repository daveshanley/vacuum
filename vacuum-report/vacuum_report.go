package vacuum_report

import (
	"bytes"
	"compress/gzip"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/rulesets"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"sync"
	"time"
)

// VacuumReport is a serialized, ready to re-replay linting report. It can be used on its own, or it
// can be used as a replay model to re-render the report again. Time is now available to vacuum.
type VacuumReport struct {
	Generated  time.Time                 `json:"generated" yaml:"generated"`
	SpecInfo   *model.SpecInfo           `json:"specInfo" yaml:"specInfo"`
	Statistics *reports.ReportStatistics `json:"statistics" yaml:"statistics"`
	ResultSet  *model.RuleResultSet      `json:"resultSet" yaml:"resultSet"`
}

// BuildVacuumReportFromFile will attempt (at great speed) to read in a file as a Vacuum Report. If successful a pointer
// to a ready to run report is returned. If the file isn't a report, or can't be read and cannot be parsed then nil is returned.
// regardless of the outcome, if the file can be read, the bytes will be returned.
func BuildVacuumReportFromFile(filePath string) (*VacuumReport, []byte, error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	vr, err := CheckFileForVacuumReport(bytes)
	if err != nil {
		return nil, bytes, err
	}

	// ok so far, so good. next we have to convert each range into a *yaml.Node again. This is so the rest of the
	// application has no idea that we're replaying and will perform normally. We want to go as fast as possible here,
	// so for each result, run each re-build in a new thread.
	var wg sync.WaitGroup
	de := rulesets.BuildDefaultRuleSets()
	rs := de.GenerateOpenAPIDefaultRuleSet()
	var rebuildNode = func(res *model.RuleFunctionResult, wg *sync.WaitGroup, rs *rulesets.RuleSet) {
		r := res.Range
		res.StartNode = new(yaml.Node)
		res.EndNode = new(yaml.Node)
		res.StartNode.Line = r.Start.Line
		res.StartNode.Column = r.Start.Char
		res.EndNode.Line = r.End.Line
		res.EndNode.Column = r.End.Char
		res.Rule = rs.Rules[res.RuleId]
		wg.Done()
	}

	wg.Add(len(vr.ResultSet.Results))
	for _, res := range vr.ResultSet.Results {
		// go fast!
		go rebuildNode(res, &wg, rs)
	}
	wg.Wait()
	return vr, bytes, nil
}

// CheckFileForVacuumReport will try to extract a vacuum report from a byte array. It checks if the
// file is compressed or not, then if it can be marshalled into a report.
func CheckFileForVacuumReport(data []byte) (*VacuumReport, error) {
	var jsonParse = jsoniter.ConfigCompatibleWithStandardLibrary
	r := bytes.NewReader(data)
	gzipRead, rerr := gzip.NewReader(r)
	var vr VacuumReport

	if rerr != nil {
		// not compressed? try unmarshal it.
		if jerr := jsonParse.Unmarshal(data, &vr); jerr != nil {
			return nil, jerr
		}

	} else {
		// ok so the file is gzipped, however, it may still not be a report.
		// run through all the checks as we would normally.
		decompressed, derr := ioutil.ReadAll(gzipRead)
		if derr != nil {
			return nil, derr
		}
		if jerr := jsonParse.Unmarshal(decompressed, &vr); jerr != nil {
			return nil, jerr
		}

	}
	return &vr, nil
}
