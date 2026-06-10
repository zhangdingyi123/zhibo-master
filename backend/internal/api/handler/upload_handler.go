package handler

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/response"
	"github.com/zhibo/backend/internal/domain"
)

const maxUploadSize = 5 << 20

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type UploadHandler struct {
	dir string
}

func NewUploadHandler(dir string) *UploadHandler {
	return &UploadHandler{dir: dir}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, domain.ErrUploadFileRequired)
		return
	}
	if file.Size > maxUploadSize {
		response.Fail(c, domain.ErrUploadFileTooLarge)
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Fail(c, domain.ErrUploadFailed)
		return
	}
	defer src.Close()

	buf := make([]byte, 512)
	n, err := io.ReadFull(src, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		response.Fail(c, domain.ErrUploadFailed)
		return
	}

	ext, ok := allowedImageTypes[http.DetectContentType(buf[:n])]
	if !ok {
		ext = strings.ToLower(filepath.Ext(file.Filename))
		if ext == ".jpeg" {
			ext = ".jpg"
		}
		switch ext {
		case ".jpg", ".png", ".gif", ".webp":
		default:
			response.Fail(c, domain.ErrUploadInvalidType)
			return
		}
	}

	name := randomHexName() + ext
	destPath := filepath.Join(h.dir, "products", name)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		response.Fail(c, domain.ErrUploadFailed)
		return
	}

	dest, err := os.Create(destPath)
	if err != nil {
		response.Fail(c, domain.ErrUploadFailed)
		return
	}
	defer dest.Close()

	if n > 0 {
		if _, err := dest.Write(buf[:n]); err != nil {
			response.Fail(c, domain.ErrUploadFailed)
			return
		}
	}
	if _, err := io.Copy(dest, src); err != nil {
		response.Fail(c, domain.ErrUploadFailed)
		return
	}

	response.OK(c, gin.H{"url": "/uploads/products/" + name})
}

func randomHexName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
