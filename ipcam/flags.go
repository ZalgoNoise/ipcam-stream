package ipcam

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ZalgoNoise/zlog/log"
)

func (s *StreamService) Flags() *StreamRequest {

	// s.Log.SetPrefix("ipcam-stream: Flags()")

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

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Flags()",
		Level:  log.LLInfo,
		Msg:    "parsed flags from CLI",
		Metadata: map[string]interface{}{
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
	}

	if *inputCfgFile != "" {
		cfg := &StreamRequest{}

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Flags()",
			Level:  log.LLDebug,
			Msg:    fmt.Sprintf("reading config file %s", *inputCfgFile),
		}

		data, err := os.ReadFile(*inputCfgFile)
		if err != nil {

			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Flags()",
				Level:  log.LLFatal,
				Msg:    "unable to read file",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}

		if err := json.Unmarshal(data, cfg); err != nil {

			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Flags()",
				Level:  log.LLFatal,
				Msg:    "unable to parse JSON data",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}

		if cfg.Logfile != "" {
			s.logfileHandler(cfg.Logfile)
		}

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Flags()",
			Level:  log.LLInfo,
			Msg:    "read config from file successfully",
		}
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
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: logfileHandler()",
			Level:  log.LLFatal,
			Msg:    "failed to setup logfile",
			Metadata: map[string]interface{}{
				"error": err.Error(),
				"path":  path,
			},
		}

	}

	s.Log = log.MultiLogger(
		s.Log,
		log.New("ipcam-stream", log.JSONFormat, logf),
	)

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: logfileHandler()",
		Level:  log.LLDebug,
		Msg:    fmt.Sprintf("added logfile as from input: %s", path),
	}
}
