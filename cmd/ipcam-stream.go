package cmd

import (
	"github.com/zalgonoise/ipcam-stream/ipcam"
)

func Run() {
	ipcam.New().Capture()
}
