package deploy

import (
	"fmt"
	"sync"

	purl "github.com/package-url/packageurl-go"

	"github.com/puerco/deployer/pkg/probers/oci"

	"github.com/puerco/deployer/pkg/deploy/options"
	"github.com/puerco/deployer/pkg/payload"
)

var (
	regMtx  sync.RWMutex
	probers = map[string]PackageProbe{}
)

func init() {
	RegisterPackageProbe(purl.TypeOCI, oci.New())
}

func RegisterPackageProbe(purlType string, pp PackageProbe) {
	regMtx.Lock()
	probers[purlType] = pp
	regMtx.Unlock()
}

// Probe is the main object that inspects repositories and looks for security
// documents. To create a new Probe use the `NewProbe` function
type Probe struct {
	impl    probeImplementation
	Options options.Options
}

// NewProbe creates a new deployer probe
func NewProbe() *Probe {
	return &Probe{
		impl:    &defaultProberImplementation{},
		Options: options.Options{},
	}
}

// Fetch probes the package url using a package prober and retrieves all the
// security documents it can find.
func (probe *Probe) Fetch(purlString string) ([]*payload.Document, error) {
	p, err := probe.impl.ParsePurl(purlString)
	if err != nil {
		return nil, fmt.Errorf("validating purl: %w", err)
	}

	pkgProbe, err := probe.impl.GetPackageProbe(probe.Options, p)
	if err != nil {
		return nil, fmt.Errorf("getting package probe for purl type %s: %w", p.Type, err)
	}

	docs, err := probe.impl.FetchDocuments(probe.Options, pkgProbe, p)
	if err != nil {
		return nil, fmt.Errorf("fetching documents: %w", err)
	}

	return docs, nil
}
