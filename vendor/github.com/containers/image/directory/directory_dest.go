package directory

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/image/types"
)

type dirImageDestination struct {
	ref dirReference
}

// newImageDestination returns an ImageDestination for writing to an existing directory.
func newImageDestination(ref dirReference) types.ImageDestination {
	return &dirImageDestination{ref}
}

// Reference returns the reference used to set up this destination.  Note that this should directly correspond to user's intent,
// e.g. it should use the public hostname instead of the result of resolving CNAMEs or following redirects.
func (d *dirImageDestination) Reference() types.ImageReference {
	return d.ref
}

func (d *dirImageDestination) SupportedManifestMIMETypes() []string {
	return nil
}

func (d *dirImageDestination) PutManifest(manifest []byte) error {
	return ioutil.WriteFile(d.ref.manifestPath(), manifest, 0644)
}

// PutBlob writes contents of stream as a blob identified by digest.
// WARNING: The contents of stream are being verified on the fly.  Until stream.Read() returns io.EOF, the contents of the data SHOULD NOT be available
// to any other readers for download using the supplied digest.
// If stream.Read() at any time, ESPECIALLY at end of input, returns an error, PutBlob MUST 1) fail, and 2) delete any data stored so far.
// Note: Calling PutBlob() and other methods may have ordering dependencies WRT other methods of this type. FIXME: Figure out and document.
func (d *dirImageDestination) PutBlob(digest string, stream io.Reader) error {
	blobPath := d.ref.layerPath(digest)
	blobFile, err := ioutil.TempFile(filepath.Dir(blobPath), filepath.Base(blobPath))
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		blobFile.Close()
		if !succeeded {
			os.Remove(blobFile.Name())
		}
	}()

	if _, err := io.Copy(blobFile, stream); err != nil {
		return err
	}
	if err := blobFile.Sync(); err != nil {
		return err
	}
	if err := blobFile.Chmod(0644); err != nil {
		return err
	}
	if err := os.Rename(blobFile.Name(), blobPath); err != nil {
		return nil
	}
	succeeded = true
	return nil
}

func (d *dirImageDestination) PutSignatures(signatures [][]byte) error {
	for i, sig := range signatures {
		if err := ioutil.WriteFile(d.ref.signaturePath(i), sig, 0644); err != nil {
			return err
		}
	}
	return nil
}
