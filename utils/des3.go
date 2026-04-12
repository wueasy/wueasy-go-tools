package utils

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"fmt"
	"io"
)

// Decrypt3DES 3DES解密
// 使用DESede/CBC/PKCS5Padding算法进行解密,与Java端保持一致
// @param ciphertext 密文(第一个字节为IV长度，后面跟着IV和实际密文)
// @param key 密钥
// @return 明文
// @return 错误信息
func Decrypt3DES(ciphertext, key []byte) ([]byte, error) {
	// 检查密钥长度是否为24字节
	if len(key) != 24 {
		return nil, fmt.Errorf("密钥长度必须为24字节")
	}

	// 创建3DES密码块
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建3DES密码块失败: %v", err)
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
	if padding > 8 || padding < 1 {
		return nil, fmt.Errorf("无效的PKCS5填充")
	}

	return plaintext[:len(plaintext)-padding], nil
}

// Encrypt3DES 3DES加密
// 使用DESede/CBC/PKCS5Padding算法进行加密,与Java端保持一致
// @param plaintext 明文
// @param key 密钥
// @return 密文(第一个字节为IV长度，后面跟着IV和实际密文)
// @return 错误信息
func Encrypt3DES(plaintext, key []byte) ([]byte, error) {
	// 检查密钥长度是否为24字节
	if len(key) != 24 {
		return nil, fmt.Errorf("密钥长度必须为24字节")
	}

	// 创建3DES密码块
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建3DES密码块失败: %v", err)
	}

	// 添加PKCS5填充
	blockSize := block.BlockSize()
	padding := blockSize - len(plaintext)%blockSize
	if padding < 1 || padding > 8 {
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

// Encrypt3DESECB 3DES ECB模式加密 + PKCS5Padding
// 与Java端 Des3Helper.encryptECB 保持一致
// @param plaintext 明文
// @param key 密钥(24字节)
// @return 密文
// @return 错误信息
func Encrypt3DESECB(plaintext, key []byte) ([]byte, error) {
	if len(key) != 24 {
		return nil, fmt.Errorf("密钥长度必须为24字节")
	}
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建3DES密码块失败: %v", err)
	}
	bs := block.BlockSize()
	// PKCS5 padding
	padding := bs - len(plaintext)%bs
	padded := make([]byte, len(plaintext)+padding)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	// ECB逐块加密
	encrypted := make([]byte, len(padded))
	for i := 0; i < len(padded); i += bs {
		block.Encrypt(encrypted[i:i+bs], padded[i:i+bs])
	}
	return encrypted, nil
}

// Decrypt3DESECB 3DES ECB模式解密 + PKCS5Unpadding
// 与Java端 Des3Helper.decryptECBStr 保持一致
// @param ciphertext 密文
// @param key 密钥(24字节)
// @return 明文
// @return 错误信息
func Decrypt3DESECB(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 24 {
		return nil, fmt.Errorf("密钥长度必须为24字节")
	}
	block, err := des.NewTripleDESCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建3DES密码块失败: %v", err)
	}
	bs := block.BlockSize()
	if len(ciphertext)%bs != 0 {
		return nil, fmt.Errorf("密文长度不是块大小的整数倍")
	}
	// ECB逐块解密
	plaintext := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += bs {
		block.Decrypt(plaintext[i:i+bs], ciphertext[i:i+bs])
	}
	// PKCS5 unpadding
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > bs {
		return nil, fmt.Errorf("无效的PKCS5填充: %d", padding)
	}
	return plaintext[:len(plaintext)-padding], nil
}
