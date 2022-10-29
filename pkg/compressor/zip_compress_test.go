package compressor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZipCompress(t *testing.T) {
	str := "test"

	g := &Zip{}

	compressRes, err := g.Compress([]byte(str))
	assert.NoError(t, err)
	t.Logf("compress res: %v", string(compressRes))

	decompressRes, err := g.Decompress(compressRes)
	assert.NoError(t, err)
	t.Logf("decompress res: %v", string(decompressRes))

	assert.Equal(t, str, string(decompressRes))
}
