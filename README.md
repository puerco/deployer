# SBOMs Away!

> 🛑 __WARNING ALPHA SOFTWARE__ 🛑
>
> This repository is under development. This module is not yet inteded to be
> used. Feel free to file issues or contribute but things will be changing 
> quickly. Thanks!

deployer is a library designed to locate payloads of security documents (SBOMs,
VEX, Vulnerability Scans, etc) associated to packages and respositories. The 
libry _deploys_ SBOMs them into your application :)

The main goal of `deployer` is to avoid rewriting logic to discover, locate and
retrieve SBOMs (Software Bills of Materials) and other security documents.

```golang

// Create a new deployer probe:
probe := deployer.Probe()

// Retrieve SBOMs and all other documents
sboms, err := probe.FetchDocuments("pkg:oci/debian:latest")
if err {
    os.Exit(1)
}

// Deploy SBOMs!
myapp.Deploy(sboms)

```

## Mode Of Operation

__diagram__

The deployer `Probe` takes a [package URL](https://github.com/package-url/purl-spec)
(purl) and, looks for SBOMs and other documents associated to the package using
one of a series of pluggable `PackageProbers` that understand each
[purl type](https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst).
The `PackageProbers` try a number of common discovery mechanisms or, when available,
use native tooling to locate the data.

The probe can be configured to look for specific types of documents and supports
various options that configure each prober type.

## purl Types Supported

Supported purl types rely on implemented `PackageProbers`. This table summarizes
the currently supported types.

| Purl Type | Supported | Implementation Details |
| --- | --- | --- |
| `oci` | ✅ | Looks for attestations attached to container images using [sigstore's attestation spec](https://github.com/sigstore/cosign/blob/main/specs/ATTESTATION_SPEC.md). SBOM support (.sbom "images") is planned but not implemented yet. |
| `github` | TBD | Find security documents published as assets of github releases. |
| `git` | TBD | Discover SBOMs and other documents published on git respositories. |

More purl types may be implemented in the near future if the need arises. Feel free
to open issues to request them.
