package ioutil

import "io"

var defaultBufferBytes = 128 * 1024

// PageWriter implement the io.Writer interface so that
// writes will either be in page chunks or from flushing.
type PageWriter struct {
	w io.Writer
	// pageOffset tracks the offset of the base of the buffer
	pageOffset int
	// pageBytes is the number of bytes per page
	pageBytes int
	// bufferedBytes counts the number of bytes pending for write in the buffer
	bufferedBytes int
	// buf holds the buffer
	buf []byte
	// bufWatermarkBytes is the number of bytes the buf can hold before it
	// needs to be flushed. It is less than len(buf) so there is space
	// for slack writes to bring the writer page alignment.
	bufWatermarkBytes int
}

func NewPageWriter(w io.Writer, pageBytes, pageOffset int) *PageWriter {
	pw := &PageWriter{
		w:                 w,
		pageOffset:        pageOffset,
		pageBytes:         pageBytes,
		buf:               make([]byte, defaultBufferBytes+pageBytes),
		bufWatermarkBytes: defaultBufferBytes,
	}
	return pw
}

func (pw *PageWriter) Write(p []byte) (n int, err error) {
	if len(p)+pw.bufferedBytes <= pw.bufWatermarkBytes {
		copy(pw.buf[pw.bufferedBytes:], p)
		pw.bufferedBytes += len(p)
		return len(p), nil
	}
	slack := pw.pageBytes - ((pw.pageOffset + pw.bufferedBytes) % pw.pageBytes)
	if slack != pw.pageBytes {
		partial := slack > len(p)
		if partial {
			// no enough data to complete the slack page
			slack = len(p)
		}
		copy(pw.buf[pw.bufferedBytes:], p[:slack])
		pw.bufferedBytes += slack
		n = slack
		if partial {
			return n, nil
		}
		p = p[slack:]
	}

	// flush buf
	if err = pw.Flush(); err != nil {
		return n, err
	}

	if len(p) > pw.pageBytes {
		// directly wirte others alignment
		pages := len(p) / pw.pageBytes
		c, werr := pw.w.Write(p[:pages*pw.pageBytes])
		n += c
		if werr != nil {
			return n, werr
		}
		p = p[pages*pw.pageBytes:]
	}
	// write remaining tail to buf
	c, werr := pw.Write(p)
	n += c
	return n, werr
}

func (pw *PageWriter) Flush() error {
	if pw.bufferedBytes == 0 {
		return nil
	}
	_, err := pw.w.Write(pw.buf[:pw.bufferedBytes])
	pw.pageOffset = (pw.pageOffset + pw.bufferedBytes) % pw.pageBytes
	pw.bufferedBytes = 0
	return err
}
