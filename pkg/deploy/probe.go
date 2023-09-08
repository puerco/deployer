package deploy

import (
	"fmt"
	"sync"

	"github.com/puerco/deployer/pkg/deploy/options"
	"github.com/puerco/deployer/pkg/payload"
)

var (
	regMtx  sync.RWMutex
	probers = map[string]PackageProbe{}
)

func init() {

}

func RegisterPackageProbe(purlType string, pp PackageProbe) {
	regMtx.Lock()
	probers[purlType] = pp
	regMtx.Unlock()
}

type Probe struct {
	impl    probeImplementation
	Options options.Options
}

func NewProbe() *Probe {
	return &Probe{
		impl:    &defaultProberImplementation{},
		Options: options.Options{},
	}
}

// Fetch probes the package url using a package prober and retrieves all the
// security documents it can find.
func (probe *Probe) Fetch(purlString string) ([]payload.Document, error) {
	p, err := probe.impl.ParsePurl(purlString)
	if err != nil {
		return nil, fmt.Errorf("validating purl: %w", err)
	}

	pkgProbe, err := probe.impl.GetPackageProbe(p)
	if err != nil {
		return nil, fmt.Errorf("getting package probe for purl type %s: %w", p.Type, err)
	}

	docs, err := probe.impl.FetchDocuments(probe.Options, pkgProbe, p)
	if err != nil {
		return nil, fmt.Errorf("fetching documents: %w", err)
	}

	return docs, nil
}
