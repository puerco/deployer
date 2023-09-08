package deploy

import (
	"fmt"

	purl "github.com/package-url/packageurl-go"

	"github.com/puerco/deployer/pkg/deploy/options"
	"github.com/puerco/deployer/pkg/payload"
)

type probeImplementation interface {
	ParsePurl(string) (purl.PackageURL, error)
	GetPackageProbe(purl.PackageURL) (PackageProbe, error)
	FetchDocuments(options.Options, PackageProbe, purl.PackageURL) ([]payload.Document, error)
}

type defaultProberImplementation struct{}

// ParsePurl checks if a purl is correctly formed
func (pi *defaultProberImplementation) ParsePurl(purlString string) (p purl.PackageURL, err error) {
	p, err = purl.FromString(purlString)
	if err != nil {
		return p, fmt.Errorf("verifyinf purl: %w", err)
	}
	return p, nil
}

// GetPackageProbe returns a PackageProbe for the specified purl type
func (pi *defaultProberImplementation) GetPackageProbe(p purl.PackageURL) (PackageProbe, error) {
	if p, ok := probers[p.Type]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("purl type %s not supported", p.Type)
}

// FetchDocuments downloads all security documents using the PackageProbe for
// the specified purl.
func (pi *defaultProberImplementation) FetchDocuments(opts options.Options, pkgProbe PackageProbe, p purl.PackageURL) ([]payload.Document, error) {
	docs, err := pkgProbe.FetchDocuments(opts, p)
	if err != nil {
		return nil, fmt.Errorf("fetching documents: %w", err)
	}
	return docs, nil
}
