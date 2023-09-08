package deploy

import (
	purl "github.com/package-url/packageurl-go"

	"github.com/puerco/deployer/pkg/deploy/options"
	"github.com/puerco/deployer/pkg/payload"
)

type PackageProbe interface {
	FetchDocuments(options.Options, purl.PackageURL) ([]payload.Document, error)
}
