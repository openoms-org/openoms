package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProbeRow feeds canned column values into scanProbe's Scan destinations,
// in providerProbeColumns order.
type fakeProbeRow struct {
	config []byte
}

func (f *fakeProbeRow) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = uuid.New()       // id
	*(dest[1].(*uuid.UUID)) = uuid.New()       // provider_version_id
	*(dest[2].(*string)) = "auth_check"        // probe_type
	*(dest[3].(*string)) = "auth"              // label
	*(dest[4].(*bool)) = false                 // destructive
	*(dest[5].(*bool)) = true                  // required
	*(dest[6].(*[]byte)) = f.config            // config
	*(dest[7].(*time.Time)) = time.Now().UTC() // created_at
	*(dest[8].(*time.Time)) = time.Now().UTC() // updated_at
	return nil
}

func TestScanProbe_CorruptConfigSurfacesError(t *testing.T) {
	p, err := scanProbe(&fakeProbeRow{config: []byte(`{"limit":`)})
	require.Error(t, err, "a corrupt probe config must not be silently dropped")
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "decode probe config")
}

func TestScanProbe_ValidConfigDecodes(t *testing.T) {
	p, err := scanProbe(&fakeProbeRow{config: []byte(`{"limit": 5}`)})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, map[string]any{"limit": float64(5)}, p.Config)
}

func TestScanProbe_EmptyConfigSkipsDecode(t *testing.T) {
	p, err := scanProbe(&fakeProbeRow{config: nil})
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Nil(t, p.Config)
}
