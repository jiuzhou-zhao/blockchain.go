package blockchain

import (
	"testing"

	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/liblog"
)

func TestMain(m *testing.M) {
	logger, err := liblog.NewZapLogger()
	if err != nil {
		panic(err)
	}
	loge.SetGlobalLogger(loge.NewLogger(logger))

	m.Run()
}
