package cmd

import (
	"fmt"

	"github.com/ZalgoNoise/ipcam-stream/ipcam"
)

func Run() {
	ipcam, err := ipcam.New()
	if err != nil {
		fmt.Printf("Unable to start service: %s", err)
	}

	ipcam.Capture(ipcam.Flags())
}
