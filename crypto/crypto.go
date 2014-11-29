package crypto

import (
    "fmt"
    "bytes"
    "encoding/base64"
    "crypto/rand"
    "crypto/cipher"
    "crypto/rsa"
    "crypto/aes"
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
)

type Encrypted struct {
    Ciphertext string
    Mode string
    Inputs map[string]string
    Keys map[string]string
}

type Signed struct {
    Message string
    Mode string
    Inputs map[string]string
    Signature string
}

/*********************************************************************
 * Helper functions (no encoding)
 *********************************************************************/

func RandomBytes(size int) ([]byte, error) {
    randomBytes := make([]byte, size)
    _, err := rand.Read(randomBytes)
    if err != nil {
        return nil, err
    } else {
        return randomBytes, nil
    }
}

// https://www.socketloop.com/tutorials/golang-padding-un-padding-data
func Pad(src []byte, blockSize int) []byte {
    padding := blockSize - len(src)%blockSize
    padtext := bytes.Repeat([]byte{byte(padding)}, padding)
    return append(src, padtext...)
}

func UnPad(src []byte) []byte {
    length := len(src)
    unpadding := int(src[length-1])
    return src[:(length - unpadding)]
}

func Base64Encode(input []byte) ([]byte) {
    return []byte(base64.StdEncoding.EncodeToString(input))
}

func Base64Decode(input []byte) ([]byte, error) {
    b, err := base64.StdEncoding.DecodeString(string(input))
    return []byte(b), err
}

func AESEncrypt(plaintext []byte, key []byte) ([]byte, []byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, nil, err
    }
    paddedPlaintext := Pad(plaintext, aes.BlockSize)
    ciphertext := make([]byte, len(paddedPlaintext))
    iv, err := RandomBytes(aes.BlockSize)
    if err != nil {
        return nil, nil, err
    }

    mode := cipher.NewCBCEncrypter(block, iv)
    mode.CryptBlocks(ciphertext, paddedPlaintext)

    return ciphertext, iv, nil
}

func AESDecrypt(ciphertext []byte, iv []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    if len(ciphertext) % aes.BlockSize != 0 {
        return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
    }

    if len(iv) != aes.BlockSize {
        return nil, fmt.Errorf("iv is wrong size")
    }

    paddedPlaintext := make([]byte, len(ciphertext))
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(paddedPlaintext, ciphertext)

    return UnPad(paddedPlaintext), nil
}

func GenerateRSAKey() (*rsa.PrivateKey, error) {
    return rsa.GenerateKey(rand.Reader, 2048)
}

func PemEncodeRSAPrivate(key *rsa.PrivateKey) ([]byte) {
    der := x509.MarshalPKCS1PrivateKey(key)
    b := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
    return pem.EncodeToMemory(b)
}

func PemEncodeRSAPublic(key *rsa.PublicKey) ([]byte) {
    der, _ := x509.MarshalPKIXPublicKey(key)
    b := &pem.Block{Type: "RSA PUBLIC KEY", Bytes: der}
    return pem.EncodeToMemory(b)
}

func PemDecodeRSAPrivate(in []byte) (*rsa.PrivateKey, error) {
    b, _ := pem.Decode(in)
    return x509.ParsePKCS1PrivateKey(b.Bytes)
}

func PemDecodeRSAPublic(in []byte) (*rsa.PublicKey, error) {
    b, _ := pem.Decode(in)
    pubKey, err := x509.ParsePKIXPublicKey(b.Bytes)
    return pubKey.(*rsa.PublicKey), err
}

func RSAEncrypt(plaintext []byte, publicKey *rsa.PublicKey) ([]byte, error) {
    label := []byte("")
    hash := sha256.New()
    return rsa.EncryptOAEP(hash, rand.Reader, publicKey, plaintext, label)
}

func RSADecrypt(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
    label := []byte("")
    hash := sha256.New()
    return rsa.DecryptOAEP(hash, rand.Reader, privateKey, ciphertext, label)
}

/*********************************************************************
 * Main functions (with encodings)
 *********************************************************************/

func GroupEncrypt(plaintext string,  publicKeys map[string]string) (*Encrypted, error) {

    keySize := 32
    key, _ := RandomBytes(keySize)
    ciphertext, iv, err := AESEncrypt([]byte(plaintext), key)
    if err != nil {
        return nil, err
    }
    inputs := make(map[string]string)
    inputs["iv"] = string(Base64Encode(iv))

    encryptedKeys := make(map[string]string)
    for id, publicKeyString := range publicKeys {
        publicKey, err := PemDecodeRSAPublic([]byte(publicKeyString))
        encryptedKey, err := RSAEncrypt(key, publicKey)
        if err != nil {
            return nil, err
        }
        encryptedKeys[id] = string(Base64Encode(encryptedKey))
    }

    return &Encrypted { Ciphertext: string(Base64Encode(ciphertext)), Mode: "aes-cbc-256+rsa", Inputs: inputs, Keys: encryptedKeys }, nil
}

func GroupDecrypt(encrypted *Encrypted, keyID string, privateKeyPem string) (string, error) {

    if encrypted.Mode != "aes-cbc-256+rsa" {
        return "", fmt.Errorf("Invalid mode '%s'", encrypted.Mode)
    }

    ciphertext, _ := Base64Decode([]byte(encrypted.Ciphertext))
    iv, _ := Base64Decode([]byte(encrypted.Inputs["iv"]))
    encryptedKey, _ := Base64Decode([]byte(encrypted.Keys[keyID]))

    privateKey, err := PemDecodeRSAPrivate([]byte(privateKeyPem))
    key, err := RSADecrypt(encryptedKey, privateKey)

    plaintext, err := AESDecrypt(ciphertext, iv, key)
    return string(plaintext), err
}

func Sign(message string, privateKey string) (*Signed, error) {
    return nil, nil
}

func Verify(message *Signed, publicKey string) (bool, error) {
    return false, nil
}
