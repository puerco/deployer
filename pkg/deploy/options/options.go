package options

import "github.com/puerco/deployer/pkg/payload"

type Options struct {
	// Formats captures a list of formats a probe wants to find. If a document
	// is found but its format is not in the list it will be ignored. By default
	// the formats list is set to payload.AllFormats
	Formats payload.FormatsList

	// Prober options is a map keyed by purl types that holds free form structs
	// that are passed as options to the corresponding PackageProber.
	ProberOptions map[string]interface{}
}

var Default = Options{
	Formats:       payload.AllFormats,
	ProberOptions: map[string]interface{}{},
}
