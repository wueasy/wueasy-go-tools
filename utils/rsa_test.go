package utils

import (
	"strings"
	"testing"
)

const testData = "hello rsa 测试数据"

// ---- 密钥生成 ----

func TestGenerateRSAKey_PKCS8(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal("GenerateRSAKey PKCS8 error:", err)
	}
	if !strings.Contains(priv, "-----BEGIN PRIVATE KEY-----") {
		t.Error("PKCS8 private key missing header")
	}
	if !strings.Contains(pub, "-----BEGIN PUBLIC KEY-----") {
		t.Error("PKCS8 public key missing header")
	}
}

func TestGenerateRSAKey_PKCS1(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS1)
	if err != nil {
		t.Fatal("GenerateRSAKey PKCS1 error:", err)
	}
	if !strings.Contains(priv, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("PKCS1 private key missing header")
	}
	if !strings.Contains(pub, "-----BEGIN RSA PUBLIC KEY-----") {
		t.Error("PKCS1 public key missing header")
	}
}

func TestGenerateRSAKey_BitsTooSmall(t *testing.T) {
	_, _, err := GenerateRSAKey(512, KeyFormatPKCS8)
	if err == nil {
		t.Error("expected error for bits < 1024")
	}
}

// ---- PKCS1v15 加密解密 ----

func TestRsaEncryptDecrypt_PKCS8Key(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt(testData, pub)
	if err != nil {
		t.Fatal("RsaEncrypt error:", err)
	}

	dec, err := RsaDecrypt(enc, priv)
	if err != nil {
		t.Fatal("RsaDecrypt error:", err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

func TestRsaEncryptDecrypt_PKCS1Key(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS1)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt(testData, pub)
	if err != nil {
		t.Fatal("RsaEncrypt error:", err)
	}

	dec, err := RsaDecrypt(enc, priv)
	if err != nil {
		t.Fatal("RsaDecrypt error:", err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

// ---- OAEP 加密解密 ----

func TestRsaEncryptDecryptOAEP_PKCS8Key(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncryptOAEP(testData, pub)
	if err != nil {
		t.Fatal("RsaEncryptOAEP error:", err)
	}

	dec, err := RsaDecryptOAEP(enc, priv)
	if err != nil {
		t.Fatal("RsaDecryptOAEP error:", err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

func TestRsaEncryptDecryptOAEP_PKCS1Key(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS1)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncryptOAEP(testData, pub)
	if err != nil {
		t.Fatal("RsaEncryptOAEP error:", err)
	}

	dec, err := RsaDecryptOAEP(enc, priv)
	if err != nil {
		t.Fatal("RsaDecryptOAEP error:", err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

// ---- 跨格式兼容 ----

func TestCrossFormat_PKCS8KeyWithPKCS1Functions(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt(testData, pub)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := RsaDecrypt(enc, priv)
	if err != nil {
		t.Fatal(err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

func TestCrossFormat_PKCS1KeyWithOAEPFunctions(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS1)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncryptOAEP(testData, pub)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := RsaDecryptOAEP(enc, priv)
	if err != nil {
		t.Fatal(err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

// ---- 无 PEM 头尾的裸 base64 密钥 ----

func TestEncryptDecrypt_WithoutPemHeaders(t *testing.T) {
	// 生成完整 PEM 密钥，然后去掉头尾模拟裸 base64 场景
	privFull, pubFull, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	// 提取 base64 部分
	stripHeaders := func(pem string) string {
		lines := strings.Split(strings.TrimSpace(pem), "\n")
		var sb strings.Builder
		for _, line := range lines {
			if strings.HasPrefix(line, "-----") {
				continue
			}
			sb.WriteString(strings.TrimSpace(line))
		}
		return sb.String()
	}

	privBare := stripHeaders(privFull)
	pubBare := stripHeaders(pubFull)

	enc, err := RsaEncrypt(testData, pubBare)
	if err != nil {
		t.Fatal("RsaEncrypt with bare key error:", err)
	}

	dec, err := RsaDecrypt(enc, privBare)
	if err != nil {
		t.Fatal("RsaDecrypt with bare key error:", err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

// ---- 长文本 ----

func TestEncryptDecrypt_LongText(t *testing.T) {
	longText := strings.Repeat("A", 500)
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt(longText, pub)
	if err != nil {
		t.Fatal("long text RsaEncrypt error:", err)
	}

	dec, err := RsaDecrypt(enc, priv)
	if err != nil {
		t.Fatal("long text RsaDecrypt error:", err)
	}
	if dec != longText {
		t.Error("long text decrypt mismatch")
	}
}

func TestEncryptDecryptOAEP_LongText(t *testing.T) {
	longText := strings.Repeat("B", 300)
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncryptOAEP(longText, pub)
	if err != nil {
		t.Fatal("long text RsaEncryptOAEP error:", err)
	}

	dec, err := RsaDecryptOAEP(enc, priv)
	if err != nil {
		t.Fatal("long text RsaDecryptOAEP error:", err)
	}
	if dec != longText {
		t.Error("long text OAEP decrypt mismatch")
	}
}

// ---- 4096 位密钥 ----

func TestEncryptDecrypt_4096Bits(t *testing.T) {
	priv, pub, err := GenerateRSAKey(4096, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt(testData, pub)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := RsaDecrypt(enc, priv)
	if err != nil {
		t.Fatal(err)
	}
	if dec != testData {
		t.Errorf("decrypt mismatch: got %q, want %q", dec, testData)
	}
}

// ---- 错误密钥 ----

func TestDecrypt_WrongKey(t *testing.T) {
	_, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt(testData, pub)
	if err != nil {
		t.Fatal(err)
	}

	otherPriv, _, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	_, err = RsaDecrypt(enc, otherPriv)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestEncrypt_InvalidPublicKey(t *testing.T) {
	_, err := RsaEncrypt(testData, "not a valid key")
	if err == nil {
		t.Error("expected error for invalid public key")
	}
}

func TestDecrypt_InvalidPrivateKey(t *testing.T) {
	_, err := RsaDecrypt("invalid", "not a valid key")
	if err == nil {
		t.Error("expected error for invalid private key")
	}
}

// ---- 空文本 ----

func TestEncryptDecrypt_EmptyText(t *testing.T) {
	priv, pub, err := GenerateRSAKey(2048, KeyFormatPKCS8)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := RsaEncrypt("", pub)
	if err != nil {
		t.Fatal(err)
	}

	dec, err := RsaDecrypt(enc, priv)
	if err != nil {
		t.Fatal(err)
	}
	if dec != "" {
		t.Errorf("decrypt mismatch: got %q, want empty", dec)
	}
}
