package compressor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeflateCompress(t *testing.T) {
	ts := []struct {
		text string
	}{
		{
			text: "Don't communicate by sharing memory, share memory by communicating.",
		},
		{
			text: "Concurrency is not parallelism.",
		},
		{
			text: "The bigger the interface, the weaker the abstraction.",
		},
		{
			text: "Documentation is for users.",
		},
	}

	dc := &DeflateCompress{}
	assert.EqualValues(t, CompressorDeflate, dc.GetCompressorType())

	for _, s := range ts {
		var data []byte = []byte(s.text)
		dataCompressed, _ := dc.Compress(data)
		ret, _ := dc.Decompress(dataCompressed)
		assert.EqualValues(t, s.text, string(ret))
	}
}
