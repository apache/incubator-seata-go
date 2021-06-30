package testutil

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"gopkg.in/yaml.v2"
)

func InitServerConfig(t *testing.T) {
	sc := config.StoreConfig{
		MaxGlobalSessionSize: config.DefaultMaxGlobalSessionSize,
	}
	ssc := config.ServerConfig{
		StoreConfig: sc,
	}

	bytes, err := yaml.Marshal(&ssc)
	assert.NoError(t, err, "Marshal() should success")

	file, err := ioutil.TempFile("/tmp", "server_config.*.yml")
	assert.NoError(t, err, "TempFile() should success")

	defer file.Close()
	defer os.Remove(file.Name())

	_, err = file.Write(bytes)
	assert.NoError(t, err, "Write() should success")

	err = config.InitConf(file.Name())
	assert.NoError(t, err, "InitConf() should success")
}
