package service

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoryObjectStorage is an in-process ObjectStorage. It never touches the
// process upload dir, so a surviving /tmp is irrelevant.
type memoryObjectStorage struct {
	objects map[string][]byte
}

func (m *memoryObjectStorage) Upload(_ context.Context, key string, reader io.Reader, _ string) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[key] = data
	// Deliberately not the /uploads/{tenant}/{file} shape — storeLabel must
	// still persist the canonical label_url that Get can resolve.
	return "https://objects.example.invalid/" + key, nil
}

func (m *memoryObjectStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memoryObjectStorage) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func TestStoreAndReadLabel_ThroughObjectStorage(t *testing.T) {
	store := &memoryObjectStorage{}
	tenantID := uuid.New()
	filename := "d49331b4-4859-4222-ac84-e99aa9f3d4cc.pdf"
	want := []byte("%PDF-1.4 inpost sandbox label")

	labelURL, err := storeLabel(context.Background(), store, "https://api.openoms.org", tenantID, filename, "application/pdf", want)
	require.NoError(t, err)
	assert.Equal(t, "https://api.openoms.org/uploads/"+tenantID.String()+"/"+filename, labelURL)

	got, err := readLabelObject(context.Background(), store, labelURL)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestStoreAndReadLabel_LocalStorageNewReader(t *testing.T) {
	dir := t.TempDir()
	tenantID := uuid.New()
	filename := "label-local.pdf"
	want := []byte("%PDF-1.4 local object storage")

	writer := storage.NewLocalStorage(dir, "https://api.example.invalid")
	labelURL, err := storeLabel(context.Background(), writer, "https://api.example.invalid", tenantID, filename, "application/pdf", want)
	require.NoError(t, err)

	// New LocalStorage on the same durable dir — not the writing process's memory,
	// and not a guessed /tmp/uploads path.
	reader := storage.NewLocalStorage(dir, "https://api.example.invalid")
	got, err := readLabelObject(context.Background(), reader, labelURL)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestReadLabelObject_MissingObject(t *testing.T) {
	store := &memoryObjectStorage{}
	tenantID := uuid.New()
	url := "https://api.openoms.org/uploads/" + tenantID.String() + "/gone.pdf"

	_, err := readLabelObject(context.Background(), store, url)
	require.ErrorIs(t, err, ErrLabelNotAvailable)
}

func TestReadLabelObject_EmptyURL(t *testing.T) {
	_, err := readLabelObject(context.Background(), &memoryObjectStorage{}, "")
	require.ErrorIs(t, err, ErrLabelNotAvailable)
}

func TestReadLabelObject_InvalidURL(t *testing.T) {
	_, err := readLabelObject(context.Background(), &memoryObjectStorage{}, "https://api.example.invalid/labels/missing.pdf")
	require.ErrorIs(t, err, ErrLabelNotAvailable)
}

func TestReadLabelObject_RejectsTraversal(t *testing.T) {
	_, err := readLabelObject(context.Background(), &memoryObjectStorage{}, "https://api.openoms.org/uploads/../secret.pdf")
	require.Error(t, err)
}

func TestReadLabelObject_DoesNotUseProcessTmp(t *testing.T) {
	store := &memoryObjectStorage{}
	tenantID := uuid.New()
	filename := "not-in-tmp.pdf"
	want := []byte("%PDF-1.4 memory only")

	labelURL, err := storeLabel(context.Background(), store, "https://api.openoms.org", tenantID, filename, "application/pdf", want)
	require.NoError(t, err)

	// A leftover file under the process upload dir must not be consulted.
	guess := filepath.Join("uploads", tenantID.String())
	require.NoError(t, os.MkdirAll(guess, 0o750))
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(filepath.Join("uploads", tenantID.String())))
	})
	require.NoError(t, os.WriteFile(filepath.Join(guess, filename), []byte("%PDF-1.4 leftover tmp"), 0o600))

	got, err := readLabelObject(context.Background(), store, labelURL)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestReadLabelObject_NilStorage(t *testing.T) {
	url := "https://api.openoms.org/uploads/" + uuid.New().String() + "/label.pdf"
	_, err := readLabelObject(context.Background(), nil, url)
	require.ErrorIs(t, err, ErrLabelNotAvailable)
}
