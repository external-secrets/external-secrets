/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implieclient.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"fmt"
)

func DecryptAES(secret string, passphrase string) (string, error) {
	return parseSecrets(secret, passphrase)
}

func getMD5Hash(text string) []byte {
	hash := md5.Sum([]byte(text))
	return hash[:]
}
func bytesToKey(data []byte, salt []byte, output int32) (finalOutput []byte) {
	merged := string(data) + string(salt)
	output = 48
	finalKey := getMD5Hash(merged)
	key := finalKey
	for len(finalKey) < int(output) {
		key = getMD5Hash(string(key) + merged)
		finalKey = []byte(string(finalKey) + string(key))
	}
	finalOutput = finalKey[0:output]
	return 
}

func parseSecrets(ciphertext string, passcode string) (string, error) {
	decodedText, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	salted := decodedText[0:8]
	if string(salted) != "Salted__" {
		return "", fmt.Errorf("invalid encrypted data")
	}
	salt := decodedText[8:16]
	keyIv := bytesToKey([]byte(passcode), salt, 48)
	key := keyIv[:32]
	iv := keyIv[32:]
	plain := decrypt(key, decodedText[16:], iv)
	return string(plain), nil
}

func decrypt(key []byte, ciphertext []byte, iv []byte) []byte {
	newCipher, _ := aes.NewCipher(key)
	cfbdec := cipher.NewCBCDecrypter(newCipher, iv)
	decipher := make([]byte, len(ciphertext))
	cfbdec.CryptBlocks(decipher, ciphertext)
	decipher = PKCS5UnPadding(decipher)
	return decipher
}

func PKCS5UnPadding(b64 []byte) []byte {
	last := b64[len(b64)-1]
	if last > 16 {
		return b64
	}
	return b64[:len(b64)-int(last)]
}