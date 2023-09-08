package main

import (
	"fmt"
	"os"

	"github.com/puerco/deployer/pkg/deploy"
)

// This program is a development sample. It is intended for
// development purposes only. It is hardcoded to a single
// purl. Keep an eye at the repository README for further
// news on when it becomes usable.

func main() {
	probe := deploy.NewProbe()
	docs, err := probe.Fetch("pkg:oci/curl?repository_url=cgr.dev/chainguard/")
	if err != nil {
		fmt.Fprintf(os.Stdout, "fetching documents: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%+v", docs)
}
