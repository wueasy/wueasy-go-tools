# file-client

`file-client` 是专为 `wueasy-file-server` 打造的高性能 Go 语言客户端，支持 Nacos 服务发现、微服务间调用以及直接 HTTP 访问。

## 🌟 特性

*   **微服务支持**：开箱即用支持基于 Nacos 注册中心的服务寻址。
*   **连接池优化**：内置 `http.Client` 并全局复用底层 TCP 连接，提高并发性能。
*   **大文件友好**：利用 `io.Pipe()` 和流式读取，普通上传、分片上传和下载均能做到极低的内存占用。
*   **完整 API 覆盖**：全面支持文件服务器的所有接口，包括基本上传、下载、删除，以及进阶的分片上传、断点下载、批量删除。
*   **规范日志**：通过传入 `context.Context` 以统一集成项目的日志上下文链路。

---

## 🚀 快速开始

### 初始化客户端

```go
import "github.com/wueasy/wueasy-go-tools/file-client"

// 方式一：使用 Nacos 服务发现（默认）
// 第一个参数是服务名，第二个参数是服务分组（留空默认 DEFAULT_GROUP）
client := fileClient.NewFileClient("file-server", "DEFAULT_GROUP")

// 方式二：直连模式（方便本地测试或不使用 Nacos 的场景）
client := fileClient.NewFileClient("file-server", "").SetBaseUrl("http://127.0.0.1:9830")

// 可以自定义超时时间（默认 30 秒）
client.SetTimeout(60 * time.Second)
```

### 1. 基础操作

**📤 上传文件**

```go
ctx := context.Background()
businessType := "document" // 业务类型，如果没有可传 ""

// 1. 上传本地文件（推荐，自动处理流）
resp, err := client.UploadLocalFile(ctx, businessType, "/path/to/local.pdf")

// 2. 上传字节数据
fileBytes := []byte("Hello, Wueasy!")
resp, err := client.UploadBytes(ctx, businessType, fileBytes, "hello.txt")

// 3. 使用任意 io.Reader 上传
fileReader, _ := os.Open("/path/to/local.pdf")
resp, err := client.Upload(ctx, businessType, fileReader, "local.pdf")

fmt.Println("上传成功，文件路径：", resp.Data.FilePath)
```

**📥 下载文件**

```go
// 1. 直接获取全部文件内容（适用于小文件）
fileData, err := client.Download(ctx, businessType, "2024/08/25/hello.txt")

// 2. 获取文件流（适用于大文件，避免内存溢出）
readCloser, err := client.DownloadStream(ctx, businessType, "2024/08/25/large.mp4")
if err == nil {
    defer readCloser.Close()
    // io.Copy(...) 将流保存到本地
}
```

**🗑️ 删除文件**

```go
// 单个文件删除
delResp, err := client.Delete(ctx, businessType, "2024/08/25/hello.txt")

// 批量删除
batchResp, err := client.BatchDelete(ctx, businessType, []string{
    "2024/08/25/file1.txt",
    "2024/08/25/file2.txt",
})
fmt.Printf("成功删除 %d 个，失败 %d 个\n", batchResp.Data.SuccessCount, batchResp.Data.FailedCount)
```

---

## 📦 大文件分片操作

适用于 >100MB 的大文件或需要显示进度、断点续传的场景。

### 分片上传流程

```go
fileSize := int64(104857600) // 假设文件 100MB
fileName := "large-video.mp4"

// 1. 初始化分片上传，获取 FileId 和分片配置
initResp, _ := client.InitChunkUpload(ctx, "video", fileName, fileSize)
fileId := initResp.Data.FileId
chunkSize := initResp.Data.ChunkSize
totalChunks := initResp.Data.TotalChunks

// 2. 上传所有分片（支持并发）
file, _ := os.Open("/path/to/large-video.mp4")
defer file.Close()

for i := 0; i < totalChunks; i++ {
    // 根据 chunkSize 计算并读取当前分片的数据...
    chunkData := make([]byte, chunkSize) // 实际读取长度需判断边界
    
    // 上传单个分片
    _, err := client.UploadChunkBytes(ctx, "video", fileId, i, chunkData, fileName)
}

// 3. 合并分片
mergeResp, _ := client.MergeChunks(ctx, "video", fileId)
fmt.Println("分片上传完成，最终路径：", mergeResp.Data.FilePath)
```

### 分片下载流程

```go
filePath := "2024/08/25/large-video.mp4"

// 1. 获取文件信息和推荐分片大小
infoResp, _ := client.GetChunkDownloadInfo(ctx, "video", filePath)
fmt.Println("总大小:", infoResp.Data.FileSize, "分片大小:", infoResp.Data.ChunkSize)

// 2. 下载指定范围的数据 (如第一片)
start := int64(0)
end := infoResp.Data.ChunkSize - 1
chunkData, _ := client.DownloadChunk(ctx, "video", filePath, start, end)

// 下载指定范围流 (适用于极大的分片范围)
readCloser, _ := client.DownloadChunkStream(ctx, "video", filePath, start, end)
defer readCloser.Close()
```