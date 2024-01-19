package serializer

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrorSerializer(t *testing.T) {
	serializer := ErrorSerializer{}
	expected := errors.New("This is a test error")
	serialized, err := serializer.Serialize(expected)
	assert.Nil(t, err)
	actual, err := serializer.Deserialize(serialized)
	assert.Nil(t, err)
	assert.Equal(t, expected.Error(), actual.Error())
}
