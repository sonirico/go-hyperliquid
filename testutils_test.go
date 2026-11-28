package hyperliquid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keyCoveragePayload = `{"a":1,"b":{"c":2,"d":[{"e":3}]}}`

type keyCoverageComplete struct {
	A int `json:"a"`
	B struct {
		C int `json:"c"`
		D []struct {
			E int `json:"e"`
		} `json:"d"`
	} `json:"b"`
}

type keyCoverageMissing struct {
	A int `json:"a"`
	B struct {
		C int        `json:"c"`
		D []struct{} `json:"d"`
	} `json:"b"`
}

func TestAssertKeysCovered(t *testing.T) {
	t.Run("complete struct passes", func(t *testing.T) {
		assertKeysCovered(t, []byte(keyCoveragePayload), &keyCoverageComplete{})
	})

	t.Run("missing key is reported with its path", func(t *testing.T) {
		missing, err := uncoveredKeyPaths([]byte(keyCoveragePayload), &keyCoverageMissing{})

		require.NoError(t, err)
		assert.Equal(t, []string{".b.d[].e"}, missing)
	})
}

func newTestLiveFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", "live", name+".json"))
	require.NoError(t, err)
	return raw
}
