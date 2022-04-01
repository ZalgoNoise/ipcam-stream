package ipcam

import (
	"os"
	"os/signal"
	"time"

	"github.com/zalgonoise/zlog/log"
)

var logCh chan *log.LogMessage
var done chan struct{}

type StreamService struct {
	request *StreamRequest
	// response *StreamResponse
	Stream *SplitStream
	Logger log.Logger
}

type StreamRequest struct {
	TimeLen   int    `json:"length,omitempty"`
	VideoURL  string `json:"videoURL,omitempty"`
	AudioURL  string `json:"audioURL,omitempty"`
	TmpDir    string `json:"tmpDir,omitempty"`
	OutDir    string `json:"outDir,omitempty"`
	OutExt    string `json:"extension,omitempty"`
	VideoRate string `json:"videoRate,omitempty"`
	Rotate    int    `json:"rotate,omitempty"`
	Logfile   string `json:"log,omitempty"`
}

var std = log.New(log.WithPrefix("ipcam-stream"), log.FormatText)

func New(loggers ...log.Logger) *StreamService {
	service := &StreamService{
		request: &StreamRequest{},
	}

	// init multilogger
	if len(loggers) == 0 {
		service.Logger = std
	} else {
		service.Logger = log.MultiLogger(loggers...)
	}

	chLogger := log.NewLogCh(service.Logger)

	logCh, done = chLogger.Channels()

	go func() {
		for {
			select {
			case msg := <-logCh:
				service.Logger.Log(msg)
			case <-done:
				service.Logger.Log(log.NewMessage().Message("done signal received").Build())
				return
			}
		}
	}()

	logCh <- log.NewMessage().Sub("New()").Message("service initialized").Build()

	return service
}

func (s *StreamService) Capture() {
	s.request = s.Flags()

	logCh <- log.NewMessage().Sub("Capture()").Message("new capture request").Metadata(log.Field{
		"length":    s.request.TimeLen,
		"videoURL":  s.request.VideoURL,
		"audioURL":  s.request.AudioURL,
		"tmpDir":    s.request.TmpDir,
		"outDir":    s.request.OutDir,
		"extension": s.request.OutExt,
		"videoRate": s.request.VideoRate,
		"rotate":    s.request.Rotate,
		"log":       s.request.Logfile,
	}).Build()

	// initialize service
	//  - clear cache
	cache := &cache{}

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("loading cache").Build()

	err := cache.load(s.request.TmpDir)
	if err != nil {
		logCh <- log.NewMessage().Level(log.LLFatal).Sub("Capture()").Message("failed to load cache").Metadata(log.Field{"error": err.Error()}).Build()
	}

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("clearing existing cache").Build()

	errList := cache.clear()
	if len(errList) > 0 {
		for _, err := range errList {
			logCh <- log.NewMessage().Level(log.LLError).Sub("Capture()").Message("failed to clear cache").Metadata(log.Field{"error": err.Error()}).Build()
		}
	}

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("cache is ready; starting capture").Build()

	s.newCaptureResponse(s.request)

}

func (s *StreamService) newCaptureResponse(req *StreamRequest) {
	// handle signal: interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(videoRate string) {
		<-c

		logCh <- log.NewMessage().Sub("Capture()").Message("received signal interrupt -- merging cached files").Build()

		s.Stream.Merge(videoRate)

		logCh <- log.NewMessage().Sub("Capture()").Message("merge completed -- exiting").Build()

		os.Exit(0)

	}(req.VideoRate)

	for {
		now := time.Now()

		folderDate := now.Format("2006-01-02")
		fileDate := now.Format("2006-01-02-15-04-05")

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("setting stream timestamp").Metadata(log.Field{"date": fileDate}).Build()
		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("loading output directory").Metadata(log.Field{"path": req.OutDir}).Build()

		dir := &dir{}

		if err := dir.load(req.OutDir); err != nil {
			logCh <- log.NewMessage().Level(log.LLFatal).Sub("Capture()").Message("unable to load output directory").Metadata(log.Field{"error": err.Error()}).Build()
		}

		if !dir.exists(folderDate) {
			logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("creating new output folder").Metadata(log.Field{"path": req.OutDir + folderDate}).Build()

			dir.mkdir(folderDate)
		}

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("started rotate routine").Metadata(log.Field{"days": req.Rotate}).Build()

		go dir.rotate(now, req.Rotate)

		s.Stream = &SplitStream{
			audio:   &Stream{},
			video:   &Stream{},
			outPath: req.OutDir + folderDate + "/" + fileDate + req.OutExt,
		}

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("starting to capture audio/video HTTP stream").Build()

		s.Stream.audio.SetSource(req.AudioURL)
		s.Stream.video.SetSource(req.VideoURL)
		s.Stream.audio.SetOutput(req.TmpDir + "a-" + fileDate + "_temp.mp4")
		s.Stream.video.SetOutput(req.TmpDir + "v-" + fileDate + "_temp.mp4")

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("stream started").Metadata(log.Field{"deadline": req.TimeLen}).Build()

		s.Stream.SyncTimeout(time.Minute * time.Duration(req.TimeLen))

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Capture()").Message("merging stream").Metadata(log.Field{"video_rate": req.VideoRate}).Build()

		go s.Stream.Merge(req.VideoRate)
	}

}
