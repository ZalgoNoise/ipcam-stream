package ipcam

import (
	"fmt"
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
}

type SplitStream struct {
	audio   *Stream
	video   *Stream
	outPath string
}

func (s *Stream) SetSource(src string) {

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: SetSource()",
		Level:  log.LLDebug,
		Msg:    fmt.Sprintf("connecting to HTTP A/V stream on %s", src),
	}

	resp, err := http.Get(src)
	if err != nil {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: SetSource()",
			Level:  log.LLFatal,
			Msg:    "failed to initialize HTTP stream",
			Metadata: map[string]interface{}{
				"error":   err.Error(),
				"service": "Stream.SetSource()",
				"inputs": map[string]interface{}{
					"source": src,
				},
				"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			},
		}
	}
	s.source = resp.Body
}

func (s *Stream) SetOutput(out string) {
	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: SetOutput()",
		Level:  log.LLDebug,
		Msg:    fmt.Sprintf("creating output A/V stream file on %s", out),
	}

	output, err := os.Create(out)
	if err != nil {

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: SetOutput()",
			Level:  log.LLFatal,
			Msg:    "failed to create cache output file",
			Metadata: map[string]interface{}{
				"error":   err.Error(),
				"service": "Stream.SetOutput()",
				"inputs": map[string]interface{}{
					"target": out,
				},
				"desc": "creating the output file which will contain the A/V stream",
			},
		}

	}
	s.output = output
	s.outPath = out
}

func (s *Stream) Close() {
	defer logPanics()
	s.output.Close()
	s.source.Close()
}

func (s *Stream) Copy() {
	defer func() {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Copy()",
			Level:  log.LLDebug,
			Msg:    "closing inputs and outputs",
			Metadata: map[string]interface{}{
				"path": s.outPath,
			},
		}

		err := s.output.Close()
		if err != nil {

			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Copy()",
				Level:  log.LLError,
				Msg:    "error closing output file",
				Metadata: map[string]interface{}{
					"path": s.outPath,
				},
			}
		}
		err = s.source.Close()
		if err != nil {

			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Copy()",
				Level:  log.LLError,
				Msg:    "error closing source stream",
				Metadata: map[string]interface{}{
					"path": s.outPath,
				},
			}
		}
	}()

	defer logPanics()

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Copy()",
		Level:  log.LLDebug,
		Msg:    "copying data stream to file",
		Metadata: map[string]interface{}{
			"path": s.outPath,
		},
	}

	n, err := io.Copy(s.output, s.source)
	if err != nil {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Copy()",
			Level:  log.LLError,
			Msg:    "failed to copy data",
			Metadata: map[string]interface{}{
				"error":      err.Error(),
				"path":       s.outPath,
				"copiedBits": n,
			},
		}
	} else {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Copy()",
			Level:  log.LLDebug,
			Msg:    "copied data successfully",
			Metadata: map[string]interface{}{
				"path":       s.outPath,
				"copiedBits": n,
			},
		}
	}
}

func (s *Stream) CopyTimeout(wait time.Duration) {
	defer logPanics()
	go s.Copy()
	time.Sleep(wait)
}

func (s *SplitStream) SyncTimeout(wait time.Duration) {
	defer logPanics()

	go s.audio.Copy()
	go s.video.Copy()

	time.Sleep(wait)

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: SyncTimeout()",
		Level:  log.LLInfo,
		Msg:    "stream timed out",
	}
}

func (s *SplitStream) Merge(videoRate string) {
	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Merge()",
		Level:  log.LLInfo,
		Msg:    "initialized merge workflow",
	}

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
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Merge()",
			Level:  log.LLError,
			Msg:    "unable to merge the cached A/V files",
			Metadata: map[string]interface{}{
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
			},
		}
	}

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Merge()",
		Level:  log.LLInfo,
		Msg:    "cleaning up cached files",
		Metadata: map[string]interface{}{
			"cache": map[string]interface{}{
				"video": s.video.outPath,
				"audio": s.audio.outPath,
			},
		},
	}

	if errs := s.Cleanup(); len(errs) > 0 {
		for _, err := range errs {
			LogCh <- log.ChLogMessage{
				Prefix: "ipcam-stream: Merge()",
				Level:  log.LLError,
				Msg:    "failed to remove cached A/V file",
				Metadata: map[string]interface{}{
					"error":   err.Error(),
					"service": "SplitStream.Merge()",
					"inputs": map[string]interface{}{
						"video": s.video.outPath,
						"audio": s.audio.outPath,
					},
					"desc": "removing cached audio and cached video files after merging",
				},
			}
		}
	}
}

func (s *SplitStream) Cleanup() []error {
	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Cleanup()",
		Level:  log.LLInfo,
		Msg:    "starting cleanup sequence for cached files",
	}

	var errs []error

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Cleanup()",
		Level:  log.LLDebug,
		Msg:    fmt.Sprintf("removing video file: %s", s.video.outPath),
	}

	if err := os.Remove(s.video.outPath); err != nil {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Cleanup()",
			Level:  log.LLWarn,
			Msg:    "failed to remove video file",
			Metadata: map[string]interface{}{
				"error": err.Error(),
				"path":  s.video.outPath,
			},
		}

		errs = append(errs, err)
	}

	LogCh <- log.ChLogMessage{
		Prefix: "ipcam-stream: Cleanup()",
		Level:  log.LLDebug,
		Msg:    fmt.Sprintf("removing audio file: %s", s.audio.outPath),
	}

	if err := os.Remove(s.audio.outPath); err != nil {

		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream: Cleanup()",
			Level:  log.LLWarn,
			Msg:    "failed to remove audio file",
			Metadata: map[string]interface{}{
				"error": err.Error(),
				"path":  s.audio.outPath,
			},
		}

		errs = append(errs, err)
	}
	return errs
}

func logPanics() {
	if r := recover(); r != nil {
		LogCh <- log.ChLogMessage{
			Prefix: "ipcam-stream",
			Level:  log.LLFatal,
			Msg:    "crashed due to a goroutine error",
			Metadata: map[string]interface{}{
				"error":   r,
				"service": "goroutine error",
			},
		}
	}

}
