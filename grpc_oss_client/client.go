package ossClient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/no-mole/file-server-gateway/enum"
	"github.com/no-mole/file-server-sdk/oss"
	fsPb "github.com/no-mole/file-server/protos/file_server"
	nepEnum "github.com/no-mole/neptune/enum"
	"github.com/no-mole/neptune/utils"
	"google.golang.org/grpc"
)

type GrpcOssClient struct {
	ctx        context.Context
	accessKey  string
	Conn       *grpc.ClientConn
	Client     fsPb.FileServerServiceClient
	Config     *ClientConfig
	BucketName string
}

type ClientOption func(*GrpcOssClient)

func NewGrpcOssClient(ctx context.Context, endpoint, accessKey string, options ...ClientOption) (oss.OssClient, error) {
	if !Authentication(ctx, accessKey) {
		return nil, errors.New("no permission")
	}

	config := getDefaultClientConfig()
	config.endpoint = endpoint

	client := &GrpcOssClient{
		ctx:       ctx,
		accessKey: accessKey,
		Config:    config,
	}

	for _, option := range options {
		option(client)
	}

	conn, err := client.initConn()
	if err != nil {
		return nil, err
	}

	serviceClient := fsPb.NewFileServerServiceClient(conn)
	client.Client = serviceClient
	return client, nil
}

func (client *GrpcOssClient) initConn() (*grpc.ClientConn, error) {
	retryOps := []grpc_retry.CallOption{
		grpc_retry.WithPerRetryTimeout(client.Config.retryTimeout),
		grpc_retry.WithBackoff(grpc_retry.BackoffLinearWithJitter(
			client.Config.retryBackoff.waitBetween,
			client.Config.retryBackoff.jitterFraction)),
	}
	conn, err := grpc.DialContext(client.ctx,
		client.Config.endpoint,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(math.MaxInt32)),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOps...)))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (client *GrpcOssClient) Upload(fileName, bucket, header string, reader io.Reader) oss.Response {
	byte, err := ioutil.ReadAll(reader)
	if err != nil {
		return GetResponseInfo(enum.ErrorFileRead, err)
	}
	return client.singleUpload(fileName, bucket, header, client.accessKey, byte)
}

func (client *GrpcOssClient) UploadFromFile(fileName, bucket, header string, filePath string) oss.Response {
	fd, err := os.Open(filePath)
	if err != nil {
		return GetResponseInfo(enum.ErrorFileOpen, err)
	}
	defer fd.Close()
	byte, err := ioutil.ReadAll(fd)
	if err != nil {
		return GetResponseInfo(enum.ErrorFileRead, err)
	}
	return client.singleUpload(fileName, bucket, header, client.accessKey, byte)
}

func (client *GrpcOssClient) singleUpload(fileName, bucket, header, accessKey string, body []byte) *fsPb.UpLoadResponse {
	resp, err := client.Client.SingleUpload(client.ctx, &fsPb.UploadInfo{
		AccessKey: accessKey,
		Header:    header,
		Bucket:    bucket,
		FileName:  fileName,
		Chunk: &fsPb.Chunk{
			Content: body,
		},
	})
	if err != nil {
		return GetResponseInfo(enum.ErrorSingleUpload, err)
	}
	return resp
}

func (client *GrpcOssClient) UploadForChunk(chunkSize int64, fileName, bucket, header string, reader io.Reader) oss.Response {
	return client.ChunkUpload(chunkSize, fileName, bucket, header, reader)
}

func (client *GrpcOssClient) UploadForChunkFromFile(chunkSize int64, fileName, bucket, header string, filePath string) oss.Response {
	fd, err := os.Open(filePath)
	if err != nil {
		return GetResponseInfo(enum.ErrorFileOpen, err)
	}
	defer fd.Close()
	return client.ChunkUpload(chunkSize, fileName, bucket, header, fd)
}

func (client *GrpcOssClient) ChunkUpload(chunkSize int64, fileName, bucket, header string, reader io.Reader) *fsPb.UpLoadResponse {
	stream, err := client.Client.ChunkUpload(client.ctx)
	if err != nil {
		return GetResponseInfo(enum.ErrorGrpcClient, err)
	}

	r := bufio.NewReader(reader)
	buffer := make([]byte, chunkSize)

	for {
		n, err := r.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return GetResponseInfo(enum.ErrorFileRead, err)
		}
		info := &fsPb.UploadInfo{
			AccessKey: client.accessKey,
			Header:    header,
			Bucket:    bucket,
			FileName:  fileName,
			Chunk: &fsPb.Chunk{
				Content: buffer[:n]},
		}
		if err := stream.Send(info); err != nil {
			return GetResponseInfo(enum.ErrorChunkUpload, err)
		}
	}

	reply, err := stream.CloseAndRecv()
	if err != nil {
		return GetResponseInfo(enum.ErrorChunkUpload, err)
	}
	return reply
}

func (client *GrpcOssClient) Download(fileName, bucket string) error {
	return client.SingleDownload(&fsPb.DownloadInfo{
		FileName: fileName,
		Bucket:   bucket,
	})
}

func (client *GrpcOssClient) DownloadForChunk(chunkSize int64, fileName, bucket string) error {
	download := &fsPb.DownloadInfo{
		Size:     chunkSize,
		FileName: fileName,
		Bucket:   bucket,
	}

	fileBody := make([]byte, 0)
	stream, err := client.Client.BigFileDownload(client.ctx, download)
	if err != nil {
		return err
	}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fileBody = append(fileBody, chunk.Chunk.Content...)
	}

	dirPath := path.Join(utils.GetCurrentAbPath(), bucket)
	if !exists(dirPath) {
		os.MkdirAll(dirPath, os.ModePerm)
	}

	file, err := os.Create(path.Join(dirPath, fileName))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(fileBody)
	if err != nil {
		return err
	}
	return nil
}

func (client *GrpcOssClient) SingleDownload(in *fsPb.DownloadInfo) error {
	resp, err := client.Client.Download(client.ctx, in)
	if err != nil {
		return nil
	}
	dirPath := path.Join(utils.GetCurrentAbPath(), in.Bucket)
	if !exists(dirPath) {
		os.MkdirAll(dirPath, os.ModePerm)
	}

	file, err := os.Create(path.Join(dirPath, in.FileName))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(resp.Chunk.Content)
	if err != nil {
		return err
	}
	return nil
}

func (client *GrpcOssClient) CloseClient() error {
	if client.Conn != nil {
		err := client.Conn.Close()
		return err
	}
	return nil
}

func GetResponseInfo(enum nepEnum.ErrorNum, err ...error) *fsPb.UpLoadResponse {
	message := enum.GetMsg()
	if err != nil {
		message = fmt.Sprintf("%s:%s", enum.GetMsg(), err[0].Error())
	}
	return &fsPb.UpLoadResponse{
		Message: message,
		Code:    int64(enum.GetCode()),
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
