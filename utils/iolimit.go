package utils

import (
	"errors"
	"fmt"
	"io"
)

var ErrIOLimitReached = fmt.Errorf("read size limit reached")

func ReadAllLimit(r io.Reader, n int) ([]byte, error) {
	limit := int(n + 1)
	buf, err := io.ReadAll(io.LimitReader(r, int64(limit)))
	if err != nil {
		return buf, err
	}
	if len(buf) >= limit {
		return buf[:limit-1], ErrIOLimitReached
	}
	return buf, nil
}

// CopyLimit copies up to `limit+1`, if it copies more than `limit`, it returns ErrIOLimitReached
func CopyLimit(dst io.Writer, src io.Reader, limit int64) (written int64, err error) {
	n, err := io.CopyN(dst, src, limit+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("copying: %w", err)
	}

	if n > limit {
		return n, ErrIOLimitReached
	}

	return n, nil
}
