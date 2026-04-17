package fileClient

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// setupClient 创建测试用的客户端，通过直连模式绕过nacos，方便本地测试
func setupClient() *FileClient {
	// 初始化全局日志实例，防止Ctx(ctx).Errorf时空指针异常
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)

	// 这里可以配置成本地启动的 wueasy-file-server 地址
	// 假设本地服务器运行在 9830 端口
	return NewFileClient("file-server", "").SetBaseUrl("http://127.0.0.1:9830")
}

// generateTempFile 生成指定大小的临时文件用于测试
func generateTempFile(t *testing.T, size int64, prefix string) string {
	tmpFile, err := os.CreateTemp("", prefix+"-*.bin")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer tmpFile.Close()

	// 写入随机数据
	buf := make([]byte, 1024*1024) // 1MB buffer
	var written int64
	for written < size {
		writeSize := int64(len(buf))
		if size-written < writeSize {
			writeSize = size - written
		}

		_, err := rand.Read(buf[:writeSize])
		if err != nil {
			t.Fatalf("生成随机数据失败: %v", err)
		}

		n, err := tmpFile.Write(buf[:writeSize])
		if err != nil {
			t.Fatalf("写入临时文件失败: %v", err)
		}
		written += int64(n)
	}
	return tmpFile.Name()
}

func TestBasicOperations(t *testing.T) {
	client := setupClient()
	ctx := context.Background()
	businessType := "demo" // 假设服务端配置了此业务类型

	// 1. 测试小文件上传
	t.Run("UploadBytes", func(t *testing.T) {
		fileContent := []byte("Hello, Wueasy File Server!")
		fileName := "test_upload.txt"

		resp, err := client.UploadBytes(ctx, businessType, fileContent, fileName)
		if err != nil {
			t.Logf("注意: 如果服务端未启动或未配置 %s 类型，此测试将失败。跳过后续验证。", businessType)
			t.Skipf("上传请求失败: %v", err)
		}
		if !resp.Successful {
			t.Fatalf("上传文件失败，服务端返回: %s", resp.Msg)
		}
		t.Logf("上传成功，文件路径: %s", resp.Data.FilePath)

		// 2. 测试文件下载
		t.Run("Download", func(t *testing.T) {
			downloadedContent, err := client.Download(ctx, businessType, resp.Data.OriginalPath)
			if err != nil {
				t.Fatalf("下载请求失败: %v", err)
			}

			if !bytes.Equal(fileContent, downloadedContent) {
				t.Fatalf("下载的内容与上传的内容不一致")
			}
			t.Logf("下载验证成功")
		})

		// 3. 测试文件删除
		t.Run("Delete", func(t *testing.T) {
			delResp, err := client.Delete(ctx, businessType, resp.Data.OriginalPath)
			if err != nil {
				t.Fatalf("删除请求失败: %v", err)
			}
			if !delResp.Successful {
				t.Fatalf("删除文件失败，服务端返回: %s", delResp.Msg)
			}
			t.Logf("删除成功")
		})
	})
}

func TestLocalFileUpload(t *testing.T) {
	client := setupClient()
	ctx := context.Background()
	businessType := "demo"

	// 生成一个 2MB 的本地测试文件
	tmpFilePath := generateTempFile(t, 2*1024*1024, "test-local")
	defer os.Remove(tmpFilePath)

	t.Run("UploadLocalFile", func(t *testing.T) {
		resp, err := client.UploadLocalFile(ctx, businessType, tmpFilePath)
		if err != nil {
			t.Skipf("本地文件上传请求失败(可能服务端未启动): %v", err)
		}
		if !resp.Successful {
			t.Fatalf("本地文件上传失败，服务端返回: %s", resp.Msg)
		}
		t.Logf("本地文件上传成功，文件路径: %s", resp.Data.FilePath)

		// 清理文件
		_, _ = client.Delete(ctx, businessType, resp.Data.FilePath)
	})
}

func TestBatchDelete(t *testing.T) {
	client := setupClient()
	ctx := context.Background()
	businessType := "demo"

	// 先上传两个文件用于删除测试
	file1Content := []byte("Batch file 1")
	file2Content := []byte("Batch file 2")

	resp1, err := client.UploadBytes(ctx, businessType, file1Content, "batch1.txt")
	if err != nil {
		t.Skipf("准备测试数据失败(可能服务端未启动): %v", err)
	}
	resp2, err := client.UploadBytes(ctx, businessType, file2Content, "batch2.txt")
	if err != nil {
		t.Skipf("准备测试数据失败: %v", err)
	}

	t.Run("BatchDelete", func(t *testing.T) {
		filePaths := []string{resp1.Data.FilePath, resp2.Data.FilePath, "non_existent_file.txt"}
		batchResp, err := client.BatchDelete(ctx, businessType, filePaths)
		if err != nil {
			t.Fatalf("批量删除请求失败: %v", err)
		}
		if !batchResp.Successful {
			t.Fatalf("批量删除失败，服务端返回: %s", batchResp.Msg)
		}

		t.Logf("批量删除完成: 总数=%d, 成功=%d, 失败=%d",
			batchResp.Data.TotalCount, batchResp.Data.SuccessCount, batchResp.Data.FailedCount)

		// 验证前两个真实文件应该删除成功
		if batchResp.Data.SuccessCount < 2 {
			t.Errorf("预期至少成功删除2个文件，实际成功: %d", batchResp.Data.SuccessCount)
		}
	})
}

func TestChunkOperations(t *testing.T) {
	client := setupClient()
	ctx := context.Background()
	businessType := "demo" // 假设视频配置允许大文件和分片

	// 生成一个稍微大一点的文件 (例如 15MB) 用于测试分片
	fileSize := int64(15 * 1024 * 1024)
	tmpFilePath := generateTempFile(t, fileSize, "test-chunk")
	defer os.Remove(tmpFilePath)

	fileName := filepath.Base(tmpFilePath)
	var fileId string
	var finalFilePath string
	var chunkSize int64

	t.Run("InitChunkUpload", func(t *testing.T) {
		initResp, err := client.InitChunkUpload(ctx, businessType, fileName, fileSize)
		if err != nil {
			t.Skipf("初始化分片上传失败(可能服务端未启动): %v", err)
		}
		if !initResp.Successful {
			t.Fatalf("初始化分片上传失败，服务端返回: %s", initResp.Msg)
		}

		fileId = initResp.Data.FileId
		chunkSize = initResp.Data.ChunkSize
		t.Logf("初始化分片成功，FileId: %s, 总分片数: %d", fileId, initResp.Data.TotalChunks)

		// 顺序上传所有分片
		file, err := os.Open(tmpFilePath)
		if err != nil {
			t.Fatalf("打开测试文件失败: %v", err)
		}
		defer file.Close()

		for i := 0; i < initResp.Data.TotalChunks; i++ {
			// 读取当前分片的数据
			readSize := chunkSize
			if int64(i)*chunkSize+chunkSize > fileSize {
				readSize = fileSize - int64(i)*chunkSize
			}

			chunkData := make([]byte, readSize)
			_, err := io.ReadFull(file, chunkData)
			if err != nil {
				t.Fatalf("读取分片数据失败: %v", err)
			}

			uploadResp, err := client.UploadChunkBytes(ctx, businessType, fileId, i, chunkData, fileName)
			if err != nil {
				t.Fatalf("上传分片 %d 失败: %v", i, err)
			}
			if !uploadResp.Successful {
				t.Fatalf("上传分片 %d 失败，服务端返回: %s", i, uploadResp.Msg)
			}
			t.Logf("分片 %d 上传成功", i)
		}
	})

	// 如果初始化都失败被跳过，后续就不执行了
	if fileId == "" {
		return
	}

	t.Run("MergeChunks", func(t *testing.T) {
		mergeResp, err := client.MergeChunks(ctx, businessType, fileId)
		if err != nil {
			t.Fatalf("合并分片请求失败: %v", err)
		}
		if !mergeResp.Successful {
			t.Fatalf("合并分片失败，服务端返回: %s", mergeResp.Msg)
		}

		finalFilePath = mergeResp.Data.OriginalPath
		t.Logf("合并分片成功，最终文件路径: %s", finalFilePath)
	})

	if finalFilePath == "" {
		return
	}

	t.Run("GetChunkDownloadInfo", func(t *testing.T) {
		infoResp, err := client.GetChunkDownloadInfo(ctx, businessType, finalFilePath)
		if err != nil {
			t.Fatalf("获取分片下载信息失败: %v", err)
		}
		if !infoResp.Successful {
			t.Fatalf("获取分片下载信息失败，服务端返回: %s", infoResp.Msg)
		}

		t.Logf("获取分片下载信息成功，文件总大小: %d, 分片大小: %d", infoResp.Data.FileSize, infoResp.Data.ChunkSize)

		if infoResp.Data.FileSize != fileSize {
			t.Errorf("下载的文件大小与上传不一致: expected %d, got %d", fileSize, infoResp.Data.FileSize)
		}
	})

	t.Run("DownloadChunk", func(t *testing.T) {
		// 尝试下载第一个分片
		start := int64(0)
		end := chunkSize - 1
		if end >= fileSize {
			end = fileSize - 1
		}

		chunkData, err := client.DownloadChunk(ctx, businessType, finalFilePath, start, end)
		if err != nil {
			t.Fatalf("下载分片请求失败: %v", err)
		}

		expectedSize := end - start + 1
		if int64(len(chunkData)) != expectedSize {
			t.Errorf("下载分片数据大小不匹配: expected %d, got %d", expectedSize, len(chunkData))
		}
		t.Logf("下载分片成功，获取数据大小: %d 字节", len(chunkData))
	})

	// 清理大文件
	t.Run("CleanupChunkFile", func(t *testing.T) {
		_, _ = client.Delete(ctx, businessType, finalFilePath)
	})
}
