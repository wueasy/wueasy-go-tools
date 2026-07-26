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
	"strings"
)

// parsePublicKey 解析 RSA 公钥，支持 PKIX 和 PKCS#1 两种格式
func parsePublicKey(derBytes []byte) (*rsa.PublicKey, error) {
	// 优先尝试 PKIX（SubjectPublicKeyInfo，PEM头: PUBLIC KEY）
	pub, err := x509.ParsePKIXPublicKey(derBytes)
	if err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not of type RSA")
		}
		return rsaPub, nil
	}
	// 回退尝试 PKCS#1（PEM头: RSA PUBLIC KEY）
	rsaPub, err := x509.ParsePKCS1PublicKey(derBytes)
	if err != nil {
		return nil, errors.New("failed to parse public key: unsupported format")
	}
	return rsaPub, nil
}

// parsePrivateKey 解析 RSA 私钥，支持 PKCS#8 和 PKCS#1 两种格式
func parsePrivateKey(derBytes []byte) (*rsa.PrivateKey, error) {
	// 优先尝试 PKCS#8（PEM头: PRIVATE KEY）
	priv, err := x509.ParsePKCS8PrivateKey(derBytes)
	if err == nil {
		rsaPriv, ok := priv.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not of type RSA")
		}
		return rsaPriv, nil
	}
	// 回退尝试 PKCS#1（PEM头: RSA PRIVATE KEY）
	rsaPriv, err := x509.ParsePKCS1PrivateKey(derBytes)
	if err != nil {
		return nil, errors.New("failed to parse private key: unsupported format")
	}
	return rsaPriv, nil
}

// RsaEncrypt 使用公钥加密数据（PKCS1v15 填充）
func RsaEncrypt(data string, publicKey string) (string, error) {
	publicKey = ensurePemFormat(publicKey, "PUBLIC KEY")

	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", errors.New("failed to decode public key PEM block")
	}

	rsaPubKey, err := parsePublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	partLen := rsaPubKey.N.BitLen()/8 - 11
	chunks := split([]byte(data), partLen)

	var encrypted []byte
	for _, chunk := range chunks {
		encryptedChunk, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPubKey, chunk)
		if err != nil {
			return "", err
		}
		encrypted = append(encrypted, encryptedChunk...)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// RsaEncryptOAEP 使用公钥加密数据（OAEP 填充）
func RsaEncryptOAEP(data string, publicKey string) (string, error) {
	publicKey = ensurePemFormat(publicKey, "PUBLIC KEY")

	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return "", errors.New("failed to decode public key PEM block")
	}

	rsaPubKey, err := parsePublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// OAEP-SHA256 开销 = 2 + 2*32 = 66 字节
	partLen := rsaPubKey.N.BitLen()/8 - 66
	chunks := split([]byte(data), partLen)

	var encrypted []byte
	for _, chunk := range chunks {
		encryptedChunk, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, chunk, nil)
		if err != nil {
			return "", err
		}
		encrypted = append(encrypted, encryptedChunk...)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// RsaDecrypt 使用私钥解密（PKCS1v15 填充）
func RsaDecrypt(str string, privatekey string) (string, error) {
	privatekey = ensurePemFormat(privatekey, "RSA PRIVATE KEY")

	block, _ := pem.Decode([]byte(privatekey))
	if block == nil {
		return "", errors.New("failed to decode private key PEM block")
	}

	rsaPrivKey, err := parsePrivateKey(block.Bytes)
	if err != nil {
		return "", err
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
	return buffer.String(), nil
}

// RsaDecryptOAEP 使用私钥解密数据（OAEP 填充）
func RsaDecryptOAEP(str string, privatekey string) (string, error) {
	privatekey = ensurePemFormat(privatekey, "RSA PRIVATE KEY")

	block, _ := pem.Decode([]byte(privatekey))
	if block == nil {
		return "", errors.New("failed to decode private key PEM block")
	}

	rsaPrivKey, err := parsePrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	partLen := rsaPrivKey.PublicKey.N.BitLen() / 8
	chunks := split([]byte(ciphertext), partLen)

	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		decrypted, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPrivKey, chunk, nil)
		if err != nil {
			return "", err
		}
		buffer.Write(decrypted)
	}
	return buffer.String(), nil
}

// KeyFormat 密钥输出格式
type KeyFormat int

const (
	KeyFormatPKCS8 KeyFormat = iota // PKCS#8 私钥（PRIVATE KEY）+ PKIX 公钥（PUBLIC KEY）
	KeyFormatPKCS1                  // PKCS#1 私钥（RSA PRIVATE KEY）+ PKCS#1 公钥（RSA PUBLIC KEY）
)

// GenerateRSAKey 生成 RSA 密钥对
// bits 为密钥长度，推荐 2048 或 4096
// format 为密钥输出格式（KeyFormatPKCS8 或 KeyFormatPKCS1）
func GenerateRSAKey(bits int, format KeyFormat) (privateKey string, publicKey string, err error) {
	if bits < 1024 {
		return "", "", errors.New("rsa key bits must be at least 1024")
	}

	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", err
	}

	switch format {
	case KeyFormatPKCS1:
		// 私钥：PKCS#1 DER
		privDer := x509.MarshalPKCS1PrivateKey(priv)
		privateKey = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privDer,
		}))
		// 公钥：PKCS#1 DER
		pubDer := x509.MarshalPKCS1PublicKey(&priv.PublicKey)
		publicKey = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubDer,
		}))
	default: // KeyFormatPKCS8
		// 私钥：PKCS#8 DER
		privDer, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return "", "", err
		}
		privateKey = string(pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privDer,
		}))
		// 公钥：PKIX DER
		pubDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		if err != nil {
			return "", "", err
		}
		publicKey = string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubDer,
		}))
	}

	return privateKey, publicKey, nil
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
