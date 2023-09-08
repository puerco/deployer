package payload

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
)

// maxInMemSize is the maximum document size that will be handled in
// memory. If the document data grows after this threshold, all I/O
// operations will be performed on a temporary file.
const maxInMemSize = 2097152

// Document is an object that abstracts a security document handled by
// deployer.
type Document struct {
	Format  Format
	tmpFile *os.File
	data    []byte
	reader  *bytes.Reader
}

func NewDocument() *Document {
	return &Document{}
}

func NewDocumentFromFile(path string) (*Document, error) {
	doc := NewDocument()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file to create new document: %w", err)
	}
	if err := doc.ReadData(f); err != nil {
		return nil, fmt.Errorf("reading new document data: %w", err)
	}
	return doc, nil
}

// ReadData takes an io.Reader r and ingests the data of the document
// from it using Read(). Data will be kept in memory until maxInMemSize
// bytes are read, after which data will be dumped to a temporary file.
func (d *Document) ReadData(r io.Reader) (err error) {
	d.data = []byte{}
	d.tmpFile = nil

	for {
		b := make([]byte, 32768)
		rs, readErr := r.Read(b)
		if readErr != nil {
			if readErr != io.EOF {
				return fmt.Errorf("reading document data: %w", readErr)
			}
			break
		}

		switch d.tmpFile == nil {
		case false:
			if _, err := d.tmpFile.Write(b[0:rs]); err != nil {
				return fmt.Errorf("writing data to document file: %w", err)
			}
		case true:
			d.data = append(d.data, b[0:rs]...)

			if len(d.data) > maxInMemSize {
				f, err := os.CreateTemp("", "deployer-payload-*.raw")
				if err != nil {
					return fmt.Errorf("creating temporary file to store payload document: %w", err)
				}
				s, err := f.Write(d.data)
				if err != nil {
					return fmt.Errorf("dumping memory buffer to disk: %w", err)
				}
				if s != len(d.data) {
					return errors.New("dumping document data resulted in short read")
				}
				d.tmpFile = f
				d.data = nil
			}
		}
	}

	if d.tmpFile == nil {
		d.reader = bytes.NewReader(d.data)
	}
	if _, err := d.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking newly buffered data: %w", err)
	}
	return err
}

// Read implements the reader interface to be able to use the document
// wherever io.Reader fits
func (d *Document) Read(b []byte) (n int, err error) {
	if d.tmpFile != nil {
		return d.tmpFile.Read(b)
	}

	return d.reader.Read(b)
}

func (d *Document) Seek(offset int64, whence int) (int64, error) {
	if d.tmpFile != nil {
		return d.tmpFile.Seek(offset, whence)
	}

	return d.reader.Seek(offset, whence)
}

func (d *Document) Hash() (hVal string, err error) {
	if _, err := d.Seek(0, 0); err != nil {
		return "", fmt.Errorf("rewinding document: %w", err)
	}
	h := sha256.New()
	defer func() { _, err = d.Seek(0, 0) }()
	if _, err := io.Copy(h, d); err != nil {
		return hVal, fmt.Errorf("copying data to hash")
	}
	return fmt.Sprintf("%x", h.Sum(nil)), err
}

// Cleanup is a function intended to be run just before a document is discarded.
// It takes care of closing its file and
func (d *Document) Cleanup() {
	d.data = nil
	d.reader = nil
	if d.tmpFile == nil {
		return
	}
	d.tmpFile.Close() //nolint
	os.Remove(d.tmpFile.Name())
}

// inMemory is an internal method to query the status of the data
func (d *Document) inMemory() bool {
	return d.tmpFile == nil
}
