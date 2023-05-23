// Copyright © 2019 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dyff

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gonvenience/neat"
	yamlv3 "gopkg.in/yaml.v3"
)

type JSONDiffDetail struct {
	Kind     string `json:"kind"`
	Addition string `json:"addition,omitempty"`
	Removal  string `json:"removal,omitempty"`
}

type JSONDiff struct {
	Path    string           `json:"path"`
	Details []JSONDiffDetail `json:"details"`
}

type JSONDiffSummary struct {
	Changes int `json:"changes"`
}

type JSONReportSpec struct {
	Summary     JSONDiffSummary `json:"summary"`
	Differences []JSONDiff      `json:"differences,omitempty"`
}

// JSONReport prints the report in JSON format.
type JSONReport struct {
	Report
	MinorChangeThreshold float64
	NoTableStyle         bool
	DoNotInspectCerts    bool
	OmitHeader           bool
	UseGoPatchPaths      bool
}

// WriteReport writes a JSON report to the provided writer.
func (report *JSONReport) WriteReport(out io.Writer) error {
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	r, err := report.GenReport()
	if err != nil {
		return err
	}

	b, err := json.Marshal(r)
	if err != nil {
		return err
	}

	_, err = writer.WriteString(string(b))
	return err
}

func (report *JSONReport) GenReport() (JSONReportSpec, error) {
	diffs := make([]JSONDiff, len(report.Diffs))

	// Only show the document index if there is more than one document to show
	showPathRoot := len(report.From.Documents) > 1

	for i, diff := range report.Diffs {
		jsonDiff, err := report.generateJSONDiffOutput(diff, report.UseGoPatchPaths, showPathRoot)
		if err != nil {
			return JSONReportSpec{}, err
		}

		diffs[i] = *jsonDiff
	}

	return JSONReportSpec{
		Summary: JSONDiffSummary{
			Changes: len(report.Diffs),
		},
		Differences: diffs,
	}, nil
}

func (report *JSONReport) generateJSONDiffOutput(diff Diff, useGoPatchPaths bool, showPathRoot bool) (*JSONDiff, error) {
	details := make([]JSONDiffDetail, len(diff.Details))
	for i, detail := range diff.Details {
		generatedOutput, err := report.generateJSONDetailOutput(detail)
		if err != nil {
			return nil, err
		}

		details[i] = generatedOutput
	}

	return &JSONDiff{
		Path:    pathToString(diff.Path, useGoPatchPaths, showPathRoot),
		Details: details,
	}, nil
}

func (report *JSONReport) generateJSONDetailOutput(detail Detail) (JSONDiffDetail, error) {
	switch detail.Kind {
	case ADDITION:
		s, err := jsonString(detail.To)
		if err != nil {
			return JSONDiffDetail{}, err
		}

		return JSONDiffDetail{
			Kind:     string(ADDITION),
			Addition: s,
		}, nil

	case REMOVAL:
		s, err := jsonString(detail.From)
		if err != nil {
			return JSONDiffDetail{}, err
		}

		return JSONDiffDetail{
			Kind:    string(REMOVAL),
			Removal: s,
		}, nil

	case MODIFICATION:
		return report.generateJSONDetailOutputModification(detail)

	case ORDERCHANGE:
		return report.generateJSONDetailOutputOrderchange(detail)
	}

	return JSONDiffDetail{}, fmt.Errorf("unsupported detail type %c", detail.Kind)
}

func (report *JSONReport) generateJSONDetailOutputModification(detail Detail) (JSONDiffDetail, error) {
	fromType := humanReadableType(detail.From)
	toType := humanReadableType(detail.To)

	jsonDetail := JSONDiffDetail{
		Kind: string(detail.Kind),
	}

	switch {
	case fromType == "string" && toType == "string":
		// delegate to special string output
		jsonDetail.Addition, jsonDetail.Removal = detail.To.Value, detail.From.Value

	case fromType == "binary" && toType == "binary":
		from, err := base64.StdEncoding.DecodeString(detail.From.Value)
		if err != nil {
			return jsonDetail, err
		}

		to, err := base64.StdEncoding.DecodeString(detail.To.Value)
		if err != nil {
			return jsonDetail, err
		}

		jsonDetail.Addition, jsonDetail.Removal = hex.Dump(to), hex.Dump(from)

	default:
		from, err := jsonString(detail.From)
		if err != nil {
			return jsonDetail, err
		}

		to, err := jsonString(detail.To)
		if err != nil {
			return jsonDetail, err
		}

		jsonDetail.Addition, jsonDetail.Removal = to, from
	}

	return jsonDetail, nil
}

func (report *JSONReport) generateJSONDetailOutputOrderchange(detail Detail) (JSONDiffDetail, error) {
	jsonDetail := JSONDiffDetail{
		Kind: string(detail.Kind),
	}

	switch detail.From.Kind {
	case yamlv3.SequenceNode:
		asJSONArray := func(sequenceNode *yamlv3.Node) (string, error) {
			result := make([]string, len(sequenceNode.Content))
			for i, entry := range sequenceNode.Content {
				result[i] = entry.Value
				if entry.Value == "" {
					s, err := jsonString(entry)
					if err != nil {
						return "", err
					}
					result[i] = s
				}
			}

			b, err := json.Marshal(result)
			if err != nil {
				return "", err
			}

			return string(b), nil
		}

		from, err := asJSONArray(detail.From)
		if err != nil {
			return jsonDetail, err
		}
		to, err := asJSONArray(detail.To)
		if err != nil {
			return jsonDetail, err
		}

		jsonDetail.Addition, jsonDetail.Removal = to, from
	}

	return jsonDetail, nil
}

func jsonString(node *yamlv3.Node) (string, error) {
	if node == nil {
		return "null", nil
	}

	return neat.NewOutputProcessor(false, false, nil).ToCompactJSON(node)
}
