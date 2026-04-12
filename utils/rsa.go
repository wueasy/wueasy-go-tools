package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"log"
	"strings"
)

// RsaEncrypt 使用公钥加密数据
func RsaEncrypt(data string, publicKey string) (string, error) {
	// 如果公钥没有PEM头尾，自动补上
	publicKey = ensurePemFormat(publicKey, "PUBLIC KEY")

	// 解析公钥
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", errors.New("public key error!")
	}

	// 解析公钥
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 类型断言以获取 RSA 公钥
	rsaPubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key is not RSA")
	}

	// 计算分块大小（密钥长度减去11字节的填充）
	partLen := rsaPubKey.N.BitLen()/8 - 11
	chunks := split([]byte(data), partLen)

	var encrypted []byte
	for _, chunk := range chunks {
		// 使用 PKCS1v15 填充进行加密
		encryptedChunk, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, chunk)
		if err != nil {
			return "", err
		}
		encrypted = append(encrypted, encryptedChunk...)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// RsaEncryptOAEP 使用公钥加密数据（OAEP填充）
func RsaEncryptOAEP(data string, publicKey string) (string, error) {
	// 如果公钥没有PEM头尾，自动补上
	publicKey = ensurePemFormat(publicKey, "PUBLIC KEY")

	// 解析公钥
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", errors.New("public key error!")
	}

	// 解析公钥
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 类型断言以获取 RSA 公钥
	rsaPubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key is not RSA")
	}

	// 计算分块大小（密钥长度减去42字节的OAEP填充）
	partLen := rsaPubKey.N.BitLen()/8 - 42
	chunks := split([]byte(data), partLen)

	var encrypted []byte
	for _, chunk := range chunks {
		// 使用 OAEP 填充进行加密
		encryptedChunk, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, chunk, nil)
		if err != nil {
			return "", err
		}
		encrypted = append(encrypted, encryptedChunk...)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// RsaDecrypt 解密
func RsaDecrypt(str string, privatekey string) (string, error) {

	// 如果私钥没有PEM头尾，自动补上
	privatekey = ensurePemFormat(privatekey, "RSA PRIVATE KEY")

	//解密
	block, _ := pem.Decode([]byte(privatekey))
	if block == nil {
		return "", errors.New("private key error!")
	}
	//解析PKCS1格式的私钥
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 类型断言以获取 RSA 私钥（如果适用）
	rsaPrivKey, ok := priv.(*rsa.PrivateKey)

	if !ok {
		log.Fatalf("private key is not RSA")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	partLen := rsaPrivKey.PublicKey.N.BitLen() / 8
	chunks := split([]byte(ciphertext), partLen)

	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, rsaPrivKey, chunk)
		if err != nil {
			return "", err
		}
		buffer.Write(decrypted)
	}
	return buffer.String(), err
}

// RsaDecryptOAEP 使用私钥解密数据（OAEP填充）
func RsaDecryptOAEP(str string, privatekey string) (string, error) {
	// 如果私钥没有PEM头尾，自动补上
	privatekey = ensurePemFormat(privatekey, "RSA PRIVATE KEY")

	// 解析私钥
	block, _ := pem.Decode([]byte(privatekey))
	if block == nil {
		return "", errors.New("private key error!")
	}

	// 解析PKCS8格式的私钥
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// 类型断言以获取 RSA 私钥
	rsaPrivKey, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("private key is not RSA")
	}

	// 解码base64密文
	ciphertext, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	// 分块解密
	partLen := rsaPrivKey.PublicKey.N.BitLen() / 8
	chunks := split([]byte(ciphertext), partLen)

	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		// 使用 OAEP 填充进行解密
		decrypted, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivKey, chunk, nil)
		if err != nil {
			return "", err
		}
		buffer.Write(decrypted)
	}
	return buffer.String(), nil
}

// ensurePemFormat 如果字符串没有PEM头尾，自动补上
func ensurePemFormat(key string, keyType string) string {
	header := "-----BEGIN " + keyType + "-----"
	footer := "-----END " + keyType + "-----"
	if strings.Contains(key, header) {
		return key
	}
	return header + "\n" + key + "\n" + footer
}

func split(buf []byte, lim int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}
