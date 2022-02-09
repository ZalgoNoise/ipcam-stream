package ipcam

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ZalgoNoise/zlog/log"
)

var LogCh = make(chan log.ChLogMessage)

type StreamService struct {
	request *StreamRequest
	// response *StreamResponse
	Stream *SplitStream
	Log    log.LoggerI
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

// type StreamResponse struct {
// 	TimeLen   int    `json:"length,omitempty"`
// 	VideoURL  string `json:"videoURL,omitempty"`
// 	AudioURL  string `json:"audioURL,omitempty"`
// 	TmpDir    string `json:"tmpDir,omitempty"`
// 	OutDir    string `json:"outDir,omitempty"`
// 	OutExt    string `json:"extension,omitempty"`
// 	VideoRate string `json:"videoRate,omitempty"`
// 	Rotate    int    `json:"rotate,omitempty"`
// 	Logfile   string `json:"log,omitempty"`
// }

var std = log.MultiLogger(log.New("ipcam-stream", log.TextFormat))
var Logger log.LoggerI

func New(loggers ...log.LoggerI) (*StreamService, error) {
	service := &StreamService{
		request: &StreamRequest{},
		Log:     Logger,
	}

	// init multilogger
	if len(loggers) == 0 {
		service.Log = std
	} else {
		service.Log = log.MultiLogger(loggers...)
	}

	go func() {
		for {
			msg, ok := <-LogCh
			if ok {
				service.Log.SetPrefix(msg.Prefix).Fields(msg.Metadata).Log(msg.Level, msg.Msg)
			} else {
				service.Log.SetPrefix("ipcam-stream: Logger").Log(log.LLInfo, "logger channel is closed")
				break
			}
		}
	}()

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: New()",
		Level:  log.LLInfo,
		Msg:    "service initialized",
	}

	return service, nil
}

func (s *StreamService) Capture(req *StreamRequest) {

	//TODO: validate input
	s.request = req

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Capture()",
		Level:  log.LLInfo,
		Msg:    "new capture request",
		Metadata: map[string]interface{}{
			"length":    req.TimeLen,
			"videoURL":  req.VideoURL,
			"audioURL":  req.AudioURL,
			"tmpDir":    req.TmpDir,
			"outDir":    req.OutDir,
			"extension": req.OutExt,
			"videoRate": req.VideoRate,
			"rotate":    req.Rotate,
			"log":       req.Logfile,
		},
	}

	// initialize service
	//  - clear cache
	cache := &cache{}

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Capture()",
		Level:  log.LLDebug,
		Msg:    "loading cache",
	}

	err := cache.load(s.request.TmpDir)
	if err != nil {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLFatal,
			Msg:    "failed to load cache",
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		}

	}

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Capture()",
		Level:  log.LLDebug,
		Msg:    "clearing existing cache",
	}

	errList := cache.clear()
	if len(errList) > 0 {
		for _, err := range errList {
			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Capture()",
				Level:  log.LLError,
				Msg:    "failed to clear cache",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}

		}
	}

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Capture()",
		Level:  log.LLDebug,
		Msg:    "cache is ready; starting capture",
	}

	s.newCaptureResponse(s.request)

}

func (s *StreamService) newCaptureResponse(req *StreamRequest) {
	// handle signal: interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(videoRate string) {
		<-c

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLInfo,
			Msg:    "received signal interrupt -- merging cached files",
		}

		s.Stream.Merge(videoRate)

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLInfo,
			Msg:    "merge completed -- exiting",
		}

		os.Exit(0)

	}(req.VideoRate)

	for {
		now := time.Now()

		folderDate := now.Format("2006-01-02")
		fileDate := now.Format("2006-01-02-15-04-05")

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLDebug,
			Msg:    fmt.Sprintf("setting stream timestamp: %s", fileDate),
		}

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLDebug,
			Msg:    fmt.Sprintf("loading output directory: %s", req.OutDir),
		}

		dir := &dir{}
		if err := dir.load(req.OutDir); err != nil {

			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Capture()",
				Level:  log.LLFatal,
				Msg:    "unable to load output directory",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}

		if !dir.exists(folderDate) {
			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Capture()",
				Level:  log.LLDebug,
				Msg:    fmt.Sprintf("creating new output folder: %s", req.OutDir+folderDate),
			}
			dir.mkdir(folderDate)
		}

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLDebug,
			Msg:    fmt.Sprintf("started rotate routine; set to: %v days", req.Rotate),
		}
		go dir.rotate(now, req.Rotate)

		s.Stream = &SplitStream{
			audio:   &Stream{},
			video:   &Stream{},
			outPath: req.OutDir + folderDate + "/" + fileDate + req.OutExt,
		}

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLDebug,
			Msg:    "starting to capture audio/video HTTP stream",
		}

		s.Stream.audio.SetSource(req.AudioURL)
		s.Stream.video.SetSource(req.VideoURL)
		s.Stream.audio.SetOutput(req.TmpDir + "a-" + fileDate + "_temp.mp4")
		s.Stream.video.SetOutput(req.TmpDir + "v-" + fileDate + "_temp.mp4")

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLDebug,
			Msg:    fmt.Sprintf("stream timing out in %v minutes", req.TimeLen),
		}

		s.Stream.SyncTimeout(time.Minute * time.Duration(req.TimeLen))

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Capture()",
			Level:  log.LLDebug,
			Msg:    fmt.Sprintf("merging stream, with %v fps video rate", req.VideoRate),
		}

		go s.Stream.Merge(req.VideoRate)
	}

}
