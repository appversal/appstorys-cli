package config

import "embed"

//go:embed templates/default.appstorys-cli.yaml
var templates embed.FS

// Template returns the commented example .appstorys-cli.yaml written by
// `config init`. It's a static file rather than a marshaled Config,
// since yaml.v3 marshal drops comments and this file is meant to be
// hand-edited.
func Template() []byte {
	data, err := templates.ReadFile("templates/default.appstorys-cli.yaml")
	if err != nil {
		// Embedded at build time; a missing file here is a build-time
		// bug, not a runtime condition callers need to handle.
		panic(err)
	}
	return data
}
