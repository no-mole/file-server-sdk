# file-server-sdk
neptune 文件服务sdk，依赖file-server-gateway 和 file-server。其中file-server-gateway 作为文件存储服务的网关层，上传文件时会负载均衡到容量最小的file-server ，同时起CDN的作用， file-server 可以横向扩容。

### upload file by bytes
```go
    client, err := ossClient.NewGrpcOssClient(context.Background(), "endpoint", "")
    if err != nil {
    	return
    }
    client.SetBucket("bucket")
    resp := client.Upload("fileName", "header", bytes.NewBuffer([]byte("")))
```

### upload file by filePath
```go
    client, err := ossClient.NewGrpcOssClient(context.Background(), "endpoint", "")
    if err != nil {
        fmt.Println(err)
        return
    }
    client.SetBucket("bucket")
    resp := client.UploadFromFile("fileName", "header", "filePath")
```

### chunk upload file by bytes
```go
    client, err := ossClient.NewGrpcOssClient(context.Background(), "endpoint", "")
    if err != nil {
        fmt.Println(err)
        return
    }
    client.SetBucket("bucket")
    resp := client.UploadForChunk(chunkSize, "fileName", "header", bytes.NewBuffer([]byte("")))
```

### upload file by filePath
```go
    client, err := ossClient.NewGrpcOssClient(context.Background(), "endpoint", "")
    if err != nil {
        fmt.Println(err)
        return
    }
    client.SetBucket("bucket")
    resp := client.UploadForChunkFromFile(chunkSize, "fileName", "header", "filePath")
```

### download file
```go
    client, err := ossClient.NewGrpcOssClient(context.Background(), "endpoint", "")
    if err != nil {
        fmt.Println(err)
        return
    }
    err := client.Download("fileName", "bucket")
    if err != nil {
        fmt.Println(err)
        return
    }
```

### download file from chunk
```go
    client, err := ossClient.NewGrpcOssClient(context.Background(), "endpoint", "")
    if err != nil {
        fmt.Println(err)
        return
    }
    err := client.DownloadForChunk(1024*1024, "fileName", "bucket")
    if err != nil {
        fmt.Println(err)
        return
    }
```
