package oci

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	purl "github.com/package-url/packageurl-go"
	"github.com/sirupsen/logrus"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/oci"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/types"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/puerco/deployer/pkg/deploy/options"
	"github.com/puerco/deployer/pkg/payload"
)

type Prober struct {
	Options options.Options
	impl    ociImplementation
}

func New() *Prober {
	p := &Prober{
		impl:    &defaultImplementation{},
		Options: options.Default,
	}
	p.Options.ProberOptions[purl.TypeOCI] = localOptions{}
	return p
}

type ociImplementation interface {
	VerifyOptions(*options.Options) error
	PurlToReference(options.Options, purl.PackageURL) (name.Reference, error)
	ResolveImageReference(options.Options, name.Reference) (oci.SignedEntity, error)
	DownloadDocuments(options.Options, oci.SignedEntity) ([]*payload.Document, error)
}

type defaultImplementation struct{}

type localOptions struct {
	Platform           string
	Repository         string
	RepositoryOverride string // COSIGN_REPOSITORY or other repo that overrides the purl repo
}

type platformList []struct {
	hash     v1.Hash
	platform *v1.Platform
}

func (pl *platformList) String() string {
	r := []string{}
	for _, p := range *pl {
		r = append(r, p.platform.String())
	}
	return strings.Join(r, ", ")
}

// FetchDocuments implements the logic to search for documents associated with
// a container image
func (prober *Prober) FetchDocuments(opts options.Options, p purl.PackageURL) ([]*payload.Document, error) {
	if err := prober.impl.VerifyOptions(&prober.Options); err != nil {
		return nil, fmt.Errorf("verifying options: %w", err)
	}

	ref, err := prober.impl.PurlToReference(prober.Options, p)
	if err != nil {
		return nil, fmt.Errorf("translating purl to image reference: %w", err)
	}

	if ref == nil {
		return nil, fmt.Errorf("could not resolve image reference from %s", p)
	}

	image, err := prober.impl.ResolveImageReference(prober.Options, ref)
	if err != nil {
		return nil, fmt.Errorf("resolving image reference: %w", err)
	}

	docs, err := prober.impl.DownloadDocuments(prober.Options, image)
	if err != nil {
		return nil, fmt.Errorf("downloading documents from registry: %w", err)
	}

	return docs, nil
}

// PurlToReference reads a purl and generates an image reference. It uses GGCR's
// name package to parse it and returns the reference.
func (di *defaultImplementation) PurlToReference(opts options.Options, p purl.PackageURL) (name.Reference, error) {
	refString, err := purlToRefString(opts, p)
	if err != nil {
		return nil, err
	}

	ref, err := name.ParseReference(refString)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %s: %w", refString, err)

	}
	return ref, nil
}

// purlToRefString returns an OCI reference from an OCI purl
func purlToRefString(opts options.Options, p purl.PackageURL) (string, error) {
	if p.Type != purl.TypeOCI {
		return "", errors.New("package URL is not of type OCI")
	}

	if p.Name == "" {
		return "", errors.New("parsed pacakge URL did not return a package name")
	}

	qualifiers := p.Qualifiers.Map()

	var refString = p.Name
	if _, ok := qualifiers["repository_url"]; ok {
		refString = fmt.Sprintf(
			"%s/%s", strings.TrimSuffix(qualifiers["repository_url"], "/"), p.Name,
		)
	} else if opts.ProberOptions[purl.TypeOCI].(localOptions).Repository != "" {
		refString = fmt.Sprintf(
			"%s/%s", strings.TrimSuffix(opts.ProberOptions[purl.TypeOCI].(localOptions).Repository, "/"), p.Name,
		)
	}

	// Of a repo override is set, rewrite the ref
	if opts.ProberOptions[purl.TypeOCI].(localOptions).RepositoryOverride != "" {
		refString = fmt.Sprintf(
			"%s/%s", strings.TrimSuffix(opts.ProberOptions[purl.TypeOCI].(localOptions).RepositoryOverride, "/"), p.Name,
		)
	}

	if p.Version != "" {
		refString = fmt.Sprintf("%s@%s", refString, p.Version)
	}

	// We add a tag, bu only if no digest is defined
	if _, ok := qualifiers["tag"]; ok && p.Version == "" {
		refString += ":" + qualifiers["tag"]
	}
	return refString, nil
}

// getIndexPlatforms returns the platforms of the single arch images fronted by
// an image index.
func getIndexPlatforms(idx oci.SignedImageIndex) (platformList, error) {
	im, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("fetching index manifest: %w", err)
	}

	platforms := platformList{}
	for _, m := range im.Manifests {
		if m.Platform == nil {
			continue
		}
		platforms = append(platforms, struct {
			hash     v1.Hash
			platform *v1.Platform
		}{m.Digest, m.Platform})
	}
	return platforms, nil
}

// ResolveImageReference takes an image ref returns the image it is pointing to.
// This process involves checking if the image is an index, a single or multi arch
// image, if we have an archi in options, etc, etc.
func (di *defaultImplementation) ResolveImageReference(opts options.Options, ref name.Reference) (oci.SignedEntity, error) {
	// o := options.RegistryOptions{}
	// ctx := context.Background()

	if ref == nil {
		return nil, fmt.Errorf("got nil value when trying to resolve OCI image reference")
	}

	ociremoteOpts := []ociremote.Option{}
	// ociremoteOpts := []ociremote.Option{ociremote.WithRemoteOptions(o.GetRegistryClientOpts(ctx)...)}
	//	if o.RefOpts.TagPrefix != "" {
	//		opts = append(ociremoteOpts, ociremote.WithPrefix(o.RefOpts.TagPrefix))
	//	}
	targetRepoOverride, err := ociremote.GetEnvTargetRepository()
	if err != nil {
		return nil, err
	}
	if (targetRepoOverride != name.Repository{}) {
		ociremoteOpts = append(ociremoteOpts, ociremote.WithTargetRepository(targetRepoOverride))
	}

	se, err := ociremote.SignedEntity(ref, ociremoteOpts...)
	if err != nil {
		return nil, err
	}

	idx, isIndex := se.(oci.SignedImageIndex)

	// We only allow --platform on multiarch indexes
	if opts.ProberOptions[purl.TypeOCI].(localOptions).Platform != "" && !isIndex {
		return nil, fmt.Errorf("specified reference is not a multiarch image")
	}

	if opts.ProberOptions[purl.TypeOCI].(localOptions).Platform != "" && isIndex {
		targetPlatform, err := v1.ParsePlatform(opts.ProberOptions[purl.TypeOCI].(localOptions).Platform)
		if err != nil {
			return nil, fmt.Errorf("parsing platform: %w", err)
		}
		platforms, err := getIndexPlatforms(idx)
		if err != nil {
			return nil, fmt.Errorf("getting available platforms: %w", err)
		}

		platforms = matchPlatform(targetPlatform, platforms)
		if len(platforms) == 0 {
			return nil, fmt.Errorf("unable to find an attestation for %s", targetPlatform.String())
		}
		if len(platforms) > 1 {
			return nil, fmt.Errorf(
				"platform spec matches more than one image architecture: %s",
				platforms.String(),
			)
		}

		nse, err := idx.SignedImage(platforms[0].hash)
		if err != nil {
			return nil, fmt.Errorf("searching for %s image: %w", platforms[0].hash.String(), err)
		}
		if nse == nil {
			return nil, fmt.Errorf("unable to find image %s", platforms[0].hash.String())
		}
		se = nse
	}

	return se, nil
}

// matchPlatform filters a list of platforms returning only those matching
// a base. "Based" on ko's internal equivalent while it moves to GGCR.
// https://github.com/google/ko/blob/e6a7a37e26d82a8b2bb6df991c5a6cf6b2728794/pkg/build/gobuild.go#L1020
func matchPlatform(base *v1.Platform, list platformList) platformList {
	ret := platformList{}
	for _, p := range list {
		if base.OS != "" && base.OS != p.platform.OS {
			continue
		}
		if base.Architecture != "" && base.Architecture != p.platform.Architecture {
			continue
		}
		if base.Variant != "" && base.Variant != p.platform.Variant {
			continue
		}

		if base.OSVersion != "" && p.platform.OSVersion != base.OSVersion {
			if base.OS != "windows" {
				continue
			} else { //nolint: revive
				if pcount, bcount := strings.Count(base.OSVersion, "."), strings.Count(p.platform.OSVersion, "."); pcount == 2 && bcount == 3 {
					if base.OSVersion != p.platform.OSVersion[:strings.LastIndex(p.platform.OSVersion, ".")] {
						continue
					}
				} else {
					continue
				}
			}
		}
		ret = append(ret, p)
	}

	return ret
}

// DownloadDocuments retrieves attested or attached document from the registry
func (di *defaultImplementation) DownloadDocuments(opts options.Options, se oci.SignedEntity) ([]*payload.Document, error) {
	//	ctx := context.Background()

	/*
		ociremoteOpts, err := regOpts.ClientOpts(ctx)
		if err != nil {
			return err
		}
	*/
	// ociremoteOpts := []ociremote.Option{}

	docs := []*payload.Document{}

	// Fetch all the attestations from the registry, we'll decide which ones
	// we want using the types in the options
	attestations, err := cosign.FetchAttestations(se, "")
	if err != nil {
		return nil, fmt.Errorf("fetching attestations: %w", err)
	}

	for _, att := range attestations {
		// We only understand intoto attestations for now.
		if att.PayloadType != types.IntotoPayloadType {
			continue
		}

		pload, err := base64.StdEncoding.DecodeString(att.PayLoad)
		if err != nil {
			return nil, fmt.Errorf("decoding document: %w", err)
		}

		statement := intoto.Statement{}
		if err := json.Unmarshal(pload, &statement); err != nil {
			return nil, fmt.Errorf("unmarshalling attestation: %w", err)
		}

		// If we're dealing with a document type we don't know
		// skip it at this point
		format := payload.NewFormatFromIntotoPredicate(statement.PredicateType)
		if format == "" {
			logrus.Warnf("ignoring attached document of unsupported type %s", statement.PredicateType)
			continue
		}

		if !opts.Formats.Has(format) {
			logrus.Warnf("ignoring attached document of type %s", statement.PredicateType)
			continue
		}

		b := new(bytes.Buffer)
		encoder := json.NewEncoder(b)
		encoder.SetIndent("", "    ")

		// Encode the document body
		if err := encoder.Encode(statement.Predicate); err != nil {
			return nil, fmt.Errorf("marshaling predicate: %w", err)
		}

		doc := payload.NewDocument()
		doc.ReadData(b)
		doc.Format = format
		docs = append(docs, doc)
	}

	return docs, nil
}

// VerifyOptions checks the options and returns an error if there is something wrong
func (di *defaultImplementation) VerifyOptions(opts *options.Options) error {
	if opts.ProberOptions == nil {
		opts.ProberOptions = map[string]interface{}{}
	}
	if _, ok := opts.ProberOptions[purl.TypeOCI]; !ok {
		opts.ProberOptions[purl.TypeOCI] = localOptions{}
	}
	return nil
}

// SetOptions sets the probe's options
func (prober *Prober) SetOptions(opts options.Options) {
	prober.Options = opts
}
