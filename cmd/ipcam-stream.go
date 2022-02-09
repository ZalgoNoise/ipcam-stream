package cmd

import (
	"github.com/ZalgoNoise/ipcam-stream/ipcam"
)

func Run() {
	ipcam := ipcam.New()

	ipcam.Capture(ipcam.Flags())
}
