package main

import (
	"github.com/jiuzhou-zhao/blockchain.go/internal/cli"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sgostarter/liblog"
)

func main() {
	logger, err := liblog.NewZapLogger()
	if err != nil {
		panic(err)
	}
	loge.SetGlobalLogger(loge.NewLogger(logger))

	cli := cli.CLI{}
	cli.Run()
}
