package oss

import (
	"io"
)

type Response interface {
	GetMessage() string
	GetCode() int64
}

type OssClient interface {
	Upload(fileName, bucket, header string, reader io.Reader) Response
	UploadFromFile(fileName, bucket, header string, filePath string) Response
	UploadForChunk(chunkSize int64, fileName, bucket, header string, reader io.Reader) Response
	UploadForChunkFromFile(chunkSize int64, fileName, bucket, header string, filePath string) Response
	Download(fileName, bucket string) error
	DownloadForChunk(chunkSize int64, fileName, bucket string) error
	CloseClient() error
}
