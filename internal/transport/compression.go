package transport

import (
	"compress/gzip"
	"net"
	"sync"
)

// CompressedConn wraps net.Conn with gzip compression
type CompressedConn struct {
	net.Conn
	gzipWriter *gzip.Writer
	gzipReader *gzip.Reader
	writeMu    sync.Mutex
	readMu     sync.Mutex
}

// NewCompressedConn creates a new compressed connection wrapper
func NewCompressedConn(conn net.Conn) *CompressedConn {
	return &CompressedConn{
		Conn: conn,
	}
}

// Write compresses data before sending
func (c *CompressedConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	
	// Initialize gzip writer if not already done
	if c.gzipWriter == nil {
		c.gzipWriter = gzip.NewWriter(c.Conn)
	}
	
	n, err := c.gzipWriter.Write(p)
	if err != nil {
		return n, err
	}
	
	// Flush to ensure data is sent
	err = c.gzipWriter.Flush()
	return n, err
}

// Read decompresses data after receiving
func (c *CompressedConn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	
	// Initialize gzip reader if not already done
	if c.gzipReader == nil {
		var err error
		c.gzipReader, err = gzip.NewReader(c.Conn)
		if err != nil {
			return 0, err
		}
	}
	
	return c.gzipReader.Read(p)
}

// Close closes the compressed connection and underlying connection
func (c *CompressedConn) Close() error {
	var err error
	
	// Close gzip writer
	c.writeMu.Lock()
	if c.gzipWriter != nil {
		err = c.gzipWriter.Close()
		c.gzipWriter = nil
	}
	c.writeMu.Unlock()
	
	// Close gzip reader
	c.readMu.Lock()
	if c.gzipReader != nil {
		readerErr := c.gzipReader.Close()
		if err == nil {
			err = readerErr
		}
		c.gzipReader = nil
	}
	c.readMu.Unlock()
	
	// Close underlying connection
	connErr := c.Conn.Close()
	if err == nil {
		err = connErr
	}
	
	return err
}

// EnableCompression wraps a connection with compression
func EnableCompression(conn net.Conn) net.Conn {
	return NewCompressedConn(conn)
}
