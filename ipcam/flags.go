package ipcam

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/zalgonoise/zlog/log"
	"github.com/zalgonoise/zlog/store"
)

func (s *StreamService) Flags() *StreamRequest {

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

	logCh <- log.NewMessage().Sub("Flags()").Message("parsed flags from CLI").Metadata(log.Field{
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
	}).Build()

	if *inputCfgFile != "" {
		cfg := &StreamRequest{}

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Flags()").Message("reading config file").Metadata(log.Field{"path": *inputCfgFile}).Build()

		data, err := os.ReadFile(*inputCfgFile)
		if err != nil {

			logCh <- log.NewMessage().Level(log.LLFatal).Sub("Flags()").Message("unable to read file").Metadata(log.Field{"path": *inputCfgFile, "error": err.Error()}).Build()
		}

		if err := json.Unmarshal(data, cfg); err != nil {

			logCh <- log.NewMessage().Level(log.LLFatal).Sub("Flags()").Message("unable to parse JSON data").Metadata(log.Field{"path": *inputCfgFile, "error": err.Error()}).Build()

		}

		if cfg.Logfile != "" {
			s.logfileHandler(cfg.Logfile)
		}

		logCh <- log.NewMessage().Sub("Flags()").Message("read config from file successfully").Metadata(log.Field{
			"len":    cfg.TimeLen,
			"vurl":   cfg.VideoURL,
			"aurl":   cfg.AudioURL,
			"tmp":    cfg.TmpDir,
			"out":    cfg.OutDir,
			"ext":    cfg.OutExt,
			"vrate":  cfg.VideoRate,
			"rotate": cfg.Rotate,
			"log":    cfg.Logfile,
			"cfg":    *inputCfgFile,
		}).Build()

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

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("logfileHandler()").Message("reading logfile as from input").Metadata(log.Field{"path": path}).Build()

	logf, err := store.NewLogfile(path)

	if err != nil {
		logCh <- log.NewMessage().Level(log.LLFatal).Sub("logfileHandler()").Message("failed to setup logfile").Metadata(log.Field{"path": path, "error": err.Error()}).Build()
	}

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("logfileHandler()").Message("added logfile as from input").Metadata(log.Field{"path": path}).Build()

	s.Logger = log.MultiLogger(
		s.Logger,
		log.New(
			log.WithPrefix("ipcam-stream"),
			log.WithOut(logf),
			log.FormatJSON,
		),
	)
}
