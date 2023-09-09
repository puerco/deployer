package payload

import (
	"fmt"
	"strings"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	openvex "github.com/openvex/go-vex/pkg/vex"
)

type FormatsList []Format

// Has returns a bool flagging if the FormatsList includes format q
func (fl *FormatsList) Has(q string) bool {
	testFormat := Format(q)

	for _, f := range *fl {
		if f.MimeType() == testFormat.MimeType() {
			return true
		}
	}
	return false
}

var List = FormatsList{
	"application/vnd.cyclonedx",
	"text/spdx",
	"application/vnd.slsa",
	"application/vnd.openvex",
}

// Format indicates the mime type of a document
type Format string

func (f Format) Parse() map[string]string {
	str := strings.TrimSpace(string(f))
	version := ""
	encoding := ""
	s := strings.Split(str, ";version=")
	str = s[0]
	if len(s) > 1 {
		version = s[1]
	}

	s = strings.Split(str, "+")
	str = s[0]
	if len(s) > 1 {
		encoding = s[1]
	}
	return map[string]string{
		"mime":     str,
		"version":  version,
		"encoding": encoding,
	}
}

// MimeType returns the mime type part of the format
func (f Format) MimeType() string {
	p := f.Parse()
	return p["mime"]
}

// Version returns the version value part of the format string
func (f Format) Version() string {
	p := f.Parse()
	return p["version"]
}

// Version returns the version value part of the format string
func (f Format) Encoding() string {
	p := f.Parse()
	return p["encoding"]
}

// NewFormatFromIntotoPredicate takes one of the registered in-toto predicate types
// (or an equivalent IRI of another recognized one) and returns a Format.
func NewFormatFromIntotoPredicate(predicateType string) Format {
	version := ""
	if strings.HasPrefix(predicateType, openvex.Context) {
		version = strings.TrimPrefix(predicateType, fmt.Sprintf("%s/v", openvex.Context))
		if version != "" {
			version = fmt.Sprintf(";version=%s", version)
		}
		predicateType = openvex.Context
	}

	switch predicateType {
	case intoto.PredicateCycloneDX:
		return Format("application/vnd.cyclonedx+json")
	case intoto.PredicateSPDX:
		return Format("text/spdx+json")
	case "https://slsa.dev/provenance/v1":
		return Format("application/vnd.slsa+json;version=1")
	case "https://openvex.dev/ns/":
		return Format(fmt.Sprintf("application/vnd.openvex+json%s", version))
	default:
		return ""
	}
}
