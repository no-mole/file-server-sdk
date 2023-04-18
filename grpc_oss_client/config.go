package ossClient

import (
	"context"
	"errors"
	"math"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/no-mole/file-server-sdk/oss"
	fsPb "github.com/no-mole/file-server/protos/file_server"
	"google.golang.org/grpc"
)

type ClientConfig struct {
	endpoint     string
	retryTimeout time.Duration
	timeout      time.Duration
	retryBackoff RetryBackoff
}

type RetryBackoff struct {
	waitBetween    time.Duration
	jitterFraction float64
}

func getDefaultClientConfig() *ClientConfig {
	config := new(ClientConfig)
	config.retryTimeout = time.Second * 2
	config.timeout = time.Second * 60
	config.retryBackoff.waitBetween = time.Second
	config.retryBackoff.jitterFraction = 0.2
	return config
}

func NewGrpcOssClients(ctx context.Context, endpoint, accessKey string, options ...ClientOption) (oss.OssClient, error) {
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

func (client *GrpcOssClient) initConnS() (*grpc.ClientConn, error) {
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
