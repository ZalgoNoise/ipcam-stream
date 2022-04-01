package ipcam

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/zalgonoise/zlog/log"
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

	defer logPanics("SetSource()")

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("SetSource()").Message("connecting to HTTP A/V stream").Metadata(log.Field{"addr": src}).Build()

	n, err := ExpBackoff(time.Second*10, func() error {
		resp, err := http.Get(src)
		if err != nil {
			logCh <- log.NewMessage().Level(log.LLError).Sub("SetSource()").Message("failed to initialize HTTP stream").Metadata(log.Field{
				"error":   err.Error(),
				"service": "Stream.SetSource()",
				"inputs": map[string]interface{}{
					"source": src,
				},
				"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			}).Build()

			return err
		}

		if resp.StatusCode != 200 {
			logCh <- log.NewMessage().Level(log.LLError).Sub("SetSource()").Message("HTTP request returned a non-200 status code").Metadata(log.Field{
				"error":   "HTTP status code is not 200",
				"service": "Stream.SetSource()",
				"inputs": map[string]interface{}{
					"source": src,
				},
				"response": map[string]interface{}{
					"statusCode": resp.StatusCode,
					"status":     resp.Status,
					"dataLength": resp.ContentLength,
					"headers":    resp.Header,
				},
				"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			}).Build()

			return errors.New("HTTP request returned a non-200 status code")
		}

		buf, err := ioutil.ReadAll(io.LimitReader(resp.Body, 128))
		if err != nil {
			logCh <- log.NewMessage().Level(log.LLError).Sub("SetSource()").Message("error reading HTTP request body").Metadata(log.Field{
				"error":   err.Error(),
				"service": "Stream.SetSource()",
				"inputs": map[string]interface{}{
					"source": src,
				},
				"response": map[string]interface{}{
					"statusCode": resp.StatusCode,
					"status":     resp.Status,
					"dataLength": resp.ContentLength,
					"headers":    resp.Header,
				},
				"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			}).Build()

			return err
		}

		if len(buf) == 0 {
			logCh <- log.NewMessage().Level(log.LLError).Sub("SetSource()").Message("HTTP request has an empty body").Metadata(log.Field{
				"error":   err.Error(),
				"service": "Stream.SetSource()",
				"inputs": map[string]interface{}{
					"source": src,
				},
				"response": map[string]interface{}{
					"statusCode": resp.StatusCode,
					"status":     resp.Status,
					"dataLength": resp.ContentLength,
					"headers":    resp.Header,
					"testRead":   len(buf),
				},
				"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			}).Build()

			return errors.New("HTTP request has an empty body")
		}

		logCh <- log.NewMessage().Level(log.LLDebug).Sub("SetSource()").Message("HTTP request seems OK").Metadata(log.Field{
			"inputs": map[string]interface{}{
				"source": src,
			},
			"response": map[string]interface{}{
				"statusCode": resp.StatusCode,
				"status":     resp.Status,
				"dataLength": resp.ContentLength,
				"headers":    resp.Header,
				"testRead":   len(buf),
			},
			"desc": "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
		}).Build()
		s.source = resp.Body
		return nil
	})

	if err != nil {
		logCh <- log.NewMessage().Level(log.LLFatal).Sub("SetSource()").Message("failed to initialize HTTP stream with exponential backoff").Metadata(log.Field{
			"error":   err.Error(),
			"service": "Stream.SetSource()",
			"inputs": map[string]interface{}{
				"source": src,
			},
			"desc":        "initializing HTTP stream from A/V endpoint, with a HTTP GET request",
			"numAttempts": n,
		}).Build()

	}

}

func (s *Stream) SetOutput(out string) {
	logCh <- log.NewMessage().Level(log.LLDebug).Sub("SetOutput()").Message("creating output A/V stream file").Metadata(log.Field{"path": out}).Build()

	output, err := os.Create(out)
	if err != nil {

		logCh <- log.NewMessage().Level(log.LLFatal).Sub("SetOutput()").Message("failed to create cache output file").Metadata(log.Field{
			"error":   err.Error(),
			"service": "Stream.SetOutput()",
			"inputs": map[string]interface{}{
				"target": out,
			},
			"desc": "creating the output file which will contain the A/V stream",
		}).Build()

	}
	s.output = output
	s.outPath = out
}

func (s *Stream) Close() {
	defer logPanics("Close()")
	s.output.Close()
	s.source.Close()
}

func (s *Stream) Copy() {

	defer logPanics("Copy()")

	defer func() {
		logCh <- log.NewMessage().Level(log.LLDebug).Sub("Copy()").Message("closing inputs and outputs").Metadata(log.Field{"path": s.outPath}).Build()

		err := s.output.Close()
		if err != nil {
			logCh <- log.NewMessage().Level(log.LLError).Sub("Copy()").Message("error closing output file").Metadata(log.Field{"path": s.outPath, "error": err.Error()}).Build()
		}

		err = s.source.Close()
		if err != nil {
			logCh <- log.NewMessage().Level(log.LLError).Sub("Copy()").Message("error closing source stream").Metadata(log.Field{"path": s.outPath, "error": err.Error()}).Build()
		}
	}()

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Copy()").Message("copying data stream to file").Metadata(log.Field{"path": s.outPath}).Build()

	n, err := io.Copy(s.output, s.source)
	if err != nil {
		logCh <- log.NewMessage().Level(log.LLError).Sub("Copy()").Message("failed to copy data").Metadata(log.Field{"path": s.outPath, "error": err.Error()}).Build()
	}

	if n == 0 {
		logCh <- log.NewMessage().Level(log.LLError).Sub("Copy()").Message("copy routine points to an empty buffer").Metadata(log.Field{"path": s.outPath, "error": "copied data is of length 0 bytes"}).Build()
	}

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Copy()").Message("copied data successfully").Metadata(log.Field{"path": s.outPath}).Build()
}

func (s *Stream) CopyTimeout(wait time.Duration) {
	defer logPanics("CopyTimeout()")
	go s.Copy()
	time.Sleep(wait)
}

func (s *SplitStream) SyncTimeout(wait time.Duration) {
	defer logPanics("SyncTimeout()")

	go s.audio.Copy()
	go s.video.Copy()

	time.Sleep(wait)

	logCh <- log.NewMessage().Sub("SyncTimeout()").Message("stream deadline reached").Build()
}

func (s *SplitStream) Merge(videoRate string) {
	logCh <- log.NewMessage().Sub("Merge()").Message("initialized merge workflow").Build()

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
		logCh <- log.NewMessage().Level(log.LLError).Sub("Merge()").Message("unable to merge the cached A/V files").Metadata(log.Field{
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
		}).Build()

	}

	logCh <- log.NewMessage().Sub("Merge()").Message("cleaning up cached files").Metadata(log.Field{
		"cache": map[string]interface{}{
			"video": s.video.outPath,
			"audio": s.audio.outPath,
		},
	}).Build()

	if errs := s.Cleanup(); len(errs) > 0 {
		for _, err := range errs {
			logCh <- log.NewMessage().Level(log.LLError).Sub("Merge()").Message("failed to remove cached A/V file").Metadata(log.Field{
				"error":   err.Error(),
				"service": "SplitStream.Merge()",
				"inputs": map[string]interface{}{
					"video": s.video.outPath,
					"audio": s.audio.outPath,
				},
				"desc": "removing cached audio and cached video files after merging",
			}).Build()
		}
	}
}

func (s *SplitStream) Cleanup() []error {
	logCh <- log.NewMessage().Sub("Cleanup()").Message("starting cleanup sequence for cached files").Build()

	var errs []error

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Cleanup()").Message("removing video file").Metadata(log.Field{"path": s.video.outPath}).Build()

	if err := os.Remove(s.video.outPath); err != nil {
		logCh <- log.NewMessage().Level(log.LLWarn).Sub("Cleanup()").Message("failed to remove video file").Metadata(log.Field{"path": s.video.outPath, "error": err.Error()}).Build()

		errs = append(errs, err)
	}

	logCh <- log.NewMessage().Level(log.LLDebug).Sub("Cleanup()").Message("removing audio file").Metadata(log.Field{"path": s.audio.outPath}).Build()

	if err := os.Remove(s.audio.outPath); err != nil {
		logCh <- log.NewMessage().Level(log.LLWarn).Sub("Cleanup()").Message("failed to remove audio file").Metadata(log.Field{"path": s.audio.outPath, "error": err.Error()}).Build()

		errs = append(errs, err)
	}
	return errs
}

func logPanics(service string) {
	if r := recover(); r != nil {
		logCh <- log.NewMessage().Level(log.LLPanic).Sub("Panic()").Message("crashed due to a goroutine error").Metadata(log.Field{
			"error":   r,
			"service": service,
		}).Build()

		panic(r)
	}

}
