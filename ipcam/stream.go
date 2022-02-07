package ipcam

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ZalgoNoise/zlog/log"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Stream struct {
	source  io.ReadCloser
	output  *os.File
	outPath string
	logger  log.LoggerI
}

type SplitStream struct {
	audio   *Stream
	video   *Stream
	outPath string
	logger  log.LoggerI
}

func (s *Stream) SetSource(src string) {
	s.logger.Debugf("connecting to HTTP A/V stream on %s", src)

	resp, err := http.Get(src)
	if err != nil {
		s.logger.SetPrefix("ipcam-stream: Stream.SetSource()").Fields(
			map[string]interface{}{
				"error":   err.Error(),
				"service": "Stream.SetSource()",
				"inputs": map[string]interface{}{
					"source": src,
				},
				"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			},
		).Fatalf("failed to initialize stream on URL %s with error: %s", src, err)
	}
	s.source = resp.Body
}

func (s *Stream) SetOutput(out string) {
	s.logger.Debugf("creating output A/V stream file on %s", out)

	output, err := os.Create(out)
	if err != nil {
		s.logger.SetPrefix("ipcam-stream: Stream.SetOutput()").Fields(
			map[string]interface{}{
				"error":   err.Error(),
				"service": "Stream.SetOutput()",
				"inputs": map[string]interface{}{
					"target": out,
				},
				"desc": "creating the output file which will contain the A/V stream",
			},
		).Fatalf("failed to initialize stream on URL %s with error: %s", out, err)
	}
	s.output = output
	s.outPath = out
}

func (s *Stream) Close() {
	defer logPanics(s.logger)
	s.output.Close()
	s.source.Close()
}

func (s *Stream) Copy() {
	defer logPanics(s.logger)
	io.Copy(s.output, s.source)
}

func (s *Stream) CopyTimeout(wait time.Duration) {
	defer logPanics(s.logger)
	defer s.Close()
	go s.Copy()
	time.Sleep(wait)
}

func (s *SplitStream) SyncTimeout(wait time.Duration) {
	defer logPanics(s.logger)
	defer s.video.Close()
	defer s.audio.Close()
	go io.Copy(s.audio.output, s.audio.source)
	go io.Copy(s.video.output, s.video.source)
	time.Sleep(wait)
}

func (s *SplitStream) Merge(videoRate string) {
	s.logger.SetPrefix("ipcam-stream: Merge()").Info("initialized merge workflow")

	err := ffmpeg.Output(
		[]*ffmpeg.Stream{
			ffmpeg.Input(
				s.video.outPath,
				ffmpeg.KwArgs{"vsync": "1"},
				ffmpeg.KwArgs{"r": videoRate},
			),
			ffmpeg.Input(s.audio.outPath),
		},
		s.outPath,
		ffmpeg.KwArgs{"input_format": "1"},
		ffmpeg.KwArgs{"b:v": "4000k"},
		ffmpeg.KwArgs{"c:v": "libx264"},
		ffmpeg.KwArgs{"c:a": "aac"},
		ffmpeg.KwArgs{"pix_fmt": "yuv420p"},
	).OverWriteOutput().ErrorToStdOut().Run()
	if err != nil {
		s.logger.Fields(map[string]interface{}{
			"error":   err.Error(),
			"service": "SplitStream.Merge()",
			"inputs": map[string]interface{}{
				"video": s.video.outPath,
				"audio": s.audio.outPath,
			},
			"desc": "merging cached audio and cached video into one file, using libx264",
			"proc": map[string]interface{}{
				"input": map[string]interface{}{
					"video": map[string]interface{}{
						"vsync": "1",
						"r":     videoRate,
					},
				},
				"output": map[string]interface{}{
					"input_format": "1",
					"b:v":          "4000k",
					"c:v":          "libx264",
					"c:a":          "aac",
					"pix_fmt":      "yuv420p",
				},
			},
		}).Error("unable to merge the cached A/V files")
	}

	s.logger.Fields(map[string]interface{}{
		"cache": map[string]interface{}{
			"video": s.video.outPath,
			"audio": s.audio.outPath,
		},
	}).Debug("cleaning up cached files")

	if errs := s.Cleanup(); len(errs) > 0 {
		for _, err := range errs {
			s.logger.SetPrefix("ipcam-stream: Merge()").Fields(
				map[string]interface{}{
					"error":   err.Error(),
					"service": "SplitStream.Merge()",
					"inputs": map[string]interface{}{
						"video": s.video.outPath,
						"audio": s.audio.outPath,
					},
					"desc": "removing cached audio and cached video files after merging",
				},
			).Error("failed to remove cached A/V file")
		}
	}

}

func (s *SplitStream) Cleanup() []error {
	s.logger.SetPrefix("ipcam-stream: Cleanup()").Info("starting cleanup sequence for cached files")

	var errs []error

	s.logger.Debugf("removing video file: %s", s.video.outPath)
	if err := os.Remove(s.video.outPath); err != nil {
		s.logger.Warnf("failed to remove video file %s with an error: %s", s.video.outPath, err)

		errs = append(errs, err)
	}

	s.logger.Debugf("removing audio file: %s", s.video.outPath)
	if err := os.Remove(s.audio.outPath); err != nil {
		s.logger.Warnf("failed to remove audio file %s with an error: %s", s.audio.outPath, err)

		errs = append(errs, err)
	}
	return errs
}

func logPanics(logger log.LoggerI) {
	if r := recover(); r != nil {
		logger.SetPrefix("ipcam-stream").Fields(
			map[string]interface{}{
				"error":   r,
				"service": "goroutine error",
			},
		).Fatal("crashed due to a goroutine error")
	}
}
