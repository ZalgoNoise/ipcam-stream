package ipcam

import (
	"io"
	"net/http"
	"os"
	"time"

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
	resp, err := http.Get(src)
	if err != nil {
		panic(err)
	}
	s.source = resp.Body
}

func (s *Stream) SetOutput(out string) {
	output, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	s.output = output
	s.outPath = out
}

func (s *Stream) Close() {
	s.output.Close()
	s.source.Close()
}

func (s *Stream) Copy() {
	io.Copy(s.output, s.source)
}

func (s *Stream) CopyTimeout(wait time.Duration) {
	go s.Copy()
	time.Sleep(wait)
	s.Close()
}

func (s *SplitStream) SyncTimeout(wait time.Duration) {
	go io.Copy(s.audio.output, s.audio.source)
	go io.Copy(s.video.output, s.video.source)
	time.Sleep(wait)
	s.audio.Close()
	s.video.Close()
}

func (s *SplitStream) Merge(videoRate string) {
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
		panic(err)
	}

	if err := s.Cleanup(); err != nil {
		panic(err)
	}

}

func (s *SplitStream) Cleanup() error {
	if err := os.Remove(s.video.outPath); err != nil {
		return err
	}
	if err := os.Remove(s.audio.outPath); err != nil {
		return err
	}
	return nil
}
