package utils

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/tjfoc/gmsm/sm4"
)

// EncryptSM4 SM4加密
// 使用SM4/CBC/PKCS5Padding算法进行加密
// @param plaintext 明文
// @param key 密钥(16字节)
// @return 密文(第一个字节为IV长度，后面跟着IV和实际密文)
// @return 错误信息
func EncryptSM4(plaintext, key []byte) ([]byte, error) {
	// 检查密钥长度是否为16字节
	if len(key) != 16 {
		return nil, fmt.Errorf("密钥长度必须为16字节")
	}

	// 创建SM4密码块
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建SM4密码块失败: %v", err)
	}

	// 添加PKCS5填充
	blockSize := block.BlockSize()
	padding := blockSize - len(plaintext)%blockSize
	if padding < 1 || padding > 16 {
		return nil, fmt.Errorf("无效的PKCS5填充")
	}

	padtext := make([]byte, len(plaintext)+padding)
	copy(padtext, plaintext)
	for i := len(plaintext); i < len(padtext); i++ {
		padtext[i] = byte(padding)
	}

	// 生成随机IV
	iv := make([]byte, blockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("生成随机IV失败: %v", err)
	}

	// 创建CBC加密器
	cipher := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(padtext))
	cipher.CryptBlocks(encrypted, padtext)

	// 将IV长度、IV和密文拼接在一起
	// 第一个字节存储IV长度
	ciphertext := make([]byte, 1+len(iv)+len(encrypted))
	ciphertext[0] = byte(len(iv))
	copy(ciphertext[1:], iv)
	copy(ciphertext[1+len(iv):], encrypted)

	return ciphertext, nil
}

// DecryptSM4 SM4解密
// 使用SM4/CBC/PKCS5Padding算法进行解密
// @param ciphertext 密文(第一个字节为IV长度，后面跟着IV和实际密文)
// @param key 密钥(16字节)
// @return 明文
// @return 错误信息
func DecryptSM4(ciphertext, key []byte) ([]byte, error) {
	// 检查密钥长度是否为16字节
	if len(key) != 16 {
		return nil, fmt.Errorf("密钥长度必须为16字节")
	}

	// 创建SM4密码块
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建SM4密码块失败: %v", err)
	}

	// 检查密文长度
	if len(ciphertext) < 2 {
		return nil, fmt.Errorf("密文长度不足")
	}

	// 获取IV长度
	ivLen := int(ciphertext[0])
	if ivLen <= 0 || ivLen > 16 {
		return nil, fmt.Errorf("无效的IV长度: %d", ivLen)
	}

	// 检查密文长度是否足够
	if len(ciphertext) < 1+ivLen {
		return nil, fmt.Errorf("密文长度不足")
	}

	// 提取IV和实际密文
	iv := ciphertext[1 : 1+ivLen]
	actualCiphertext := ciphertext[1+ivLen:]

	// 创建CBC解密器
	cipher := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(actualCiphertext))
	cipher.CryptBlocks(plaintext, actualCiphertext)

	// 去除PKCS5填充
	padding := int(plaintext[len(plaintext)-1])
	if padding > 16 || padding < 1 {
		return nil, fmt.Errorf("无效的PKCS5填充")
	}

	return plaintext[:len(plaintext)-padding], nil
}
