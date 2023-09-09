package payload

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	for i, filename := range []string{
		"testdata/chainguard-node.spdx.json", // Large SBOM (dump to disk)
		"testdata/curl.spdx.json",            // Smaller SBOM (memory based)
	} {
		doc := NewDocument()
		f, err := os.Open(filename)
		require.NoError(t, err)
		defer f.Close()
		defer doc.Cleanup()

		err = doc.ReadData(f)
		require.NoError(t, err, fmt.Sprintf("reading SBOM %d", i))
		_, err = f.Seek(0, 0)
		require.NoError(t, err, "rewinding file")

		if i == 0 {
			require.False(t, doc.inMemory())
		} else {
			require.True(t, doc.inMemory())
		}

		fileHash := sha256.New()
		_, err = io.Copy(fileHash, f)
		require.NoError(t, err)

		docHash := sha256.New()
		_, err = io.Copy(docHash, doc)
		require.NoError(t, err)

		require.Equal(
			t, fmt.Sprintf("%x", fileHash.Sum(nil)),
			fmt.Sprintf("%x", docHash.Sum(nil)),
			fmt.Sprintf("checking SBOM #%d hash", i),
		)
	}
}

func TestHash(t *testing.T) {
	for filename, goldenHash := range map[string]string{
		"testdata/chainguard-node.spdx.json": "135f9c639185d53a30c41f9df150202f0c35b63691221ec246fec480f6afdef0",
		"testdata/curl.spdx.json":            "b4d5f8d5586f6098b86e054fac988b9c0db807c95c927422cf12e3e46ee827e7",
	} {
		doc, err := NewDocumentFromFile(filename)
		require.NoError(t, err)
		docHash, err := doc.Hash()
		require.NoError(t, err)
		require.Equal(t, goldenHash, docHash)
	}
}
