package ipcam

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/ZalgoNoise/zlog/log"
)

func (s *StreamService) Flags() *StreamRequest {
	s.Log.SetPrefix("ipcam-stream: Flags()")

	inputLen := flag.Int("len", 60, "Length (in minutes) for each video chunk")
	inputVideoURL := flag.String("vurl", "", "Video's URL endpoint")
	inputAudioURL := flag.String("aurl", "", "Audio's URL endpoint")
	inputTmpDir := flag.String("tmp", "/tmp/", "Temporary directory to place files")
	inputOutDir := flag.String("out", "~/", "Output directory to place files")
	inputExtension := flag.String("ext", ".mp4", "Output extension")
	inputVideoRate := flag.String("vrate", "25", "Input framerate of the MJPEG stream")
	inputRotate := flag.Int("rotate", 7, "Number of days to keep data streams; rotate will remove streams older than # days")
	inputLogfile := flag.String("log", "/tmp/ipcam-stream.log", "File to register logs")

	inputCfgFile := flag.String("cfg", "", "Input configuration file (JSON)")

	flag.Parse()

	// handle logfile config
	if *inputLogfile != "" {
		s.logfileHandler(*inputLogfile)
	}

	s.Log.Fields(
		map[string]interface{}{
			"len":    *inputLen,
			"vurl":   *inputVideoURL,
			"aurl":   *inputAudioURL,
			"tmp":    *inputTmpDir,
			"out":    *inputOutDir,
			"ext":    *inputExtension,
			"vrate":  *inputVideoRate,
			"rotate": *inputRotate,
			"log":    *inputLogfile,
			"cfg":    *inputCfgFile,
		},
	).Infof("parsed flags from CLI")

	if *inputCfgFile != "" {
		cfg := &StreamRequest{}

		s.Log.Debugf("reading config file %s", *inputCfgFile)
		data, err := os.ReadFile(*inputCfgFile)
		if err != nil {
			s.Log.Fatalf("unable to read file with error: %s", err)
		}

		if err := json.Unmarshal(data, cfg); err != nil {
			s.Log.Fatalf("unable to parse JSON data: %s", err)
		}

		if cfg.Logfile != "" {
			s.logfileHandler(cfg.Logfile)
		}

		s.Log.Info("read config from file successfully")

		return cfg
	}

	return &StreamRequest{
		TimeLen:   *inputLen,
		VideoURL:  *inputVideoURL,
		AudioURL:  *inputAudioURL,
		TmpDir:    *inputTmpDir,
		OutDir:    *inputOutDir,
		OutExt:    *inputExtension,
		VideoRate: *inputVideoRate,
		Rotate:    *inputRotate,
	}
}

func (s *StreamService) logfileHandler(path string) {
	logf, err := log.NewLogfile(path)
	if err != nil {
		s.Log.Fatalf("failed to setup logfile %s with error: %s", path, err)
	}

	s.Log = log.MultiLogger(
		s.Log,
		log.New("ipcam-stream", log.JSONFormat, logf),
	)

	s.Log.SetPrefix("ipcam-stream: logfileHandler()").Infof("added logfile as from input: %s", path)
}
