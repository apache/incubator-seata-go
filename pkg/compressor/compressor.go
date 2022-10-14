package compressor

type CompressorType int8

const (
	None    CompressorType = 0
	Gzip    CompressorType = 1
	Zip     CompressorType = 1
	Sevenz  CompressorType = 1
	Bzip2   CompressorType = 1
	Lz4     CompressorType = 1
	Default CompressorType = 1
	Zstd    CompressorType = 1
)

type Compressor interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
	GetCompressorType() CompressorType
}
