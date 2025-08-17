package stream

import (
	"compress/gzip"
	"fmt"
	"io"
	"net"
)

// StreamOptions holds configuration for stream processing
type StreamOptions struct {
	EnableCompression bool
	BufferSize        int
}

// DefaultStreamOptions returns default stream options
func DefaultStreamOptions() *StreamOptions {
	return &StreamOptions{
		EnableCompression: false,
		BufferSize:        32 * 1024, // 32KB buffer
	}
}

// StreamProcessor handles data streaming with optional compression
type StreamProcessor struct {
	options *StreamOptions
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(options *StreamOptions) *StreamProcessor {
	if options == nil {
		options = DefaultStreamOptions()
	}
	return &StreamProcessor{
		options: options,
	}
}

// CopyWithCompression copies data between connections with optional compression
func (sp *StreamProcessor) CopyWithCompression(dst io.Writer, src io.Reader) (int64, error) {
	// For now, disable compression as it interferes with protocol compatibility
	// TODO: Implement intelligent compression that detects data type
	return io.Copy(dst, src)
}

// copyCompressed performs compressed data copy
func (sp *StreamProcessor) copyCompressed(dst io.Writer, src io.Reader) (int64, error) {
	// Create gzip writer
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()

	// Copy with compression
	written, err := io.Copy(gzWriter, src)
	if err != nil {
		return written, fmt.Errorf("compressed copy error: %w", err)
	}

	// Ensure all data is flushed
	if err := gzWriter.Flush(); err != nil {
		return written, fmt.Errorf("compression flush error: %w", err)
	}

	return written, nil
}

// CreateCompressedReader creates a reader that decompresses data
func (sp *StreamProcessor) CreateCompressedReader(r io.Reader) (io.ReadCloser, error) {
	if !sp.options.EnableCompression {
		return io.NopCloser(r), nil
	}

	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	return gzReader, nil
}

// CreateCompressedWriter creates a writer that compresses data
func (sp *StreamProcessor) CreateCompressedWriter(w io.Writer) io.WriteCloser {
	if !sp.options.EnableCompression {
		return &nopWriteCloser{w}
	}

	return gzip.NewWriter(w)
}

// ProxyConnection proxies data between two connections with optional compression
func (sp *StreamProcessor) ProxyConnection(conn1, conn2 net.Conn) error {
	errCh := make(chan error, 2)

	// Copy conn1 -> conn2
	go func() {
		_, err := sp.CopyWithCompression(conn2, conn1)
		errCh <- err
	}()

	// Copy conn2 -> conn1
	go func() {
		_, err := sp.CopyWithCompression(conn1, conn2)
		errCh <- err
	}()

	// Wait for first error or completion
	return <-errCh
}

// nopWriteCloser wraps an io.Writer to add a no-op Close method
type nopWriteCloser struct {
	io.Writer
}

func (nwc *nopWriteCloser) Close() error {
	return nil
}

// BufferedCopyWithCompression copies data with buffering and optional compression
func (sp *StreamProcessor) BufferedCopyWithCompression(dst io.Writer, src io.Reader) (int64, error) {
	if !sp.options.EnableCompression {
		return sp.bufferedCopy(dst, src)
	}

	return sp.bufferedCopyCompressed(dst, src)
}

// bufferedCopy performs buffered copy without compression
func (sp *StreamProcessor) bufferedCopy(dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, sp.options.BufferSize)
	return io.CopyBuffer(dst, src, buf)
}

// bufferedCopyCompressed performs buffered copy with compression
func (sp *StreamProcessor) bufferedCopyCompressed(dst io.Writer, src io.Reader) (int64, error) {
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()

	buf := make([]byte, sp.options.BufferSize)
	written, err := io.CopyBuffer(gzWriter, src, buf)
	if err != nil {
		return written, fmt.Errorf("buffered compressed copy error: %w", err)
	}

	if err := gzWriter.Flush(); err != nil {
		return written, fmt.Errorf("compression flush error: %w", err)
	}

	return written, nil
}
