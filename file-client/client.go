package fileClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	log2 "github.com/wueasy/wueasy-go-tools/log"
	nacosClient "github.com/wueasy/wueasy-go-tools/nacos"
)

// FileClient 文件服务客户端
type FileClient struct {
	ServiceName string
	GroupName   string
	BaseUrl     string // 基础地址，如果有值则优先使用，不通过nacos发现
	Timeout     time.Duration
	httpClient  *http.Client
}

// NewFileClient 创建文件服务客户端
// serviceName: nacos注册的服务名
// groupName: nacos注册的分组名，默认DEFAULT_GROUP
func NewFileClient(serviceName, groupName string) *FileClient {
	if groupName == "" {
		groupName = "DEFAULT_GROUP"
	}
	timeout := 30 * time.Second
	return &FileClient{
		ServiceName: serviceName,
		GroupName:   groupName,
		Timeout:     timeout,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// SetBaseUrl 设置基础地址，如果设置了此地址，则不会通过nacos进行服务发现
func (c *FileClient) SetBaseUrl(baseUrl string) *FileClient {
	c.BaseUrl = baseUrl
	return c
}

// SetTimeout 设置客户端超时时间
func (c *FileClient) SetTimeout(timeout time.Duration) *FileClient {
	c.Timeout = timeout
	c.httpClient.Timeout = timeout
	return c
}

// getServerUrl 获取服务地址
func (c *FileClient) getServerUrl(ctx context.Context) (string, error) {
	if c.BaseUrl != "" {
		return c.BaseUrl, nil
	}

	instance, err := nacosClient.GetHealthyInstanceWithGroup(c.ServiceName, c.GroupName, "")
	if err != nil {
		log2.Ctx(ctx).Errorf("获取文件服务[%s]实例失败: %v", c.ServiceName, err)
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", instance.Ip, instance.Port), nil
}

// UploadResultData 上传结果数据
type UploadResultData struct {
	FilePath     string `json:"filePath"`
	FileName     string `json:"fileName"`
	OriginalPath string `json:"originalPath"`
	FileSize     int64  `json:"fileSize"`
	ExtName      string `json:"extName"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	Code       int              `json:"code"`
	Successful bool             `json:"successful"`
	Msg        string           `json:"msg"`
	Data       UploadResultData `json:"data"`
}

// Upload 上传文件
// ctx: 上下文
// businessType: 业务类型代码
// fileData: 文件数据读取器
// fileName: 文件名
func (c *FileClient) Upload(ctx context.Context, businessType string, fileData io.Reader, fileName string) (*UploadResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	reqUrl := fmt.Sprintf("%s/upload/%s", serverUrl, businessType)
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/upload", serverUrl)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			errChan <- fmt.Errorf("创建表单文件失败: %v", err)
			return
		}

		_, err = io.Copy(part, fileData)
		if err != nil {
			errChan <- fmt.Errorf("拷贝文件内容失败: %v", err)
			return
		}
		errChan <- nil
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, pr)
	if err != nil {
		log2.Ctx(ctx).Errorf("创建上传请求失败: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行上传请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if err := <-errChan; err != nil {
		log2.Ctx(ctx).Errorf("上传流处理错误: %v", err)
		return nil, err
	}

	var result UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析上传响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}

// --- 分片下载接口 ---

// ChunkDownloadInfoData 分片下载信息
type ChunkDownloadInfoData struct {
	FileSize    int64  `json:"fileSize"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
	FileName    string `json:"fileName"`
}

// ChunkDownloadInfoResponse 分片下载信息响应
type ChunkDownloadInfoResponse struct {
	Code       int                   `json:"code"`
	Successful bool                  `json:"successful"`
	Msg        string                `json:"msg"`
	Data       ChunkDownloadInfoData `json:"data"`
}

// GetChunkDownloadInfo 获取分片下载信息
// ctx: 上下文
// businessType: 业务类型代码
// filePath: 文件相对路径
func (c *FileClient) GetChunkDownloadInfo(ctx context.Context, businessType, filePath string) (*ChunkDownloadInfoResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/download/chunk/info?filePath=%s", serverUrl, parseFilePath(filePath))
	} else {
		reqUrl = fmt.Sprintf("%s/download/chunk/info/%s/%s", serverUrl, businessType, parseFilePath(filePath))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	if err != nil {
		log2.Ctx(ctx).Errorf("创建获取分片下载信息请求失败: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行获取分片下载信息请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result ChunkDownloadInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析获取分片下载信息响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}

// DownloadChunkStream 分片下载流
// ctx: 上下文
// businessType: 业务类型代码
// filePath: 文件相对路径
// start: 起始字节位置
// end: 结束字节位置
// 返回文件内容读取器，调用方需要负责关闭 io.ReadCloser
func (c *FileClient) DownloadChunkStream(ctx context.Context, businessType, filePath string, start, end int64) (io.ReadCloser, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/download/chunk?filePath=%s&start=%d&end=%d", serverUrl, parseFilePath(filePath), start, end)
	} else {
		reqUrl = fmt.Sprintf("%s/download/chunk/%s/%s?start=%d&end=%d", serverUrl, businessType, parseFilePath(filePath), start, end)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	if err != nil {
		log2.Ctx(ctx).Errorf("创建分片下载请求失败: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行分片下载请求失败: %v", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		defer resp.Body.Close()
		err := fmt.Errorf("分片下载失败，HTTP状态码: %d", resp.StatusCode)
		log2.Ctx(ctx).Error(err.Error())
		return nil, err
	}

	return resp.Body, nil
}

// DownloadChunk 分片下载字节
// ctx: 上下文
// businessType: 业务类型代码
// filePath: 文件相对路径
// start: 起始字节位置
// end: 结束字节位置
func (c *FileClient) DownloadChunk(ctx context.Context, businessType, filePath string, start, end int64) ([]byte, error) {
	rc, err := c.DownloadChunkStream(ctx, businessType, filePath, start, end)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// --- 分片上传接口 ---

// InitChunkUploadRequest 初始化分片上传请求
type InitChunkUploadRequest struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
}

// InitChunkUploadData 初始化分片上传结果
type InitChunkUploadData struct {
	FileId      string `json:"fileId"`
	UploadId    string `json:"uploadId"`
	FileName    string `json:"fileName"`
	ChunkSize   int64  `json:"chunkSize"`
	TotalChunks int    `json:"totalChunks"`
	ExpiresAt   int64  `json:"expiresAt"`
}

// InitChunkUploadResponse 初始化分片上传响应
type InitChunkUploadResponse struct {
	Code       int                 `json:"code"`
	Successful bool                `json:"successful"`
	Msg        string              `json:"msg"`
	Data       InitChunkUploadData `json:"data"`
}

// InitChunkUpload 初始化分片上传
// ctx: 上下文
// businessType: 业务类型代码
// fileName: 文件名
// fileSize: 文件总大小(字节)
func (c *FileClient) InitChunkUpload(ctx context.Context, businessType, fileName string, fileSize int64) (*InitChunkUploadResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/upload/chunk/init", serverUrl)
	} else {
		reqUrl = fmt.Sprintf("%s/upload/chunk/init/%s", serverUrl, businessType)
	}

	reqBody := InitChunkUploadRequest{
		FileName: fileName,
		FileSize: fileSize,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log2.Ctx(ctx).Errorf("序列化初始化分片请求失败: %v", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log2.Ctx(ctx).Errorf("创建初始化分片请求失败: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行初始化分片请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result InitChunkUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析初始化分片响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}

// UploadChunkData 分片上传结果
type UploadChunkData struct {
	FileId      string `json:"fileId"`
	ChunkIndex  int    `json:"chunkIndex"`
	TotalChunks int    `json:"totalChunks"`
	Uploaded    bool   `json:"uploaded"`
}

// UploadChunkResponse 分片上传响应
type UploadChunkResponse struct {
	Code       int             `json:"code"`
	Successful bool            `json:"successful"`
	Msg        string          `json:"msg"`
	Data       UploadChunkData `json:"data"`
}

// UploadChunk 上传分片
// ctx: 上下文
// businessType: 业务类型代码
// fileId: 文件唯一标识
// chunkIndex: 分片索引(从0开始)
// fileData: 分片数据读取器
// fileName: 文件名
func (c *FileClient) UploadChunk(ctx context.Context, businessType, fileId string, chunkIndex int, fileData io.Reader, fileName string) (*UploadChunkResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/upload/chunk", serverUrl)
	} else {
		reqUrl = fmt.Sprintf("%s/upload/chunk/%s", serverUrl, businessType)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer writer.Close()

		_ = writer.WriteField("fileId", fileId)
		_ = writer.WriteField("chunkIndex", fmt.Sprintf("%d", chunkIndex))

		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			errChan <- fmt.Errorf("创建分片表单文件失败: %v", err)
			return
		}

		_, err = io.Copy(part, fileData)
		if err != nil {
			errChan <- fmt.Errorf("拷贝分片文件内容失败: %v", err)
			return
		}
		errChan <- nil
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, pr)
	if err != nil {
		log2.Ctx(ctx).Errorf("创建分片上传请求失败: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行分片上传请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if err := <-errChan; err != nil {
		log2.Ctx(ctx).Errorf("分片上传流处理错误: %v", err)
		return nil, err
	}

	var result UploadChunkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析分片上传响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}

// UploadChunkBytes 上传分片字节
func (c *FileClient) UploadChunkBytes(ctx context.Context, businessType, fileId string, chunkIndex int, fileData []byte, fileName string) (*UploadChunkResponse, error) {
	return c.UploadChunk(ctx, businessType, fileId, chunkIndex, bytes.NewReader(fileData), fileName)
}

// MergeChunksRequest 合并分片请求
type MergeChunksRequest struct {
	FileId string `json:"fileId"`
}

// MergeChunks 合并分片
// ctx: 上下文
// businessType: 业务类型代码
// fileId: 文件唯一标识
func (c *FileClient) MergeChunks(ctx context.Context, businessType, fileId string) (*UploadResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/upload/merge", serverUrl)
	} else {
		reqUrl = fmt.Sprintf("%s/upload/merge/%s", serverUrl, businessType)
	}

	reqBody := MergeChunksRequest{
		FileId: fileId,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log2.Ctx(ctx).Errorf("序列化合并分片请求失败: %v", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log2.Ctx(ctx).Errorf("创建合并分片请求失败: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行合并分片请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析合并分片响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}

// UploadBytes 上传文件字节
// ctx: 上下文
// businessType: 业务类型代码
// fileData: 文件二进制数据
// fileName: 文件名
func (c *FileClient) UploadBytes(ctx context.Context, businessType string, fileData []byte, fileName string) (*UploadResponse, error) {
	return c.Upload(ctx, businessType, bytes.NewReader(fileData), fileName)
}

// UploadLocalFile 上传本地文件
// ctx: 上下文
// businessType: 业务类型代码
// localFilePath: 本地文件路径
func (c *FileClient) UploadLocalFile(ctx context.Context, businessType string, localFilePath string) (*UploadResponse, error) {
	file, err := os.Open(localFilePath)
	if err != nil {
		log2.Ctx(ctx).Errorf("打开本地文件失败: %v", err)
		return nil, err
	}
	defer file.Close()

	fileName := filepath.Base(localFilePath)
	return c.Upload(ctx, businessType, file, fileName)
}

// parseFilePath 处理文件路径编码
func parseFilePath(filePath string) string {
	parts := strings.Split(filePath, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// DownloadStream 下载文件流
// ctx: 上下文
// businessType: 业务类型代码
// filePath: 文件相对路径
// 返回文件内容读取器，调用方需要负责关闭 io.ReadCloser
func (c *FileClient) DownloadStream(ctx context.Context, businessType string, filePath string) (io.ReadCloser, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	// 移除可能的前导斜杠
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/download?filePath=%s", serverUrl, parseFilePath(filePath))
	} else {
		reqUrl = fmt.Sprintf("%s/download/%s/%s", serverUrl, businessType, parseFilePath(filePath))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqUrl, nil)
	if err != nil {
		log2.Ctx(ctx).Errorf("创建下载请求失败: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行下载请求失败: %v", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		defer resp.Body.Close()
		err := fmt.Errorf("下载失败，HTTP状态码: %d", resp.StatusCode)
		log2.Ctx(ctx).Error(err.Error())
		return nil, err
	}

	return resp.Body, nil
}

// Download 下载文件 (返回文件内容)
// ctx: 上下文
// businessType: 业务类型代码
// filePath: 文件相对路径
func (c *FileClient) Download(ctx context.Context, businessType string, filePath string) ([]byte, error) {
	rc, err := c.DownloadStream(ctx, businessType, filePath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// DeleteResponse 删除响应
type DeleteResponse struct {
	Code       int    `json:"code"`
	Successful bool   `json:"successful"`
	Msg        string `json:"msg"`
}

// Delete 删除文件
// ctx: 上下文
// businessType: 业务类型代码
// filePath: 文件相对路径
func (c *FileClient) Delete(ctx context.Context, businessType string, filePath string) (*DeleteResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	// 移除可能的前导斜杠
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}

	var reqUrl string
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/delete?filePath=%s", serverUrl, parseFilePath(filePath))
	} else {
		reqUrl = fmt.Sprintf("%s/delete/%s/%s", serverUrl, businessType, parseFilePath(filePath))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, nil)
	if err != nil {
		log2.Ctx(ctx).Errorf("创建删除请求失败: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行删除请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result DeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析删除响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	FilePaths []string `json:"filePaths"`
}

// BatchDeleteResponse 批量删除响应
type BatchDeleteResponse struct {
	Code       int    `json:"code"`
	Successful bool   `json:"successful"`
	Msg        string `json:"msg"`
	Data       struct {
		TotalCount   int `json:"totalCount"`
		SuccessCount int `json:"successCount"`
		FailedCount  int `json:"failedCount"`
		Results      []struct {
			FilePath string `json:"filePath"`
			Success  bool   `json:"success"`
			Message  string `json:"message"`
		} `json:"results"`
	} `json:"data"`
}

// BatchDelete 批量删除文件
// ctx: 上下文
// businessType: 业务类型代码
// filePaths: 文件相对路径数组
func (c *FileClient) BatchDelete(ctx context.Context, businessType string, filePaths []string) (*BatchDeleteResponse, error) {
	serverUrl, err := c.getServerUrl(ctx)
	if err != nil {
		return nil, err
	}

	reqUrl := fmt.Sprintf("%s/delete/batch/%s", serverUrl, businessType)
	if businessType == "" {
		reqUrl = fmt.Sprintf("%s/delete/batch", serverUrl)
	}

	reqBody := BatchDeleteRequest{
		FilePaths: filePaths,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log2.Ctx(ctx).Errorf("序列化批量删除请求失败: %v", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		log2.Ctx(ctx).Errorf("创建批量删除请求失败: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log2.Ctx(ctx).Errorf("执行批量删除请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result BatchDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log2.Ctx(ctx).Errorf("解析批量删除响应失败: %v", err)
		return nil, err
	}

	return &result, nil
}
