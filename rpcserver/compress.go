package server

import (
	"io"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

type brotliResponseWriter struct {
	io.Writer
	gin.ResponseWriter
}

func (w *brotliResponseWriter) Write(data []byte) (int, error) {
	return w.Writer.Write(data)
}

type zstdResponseWriter struct {
	io.Writer
	gin.ResponseWriter
}

func (w *zstdResponseWriter) Write(data []byte) (int, error) {
	return w.Writer.Write(data)
}

type gzipResponseWriter struct {
	io.Writer
	gin.ResponseWriter
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	return w.Writer.Write(data)
}

// CompressionMiddleware handles gzip, brotli, and zstd compression
func CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "br") {
			c.Writer.Header().Set("Content-Encoding", "br")
			brWriter := brotli.NewWriter(c.Writer)
			defer brWriter.Close()
			c.Writer = &brotliResponseWriter{Writer: brWriter, ResponseWriter: c.Writer}
		} else if strings.Contains(acceptEncoding, "zstd") {
			c.Writer.Header().Set("Content-Encoding", "zstd")
			zstdWriter, _ := zstd.NewWriter(c.Writer)
			defer zstdWriter.Close()
			c.Writer = &zstdResponseWriter{Writer: zstdWriter, ResponseWriter: c.Writer}
		} else if strings.Contains(acceptEncoding, "gzip") {
			c.Writer.Header().Set("Content-Encoding", "gzip")
			gzipWriter := gzip.NewWriter(c.Writer)
			defer gzipWriter.Close()
			c.Writer = &gzipResponseWriter{Writer: gzipWriter, ResponseWriter: c.Writer}
		}

		c.Next()
	}
}
