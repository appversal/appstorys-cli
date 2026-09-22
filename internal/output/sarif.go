package output

import (
	"encoding/json"
	"io"
)

// SARIFResult is one lint finding rendered as a SARIF result.
type SARIFResult struct {
	RuleID  string
	Level   string // "error" | "warning" | "note"
	Message string
	File    string
	Line    int
	Col     int
}

// SeverityToSARIFLevel maps a rules.Severity string to a SARIF level.
func SeverityToSARIFLevel(severity string) string {
	switch severity {
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "note"
	}
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
}

// WriteSARIF writes results as a SARIF 2.1.0 log with a single run.
func WriteSARIF(w io.Writer, toolName, toolVersion string, results []SARIFResult) error {
	log := sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool:    sarifTool{Driver: sarifDriver{Name: toolName, Version: toolVersion}},
				Results: make([]sarifResult, 0, len(results)),
			},
		},
	}
	for _, r := range results {
		log.Runs[0].Results = append(log.Runs[0].Results, sarifResult{
			RuleID:  r.RuleID,
			Level:   r.Level,
			Message: sarifMessage{Text: r.Message},
			Locations: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: r.File},
						Region:           sarifRegion{StartLine: r.Line, StartColumn: r.Col},
					},
				},
			},
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}
