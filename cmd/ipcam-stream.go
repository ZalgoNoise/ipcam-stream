package cmd

import (
	"fmt"
	"os"

	"github.com/ZalgoNoise/ipcam-stream/ipcam"
)

func Run() {
	ipcam := ipcam.New()

	request, err := getFlags()
	if err != nil {
		fmt.Printf("[ipcam-stream] [Critical] failed to parse config:\n%s\n----\n", err)
		os.Exit(1)
	}

	ipcam.Capture(request)

}
