package compressor

type CompressorType int8

const (
	None    CompressorType = 0
	Gzip    CompressorType = 1
	Zip     CompressorType = 2
	Sevenz  CompressorType = 3
	Bzip2   CompressorType = 4
	Lz4     CompressorType = 5
	Default CompressorType = 6
	Zstd    CompressorType = 7
)

type Compressor interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
	GetCompressorType() CompressorType
}
