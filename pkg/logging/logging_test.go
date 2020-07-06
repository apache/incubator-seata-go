package logging

import "testing"

func TestSeataLogger_Info(t *testing.T) {
	Logger.Info("there is a bug")
}
