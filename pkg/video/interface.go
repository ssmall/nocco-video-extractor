// Package video provides an interface for manipulating video
package video

import (
	"context"
	"io"
	"time"
)

// Extractor extracts a clip from a video source
type Extractor interface {
	// Clip extracts a clip from the given video (or audio) file between the given start time and end time (inclusive)
	Clip(ctx context.Context, filename string, start time.Duration, end time.Duration) (io.ReadCloser, error)
}
