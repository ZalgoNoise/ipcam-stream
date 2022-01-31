package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ZalgoNoise/ipcam-stream/ipcam"
)

func getFlags() (*ipcam.StreamRequest, error) {
	inputLen := flag.Int("len", 60, "Length (in minutes) for each video chunk")
	inputVideoURL := flag.String("vurl", "", "Video's URL endpoint")
	inputAudioURL := flag.String("aurl", "", "Audio's URL endpoint")
	inputTmpDir := flag.String("tmp", "/tmp/", "Temporary directory to place files")
	inputOutDir := flag.String("out", "~/", "Output directory to place files")
	inputExtension := flag.String("ext", ".mp4", "Output extension")
	inputVideoRate := flag.String("vrate", "25", "Input framerate of the MJPEG stream")

	inputCfgFile := flag.String("cfg", "", "Input configuration file (JSON)")

	flag.Parse()

	if *inputCfgFile != "" {
		cfg := &ipcam.StreamRequest{}
		data, err := os.ReadFile(*inputCfgFile)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
		fmt.Printf("[ipcam-stream]\tGot config from file:\n----\n%s\n----\n", string(data))
		return cfg, nil
	}

	return &ipcam.StreamRequest{
		TimeLen:   *inputLen,
		VideoURL:  *inputVideoURL,
		AudioURL:  *inputAudioURL,
		TmpDir:    *inputTmpDir,
		OutDir:    *inputOutDir,
		OutExt:    *inputExtension,
		VideoRate: *inputVideoRate,
	}, nil
}
