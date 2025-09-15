package vacuum_report

import (
	"bytes"
	"compress/gzip"
	"github.com/daveshanley/vacuum/model"
	jsoniter "github.com/json-iterator/go"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"os"
	"testing"
	"time"
)

var jsonParse = jsoniter.ConfigCompatibleWithStandardLibrary

func BenchmarkCheckFileForVacuumReport_Compressed(b *testing.B) {
	j := testhelp_compressedJSON()
	for n := 0; n < b.N; n++ {
		rpt, _ := CheckFileForVacuumReport(j)
		if rpt == nil {
			continue
		}
	}
}

func BenchmarkCheckFileForVacuumReport_Uncompressed(b *testing.B) {
	j := testhelp_uncompressedJSON()
	for n := 0; n < b.N; n++ {
		rpt, _ := CheckFileForVacuumReport(j)
		if rpt == nil {
			continue
		}
	}
}

func TestBuildVacuumReport_Valid_Compressed(t *testing.T) {
	j := testhelp_compressedJSON()
	tmp, _ := os.CreateTemp("", "")
	defer os.Remove(tmp.Name())
	_, wErr := tmp.Write(j)
	assert.NoError(t, wErr)

	r, _, err := BuildVacuumReportFromFile(tmp.Name())
	assert.NotNil(t, r)
	assert.NoError(t, err)
}

func TestBuildVacuumReport_Invalid_Compressed(t *testing.T) {
	j := []byte("melody and pumpkin go on an adventure")
	tmp, _ := os.CreateTemp("", "")
	defer os.Remove(tmp.Name())
	_, wErr := tmp.Write(testhelp_compress(j))
	assert.NoError(t, wErr)

	r, _, err := BuildVacuumReportFromFile(tmp.Name())
	assert.Nil(t, r)
	assert.Error(t, err)
}

func TestBuildVacuumReport_Valid_Uncompressed(t *testing.T) {
	j := testhelp_uncompressedJSON()
	tmp, _ := os.CreateTemp("", "")
	defer os.Remove(tmp.Name())
	_, wErr := tmp.Write(j)
	assert.NoError(t, wErr)

	r, _, err := BuildVacuumReportFromFile(tmp.Name())
	assert.NotNil(t, r)
	assert.NoError(t, err)
}

func TestBuildVacuumReport_Invalid_Uncompressed(t *testing.T) {
	j := []byte("melody and pumpkin discover a secret castle in the shanley woods")
	tmp, _ := os.CreateTemp("", "")
	defer os.Remove(tmp.Name())
	_, wErr := tmp.Write(j)
	assert.NoError(t, wErr)

	r, _, err := BuildVacuumReportFromFile(tmp.Name())
	assert.Nil(t, r)
	assert.Error(t, err)
}

func TestBuildVacuumReport_Fail(t *testing.T) {
	r, _, err := BuildVacuumReportFromFile("I am not a real thing")
	assert.Nil(t, r)
	assert.Error(t, err)
}

func TestCheckFileForVacuumReport_CompressedJSON(t *testing.T) {
	// check for compressed JSON
	j := testhelp_compressedJSON()
	vr, err := CheckFileForVacuumReport(j)
	assert.NoError(t, err)
	assert.NotNil(t, vr)
	//assert.Len(t, *vr.SpecInfo.SpecBytes, 11730)
}

func TestCheckFileForVacuumReport_UncompressedJSON(t *testing.T) {
	// check for compressed JSON
	j := testhelp_uncompressedJSON()
	vr, err := CheckFileForVacuumReport(j)
	assert.NoError(t, err)
	assert.NotNil(t, vr)
	//assert.Len(t, *vr.SpecInfo.SpecBytes, 11730)
}

func TestCheckFileForVacuumReport_BadJSON_Uncompressed(t *testing.T) {
	// check for compressed JSON
	j := []byte("[{}{A{SOK)(*@()UEJH")
	vr, err := CheckFileForVacuumReport(j)
	assert.Error(t, err)
	assert.Nil(t, vr)
}

func TestCheckFileForVacuumReport_BadJSON_Compressed(t *testing.T) {
	// check for compressed JSON
	j := []byte("[{}{A{SOK)(*@()UEJH")
	vr, err := CheckFileForVacuumReport(testhelp_compress(j))
	assert.Error(t, err)
	assert.Nil(t, vr)
}

func testhelp_generateReport() *VacuumReport {

	vr := new(VacuumReport)
	si := new(datamodel.SpecInfo)

	bytes, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")
	si.SpecBytes = &bytes

	vr.Generated = time.Now()
	vr.SpecInfo = si

	r1 := model.RuleFunctionResult{Rule: &model.Rule{
		Id:           "one",
		Description:  "one",
		Severity:     model.SeverityError,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
	}, StartNode: &yaml.Node{Line: 1, Column: 10}, EndNode: &yaml.Node{Line: 20, Column: 20}}

	vr.ResultSet = model.NewRuleResultSet([]model.RuleFunctionResult{r1})
	vr.ResultSet.PrepareForSerialization(si)

	return vr
}

func testhelp_compressedJSON() []byte {
	vr := testhelp_generateReport()
	data, _ := jsonParse.Marshal(vr)
	return testhelp_compress(data)
}

func testhelp_uncompressedJSON() []byte {
	vr := testhelp_generateReport()
	data, _ := jsonParse.Marshal(vr)
	return data
}

func testhelp_compress(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		return nil
	}
	err = gz.Close()
	if err != nil {
		return nil
	}
	return b.Bytes()
}
