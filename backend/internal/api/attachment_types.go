package api

import (
	"path/filepath"
	"strings"
)

var attachmentMIMETypesByExtension = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".pdf":  "application/pdf",
	".txt":  "text/plain",
	".md":   "text/markdown",
	".csv":  "text/csv",
	".json": "application/json",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

func mimeFromFilename(filename string) string {
	extension := strings.ToLower(filepath.Ext(filename))
	if mimeType, found := attachmentMIMETypesByExtension[extension]; found {
		return mimeType
	}
	return "application/octet-stream"
}
