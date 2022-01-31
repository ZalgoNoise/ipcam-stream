package ipcam

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

type StreamService struct {
	request  *StreamRequest
	response *StreamResponse
	Stream   *SplitStream
}

type StreamRequest struct {
	TimeLen   int    `json:"length,omitempty"`
	VideoURL  string `json:"videoURL,omitempty"`
	AudioURL  string `json:"audioURL,omitempty"`
	TmpDir    string `json:"tmpDir,omitempty"`
	OutDir    string `json:"outDir,omitempty"`
	OutExt    string `json:"extension,omitempty"`
	VideoRate string `json:"videoRate,omitempty"`
}

type StreamResponse struct {
	TimeLen   int    `json:"length,omitempty"`
	VideoURL  string `json:"videoURL,omitempty"`
	AudioURL  string `json:"audioURL,omitempty"`
	TmpDir    string `json:"tmpDir,omitempty"`
	OutDir    string `json:"outDir,omitempty"`
	OutExt    string `json:"extension,omitempty"`
	VideoRate string `json:"videoRate,omitempty"`
}

func New() *StreamService {
	return &StreamService{
		request: &StreamRequest{},
	}
}

func (s *StreamService) Capture(req *StreamRequest) error {
	//TODO: validate input
	s.request = req

	// initialize service
	//  - clear cache
	cache := &cache{}

	err := cache.load(s.request.TmpDir)
	if err != nil {
		return err
	}
	errList := cache.clear()
	if len(errList) > 0 {
		fmt.Printf("[ipcam-stream] [cache] ERR: %s\n", err)
	}

	s.newCaptureResponse(s.request)
	return nil
}

func (s *StreamService) newCaptureResponse(req *StreamRequest) {
	// handle signal: interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(videoRate string) {
		<-c
		s.Stream.Merge(videoRate)
		os.Exit(0)
	}(req.VideoRate)

	for {
		now := time.Now().Format("2006_02_01-15_04_05")
		s.Stream = &SplitStream{
			audio:   &Stream{},
			video:   &Stream{},
			outPath: req.OutDir + now + req.OutExt,
		}

		go s.Stream.video.SetSource(req.VideoURL)
		s.Stream.audio.SetSource(req.AudioURL)

		s.Stream.audio.SetOutput(req.TmpDir + "a-" + now + "_temp.mp4")
		s.Stream.video.SetOutput(req.TmpDir + "v-" + now + "_temp.mp4")

		s.Stream.SyncTimeout(time.Minute * time.Duration(req.TimeLen))

		go s.Stream.Merge(req.VideoRate)
	}

}