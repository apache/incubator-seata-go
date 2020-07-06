package schema

type KeyType byte

const (
	NULL KeyType = iota
	PRIMARY_KEY
)

type Field struct {
	Name    string
	KeyType KeyType
	Type    int32
	Value   interface{}
}
