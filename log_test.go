package log

import (
	"testing"
)

func TestEncoderConsole(t *testing.T) {
	SetEncodingConsole()
	Info("aaa")
	Error("error")
}
