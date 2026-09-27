// Package output renders scan/lint/catalog results as a table, JSON,
// SARIF or Markdown. It stays decoupled from internal/extract,
// internal/rules and internal/catalog: callers hand it plain
// headers/rows or JSON-able values, so it doesn't need to know those
// packages' types.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
)

// Envelope wraps a JSON payload with a versioned schema, per the spec's
// "JSON schema is versioned (schemaVersion)" requirement.
type Envelope struct {
	SchemaVersion int `json:"schemaVersion"`
	Data          any `json:"data"`
}

// WriteJSON writes data wrapped in an Envelope. A nil top-level slice is
// written as [] rather than null, so "no results" is an empty list for
// every command instead of a value consumers must special-case.
func WriteJSON(w io.Writer, data any) error {
	if rv := reflect.ValueOf(data); rv.Kind() == reflect.Slice && rv.IsNil() {
		data = reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(Envelope{SchemaVersion: 1, Data: data})
}

// WriteTable writes headers and rows as an aligned table.
func WriteTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.ToUpper(strings.Join(headers, "\t"))); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(r, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// WriteMarkdownTable writes headers and rows as a Markdown table.
func WriteMarkdownTable(w io.Writer, headers []string, rows [][]string) error {
	if _, err := fmt.Fprintln(w, "| "+strings.Join(headers, " | ")+" |"); err != nil {
		return err
	}
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	if _, err := fmt.Fprintln(w, "| "+strings.Join(sep, " | ")+" |"); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := fmt.Fprintln(w, "| "+strings.Join(r, " | ")+" |"); err != nil {
			return err
		}
	}
	return nil
}
