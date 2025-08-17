package stream

import (
	"compress/gzip"
	"io"
)

// CompressedReader wraps a reader with gzip decompression
type CompressedReader struct {
	underlying io.Reader
	gzipReader *gzip.Reader
}

// NewCompressedReader creates a new compressed reader
func NewCompressedReader(r io.Reader) (*CompressedReader, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	
	return &CompressedReader{
		underlying: r,
		gzipReader: gr,
	}, nil
}

// Read implements io.Reader interface with decompression
func (cr *CompressedReader) Read(p []byte) (int, error) {
	return cr.gzipReader.Read(p)
}

// Close closes the gzip reader
func (cr *CompressedReader) Close() error {
	if cr.gzipReader != nil {
		return cr.gzipReader.Close()
	}
	return nil
}

// CompressedWriter wraps a writer with gzip compression
type CompressedWriter struct {
	underlying io.Writer
	gzipWriter *gzip.Writer
}

// NewCompressedWriter creates a new compressed writer
func NewCompressedWriter(w io.Writer) *CompressedWriter {
	gw := gzip.NewWriter(w)
	
	return &CompressedWriter{
		underlying: w,
		gzipWriter: gw,
	}
}

// Write implements io.Writer interface with compression
func (cw *CompressedWriter) Write(p []byte) (int, error) {
	return cw.gzipWriter.Write(p)
}

// Flush flushes the gzip writer
func (cw *CompressedWriter) Flush() error {
	return cw.gzipWriter.Flush()
}

// Close closes the gzip writer
func (cw *CompressedWriter) Close() error {
	if cw.gzipWriter != nil {
		return cw.gzipWriter.Close()
	}
	return nil
}
