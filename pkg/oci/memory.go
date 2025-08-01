package oci

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ Store = &Memory{}

type Memory struct {
	descs  map[digest.Digest]ocispec.Descriptor
	blobs  map[digest.Digest][]byte
	tags   map[string]digest.Digest
	images []Image
	mx     sync.RWMutex
}

func NewMemory() *Memory {
	return &Memory{
		images: []Image{},
		tags:   map[string]digest.Digest{},
		descs:  map[digest.Digest]ocispec.Descriptor{},
		blobs:  map[digest.Digest][]byte{},
	}
}

func (m *Memory) Name() string {
	return "memory"
}

func (m *Memory) Verify(ctx context.Context) error {
	return nil
}

func (m *Memory) Subscribe(ctx context.Context) (<-chan OCIEvent, error) {
	return nil, nil
}

func (m *Memory) ListImages(ctx context.Context) ([]Image, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	return m.images, nil
}

func (m *Memory) Resolve(ctx context.Context, ref string) (digest.Digest, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	dgst, ok := m.tags[ref]
	if !ok {
		return "", fmt.Errorf("could not resolve tag %s to a digest", ref)
	}
	return dgst, nil
}

func (m *Memory) ListContents(ctx context.Context) ([]Content, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	contents := []Content{}
	for k := range m.blobs {
		contents = append(contents, Content{Digest: k})
	}
	return contents, nil
}

func (m *Memory) Size(ctx context.Context, dgst digest.Digest) (int64, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	b, ok := m.blobs[dgst]
	if !ok {
		return 0, errors.Join(ErrNotFound, fmt.Errorf("size information for digest %s not found", dgst))
	}
	return int64(len(b)), nil
}

func (m *Memory) GetManifest(ctx context.Context, dgst digest.Digest) ([]byte, string, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	desc, ok := m.descs[dgst]
	if !ok {
		return nil, "", errors.Join(ErrNotFound, fmt.Errorf("manifest with digest %s not found", dgst))
	}
	b, ok := m.blobs[dgst]
	if !ok {
		return nil, "", errors.Join(ErrNotFound, fmt.Errorf("manifest with digest %s not found", dgst))
	}
	return b, desc.MediaType, nil
}

func (m *Memory) GetBlob(ctx context.Context, dgst digest.Digest) (io.ReadSeekCloser, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	b, ok := m.blobs[dgst]
	if !ok {
		return nil, errors.Join(ErrNotFound, fmt.Errorf("blob with digest %s not found", dgst))
	}
	rc := io.NewSectionReader(bytes.NewReader(b), 0, int64(len(b)))
	return struct {
		io.ReadSeeker
		io.Closer
	}{
		ReadSeeker: rc,
		Closer:     io.NopCloser(nil),
	}, nil
}

func (m *Memory) AddImage(img Image) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.images = append(m.images, img)
	tagName, ok := img.TagName()
	if !ok {
		return
	}
	m.tags[tagName] = img.Digest
}

func (m *Memory) Write(desc ocispec.Descriptor, b []byte) {
	m.mx.Lock()
	defer m.mx.Unlock()

	m.descs[desc.Digest] = desc
	m.blobs[desc.Digest] = b
}
