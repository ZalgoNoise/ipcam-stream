package cmd

import (
	"github.com/ZalgoNoise/ipcam-stream/ipcam"
)

func Run() {
	ipcam.New().Capture()
}
