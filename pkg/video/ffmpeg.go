package video

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

type ffmpegExtractor struct {
}

// NewExtractor creates a new Extractor that uses ffmpeg as a backend
func NewExtractor() Extractor {
	return &ffmpegExtractor{}
}

func (f *ffmpegExtractor) Clip(ctx context.Context, filename string, start time.Duration, end time.Duration) (io.ReadCloser, error) {
	if _, err := os.Stat(filename); err != nil {
		return nil, err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "ffmpeg-*.mp4")

	if err != nil {
		return nil, err
	}

	log.Println("Created temp file for transcoding:", tmpFile.Name())

	dur := end - start

	cmd := exec.CommandContext(ctx, "ffmpeg", "-noaccurate_seek", "-ss", formatHHMMSS(start), "-i", filename, "-t", formatHHMMSS(dur), "-avoid_negative_ts", "make_zero", "-y", "-c", "copy", tmpFile.Name())
	log.Println("Running command:", cmd)

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	e, err := ioutil.ReadAll(stderr)

	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		log.Println(string(e))
		return nil, err
	}

	log.Printf("File %q finished", tmpFile.Name())

	return &tmpFileAutoCleanup{tmpFile}, nil
}

func formatHHMMSS(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

type tmpFileAutoCleanup struct {
	file *os.File
}

func (f *tmpFileAutoCleanup) Read(p []byte) (n int, err error) {
	return f.file.Read(p)
}

func (f *tmpFileAutoCleanup) Close() error {
	defer func() {
		if err := os.Remove(f.file.Name()); err != nil {
			log.Println("Error deleting file:", err)
		} else {
			log.Println("Deleted", f.file.Name())
		}
	}()
	return f.file.Close()
}
