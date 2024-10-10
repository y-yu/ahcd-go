package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type RecordValue struct {
	Unit          string `xml:"unit,attr"`
	Value         string `xml:"value,attr"`
	SourceName    string `xml:"sourceName,attr"`
	SourceVersion string `xml:"sourceVersion,attr"`
	Device        string `xml:"device,attr"`
	CreationDate  string `xml:"creationDate,attr"`
	StartDate     string `xml:"startDate,attr"`
	EndDate       string `xml:"endDate,attr"`
}

type Record struct {
	RecordValue
	Type string `xml:"type,attr"`
}

type WriteCSV interface {
	Execute(name string, ch chan RecordValue) error
}

var typeRe = regexp.MustCompile(`^HK.*TypeIdentifier(.+)$`)

func rootCmdConstructor(writeCSV WriteCSV) *cobra.Command {
	var rootCmd = &cobra.Command{
		Short: "Apple Health Care Data XML file converter to CSV.",
		Long:  "",
		Use:   "ahcd-go [-t <type>] [-d <dir>] <export.xml>",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := cmd.Flags().GetString("target")
			if err != nil {
				return err
			}
			directory, err := cmd.Flags().GetString("directory")
			if err != nil {
				return err
			}
			if len(args) < 1 {
				return fmt.Errorf("No XML file specified!\n")
			}

			fp, err := os.Open(args[0])
			if err != nil {
				return err
			}

			decoder := xml.NewDecoder(fp)

			var channels = make(map[string]chan RecordValue)
			for {
				token, tokenErr := decoder.Token()
				if tokenErr != nil {
					if tokenErr == io.EOF {
						fmt.Println("Done")
						break
					}
					return fmt.Errorf("decoding token: %v\n", err)
				}
				switch t := token.(type) {
				case xml.StartElement:
					if t.Name.Local == "Record" {
						var record Record
						if err := decoder.DecodeElement(&record, &t); err != nil {
							return fmt.Errorf("decoding element %q: %v\n", t.Name.Local, err)
						}
						if tys := typeRe.FindStringSubmatch(record.Type); len(tys) > 1 {
							ty := tys[1]
							if target == "" || (target != "" && target == ty) {
								if channel := channels[ty]; channel != nil {
									channel <- record.RecordValue
								} else {
									channels[ty] = make(chan RecordValue)
									go writeCSV.Execute(filepath.Join(directory, ty+".csv"), channels[ty])
									channels[ty] <- record.RecordValue
								}
							}
						}
					}
				}
			}
			err = fp.Close()
			if err != nil {
				return err
			}

			for _, channel := range channels {
				close(channel)
			}

			return nil
		},
	}
	rootCmd.Flags().StringP("target", "t", "", "Specifies a target you want to output to the CSV (e.g., -t BodyMass)")
	rootCmd.Flags().StringP("directory", "d", ".", "Specifies the directory to output to (e.g., -d /path/to)")

	return rootCmd
}

type WriteCSVImpl struct{}

func (_ WriteCSVImpl) Execute(name string, ch chan RecordValue) error {
	fp, err := os.Create(name)
	if err != nil {
		return err
	}
	defer fp.Close()

	writer := csv.NewWriter(fp)
	err = writer.Write(
		[]string{
			"Unit",
			"Value",
			"SourceName",
			"SourceVersion",
			"Device",
			"CreationDate",
			"StartDate",
			"EndDate",
		},
	)

	for value := range ch {
		err = writer.Write(
			[]string{
				value.Unit,
				value.Value,
				value.SourceName,
				value.SourceVersion,
				value.Device,
				value.StartDate,
				value.EndDate,
			},
		)
		if err != nil {
			return err
		}
		writer.Flush()
	}

	return nil
}

func main() {
	var rootCmd = rootCmdConstructor(WriteCSVImpl{})
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
