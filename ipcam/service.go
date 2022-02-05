package ipcam

import (
	"os"
	"os/signal"
	"time"

	"github.com/ZalgoNoise/zlog/log"
)

type StreamService struct {
	request  *StreamRequest
	response *StreamResponse
	Stream   *SplitStream
	Log      log.LoggerI
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
	Logfile   string `json:"log,omitemtpy"`
}

type StreamResponse struct {
	TimeLen   int    `json:"length,omitempty"`
	VideoURL  string `json:"videoURL,omitempty"`
	AudioURL  string `json:"audioURL,omitempty"`
	TmpDir    string `json:"tmpDir,omitempty"`
	OutDir    string `json:"outDir,omitempty"`
	OutExt    string `json:"extension,omitempty"`
	VideoRate string `json:"videoRate,omitempty"`
	Rotate    int    `json:"rotate,omitempty"`
	Logfile   string `json:"log,omitemtpy"`
}

var std = log.New("ipcam-stream", &log.TextFmt{})

func New(loggers ...log.LoggerI) (*StreamService, error) {
	// init multilogger
	if len(loggers) == 0 {
		loggers = []log.LoggerI{std}
	}

	logger := log.MultiLogger(loggers...)

	logger.SetPrefix("ipcam-stream: New()").Info("service initialized")

	return &StreamService{
		request: &StreamRequest{},
		Log:     logger,
	}, nil
}

func (s *StreamService) Capture(req *StreamRequest) {
	s.Log.SetPrefix("ipcam-stream: Capture()")

	//TODO: validate input
	s.request = req

	s.Log.Fields(
		map[string]interface{}{
			"length":    req.TimeLen,
			"videoURL":  req.VideoURL,
			"audioURL":  req.AudioURL,
			"tmpDir":    req.TmpDir,
			"outDir":    req.OutDir,
			"extension": req.OutExt,
			"videoRate": req.VideoRate,
			"rotate":    req.Rotate,
		},
	).Info("new capture request")

	// initialize service
	//  - clear cache
	cache := &cache{}

	s.Log.Debug("loading cache")
	err := cache.load(s.request.TmpDir)
	if err != nil {
		s.Log.Fatalf("failed to load cache: %s\n", err)
	}

	s.Log.Debug("clearing existing cache")
	errList := cache.clear()
	if len(errList) > 0 {
		for _, err := range errList {
			s.Log.Errorf("failed to clear cache: %s\n", err)
		}
	}

	s.Log.Debug("cache is ready; starting capture")
	s.newCaptureResponse(s.request)

}

func (s *StreamService) newCaptureResponse(req *StreamRequest) {
	s.Log.SetPrefix("ipcam-stream: Capture()")

	// handle signal: interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(videoRate string) {
		<-c
		s.Log.Info("received signal interrupt -- merging cached files")

		s.Stream.Merge(videoRate)

		s.Log.Info("merge completed -- exiting")
		os.Exit(0)

	}(req.VideoRate)

	for {
		now := time.Now()

		folderDate := now.Format("2006-01-02")
		fileDate := now.Format("2006-01-02-15-04-05")

		s.Log.Debugf("setting stream timestamp: %s\n", fileDate)

		s.Log.Debugf("loading output directory: %s\n", req.OutDir)
		dir := &dir{}
		if err := dir.load(req.OutDir); err != nil {
			s.Log.Fatalf("unable to load output directory: %s\n", err)
		}

		if !dir.exists(folderDate) {
			s.Log.Debugf("creating new output folder: %s\n", req.OutDir+folderDate)
			dir.mkdir(folderDate)
		}

		s.Log.Debugf("started rotate routine; set to: %v days\n", req.Rotate)
		go dir.rotate(now, req.Rotate)

		s.Stream = &SplitStream{
			audio: &Stream{
				logger: s.Log,
			},
			video: &Stream{
				logger: s.Log,
			},
			outPath: req.OutDir + folderDate + "/" + fileDate + req.OutExt,
			logger:  s.Log,
		}

		s.Log.Debug("starting to capture audio/video HTTP stream")
		go s.Stream.video.SetSource(req.VideoURL)
		s.Stream.audio.SetSource(req.AudioURL)

		s.Stream.audio.SetOutput(req.TmpDir + "a-" + fileDate + "_temp.mp4")
		s.Stream.video.SetOutput(req.TmpDir + "v-" + fileDate + "_temp.mp4")

		s.Log.Debugf("stream timing out in %v minutes\n", req.TimeLen)
		s.Stream.SyncTimeout(time.Minute * time.Duration(req.TimeLen))

		s.Log.Debugf("merging stream, with %v fps video rate\n", req.VideoRate)
		go s.Stream.Merge(req.VideoRate)
	}

}
