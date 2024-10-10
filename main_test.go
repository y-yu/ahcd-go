package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

type WriteMapInsteadOfCSV struct {
	Results map[string][]RecordValue
}

func (this *WriteMapInsteadOfCSV) Execute(name string, ch chan RecordValue) error {
	for value := range ch {
		this.Results[name] = append(this.Results[name], value)
	}

	return nil
}

func TestParseXML(t *testing.T) {
	var actual = WriteMapInsteadOfCSV{map[string][]RecordValue{}}
	var rootCmd = rootCmdConstructor(&actual)
	output := new(bytes.Buffer)

	rootCmd.SetOut(output)
	rootCmd.SetErr(output)
	rootCmd.SetArgs([]string{"test_data/export.xml"})

	err := rootCmd.Execute()

	assert.Nil(t, err)

	assert.Equal(t, 1, len(actual.Results["HeartRate.csv"]))
	assert.Equal(t, 2, len(actual.Results["BodyMassIndex.csv"]))
	assert.Equal(t, 7, len(actual.Results["BloodPressureSystolic.csv"]))
}
