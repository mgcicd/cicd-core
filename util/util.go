package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	b64 "encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	http "net/http"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

func StructToJson(v interface{}) string {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	data, err := json.Marshal(v)

	if err != nil {
		fmt.Println(err)
		return ""
	}

	return string(data)
}

func JsonToStruct(str string, object interface{}) error {

	if str == "" {
		return errors.New("str is empty")
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Unmarshal([]byte(str), object)

	// return nil
}

func ByteToStruct(data []byte, object interface{}) error {

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	err := json.Unmarshal(data, object)

	return err
}

func StringJoin(args ...string) string {

	if len(args) == 0 {
		return ""
	}
	var buffer bytes.Buffer

	for _, arg := range args {
		buffer.WriteString(arg)
	}

	return buffer.String()
}

//CBC加密
func EncryptdesCbc(src, key string) string {
	data := []byte(src)
	keyByte := []byte(key)
	block, err := des.NewCipher(keyByte)
	if err != nil {
		panic(err)
	}
	data = _PKCS5Padding(data, block.BlockSize()) //获取CBC加密模式
	iv := keyByte                                 //用密钥作为向量(不建议这样使用)
	mode := cipher.NewCBCEncrypter(block, iv)
	out := make([]byte, len(data))
	mode.CryptBlocks(out, data)
	return fmt.Sprintf("%X", out)
}

func _PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//生成32位md5字串
func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

//生成Guid字串
func UniqueId() string {
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return GetMd5String(base64.URLEncoding.EncodeToString(b))
}

func SendEmail(email string, title string, content string, from string, to string, deliverContent string) {

	sBody := fmt.Sprintf(`{"Email":"%s","MessageType":115,"KeyParam":{"Title":"%s","Content":"%s","DeliverPersion":"%s","DeliverContent":"%s","Ps":"@%s"}}`, email, title, content, from, deliverContent, to)

	_, err := doPost(fmt.Sprintf("http://sendmessage:8080/sz/Message/SendMessage"), sBody, "application/json")

	if err != nil {
		fmt.Println(err)
	}
}

func doPost(url string, req string, contentType string) ([]byte, error) {

	if contentType == "" {
		contentType = "application/json"
	}

	response, err := post(url, contentType,
		req)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return response, err
}

func post(url string, contenType string, payload string) ([]byte, error) {

	resp, err := http.Post(url, contenType, strings.NewReader(payload))

	if err != nil {
		fmt.Println("get error", err)
		return nil, err
	}
	//body ,err:= ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	//if err != nil{
	//    fmt.Println("read body",err)
	//    return
	//}
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

type WxContent struct {
	EmpNos  []string `json:"empNos"`
	Content string   `json:"content"`
}

func SendWX(empNos []string, wxTemplate string) {

	postJson := StructToJson(WxContent{EmpNos: empNos, Content: wxTemplate})

	_, err := doPost("https://hr.34580.com/message/api/basic/message/send/0?code=LGJYZUPWIO", postJson, "application/json")

	if err != nil {
		log.Fatal(err)
	}
}

//基于和spring机制一样当存在多个一样的参数在querystring上取数组最后一个参数值为基准
func GetUrlQueryStringByLastOne(key string, params map[string][]string) string {

	if key == "" {
		return ""
	}

	if params == nil {
		return ""
	}

	value := params[key]

	if value == nil {
		return ""
	}

	if len(value) == 0 {
		return ""
	}

	return value[len(value)-1]
}

func Sha1String(data string) string {
	sha1 := sha1.New()
	sha1.Write([]byte(data))
	return hex.EncodeToString(sha1.Sum([]byte(nil)))
}

// key BASE64之后的AES密钥
//AES加密算法，默认会加密后的字符串进行Base64转码
func AESEncrypt(key string, requestBody string) (string, error) {

	//fmt.Println("AES加密 Base64解码之前的AES Key 为:", key)
	decodeKey, err := b64.StdEncoding.DecodeString(key)
	//fmt.Println("AES加密 Base64解码之后的AES Key为:", string(decodeKey))

	c, err := aes.NewCipher([]byte(string(decodeKey)))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)

	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	//text := StructToJson(requestBody)

	seal := gcm.Seal(nonce, nonce, []byte(requestBody), nil)
	sEnc := b64.StdEncoding.EncodeToString(seal)
	return sEnc, nil
}

// key  :  AES算法的密钥(被base64之后的值)
// sEnc : AES加密之后的数据
func AESDecrypt(key string, sEnc string) (string, error) {
	fmt.Println("待解密的内容 :", sEnc)
	ciphertext, err := b64.StdEncoding.DecodeString(sEnc)
	fmt.Println("base64解密后的内容 :", sEnc)
	if err != nil {
		fmt.Printf("error ....., %+v\n ", err)
		return "", err
	}
	fmt.Println("Base64解码之前的AES Key 为:", key)
	decodeKey, err := b64.StdEncoding.DecodeString(key)
	fmt.Println("Base64解码之后的AES Key为:", string(decodeKey))
	if err != nil {
		return "", err
	}
	c, err := aes.NewCipher([]byte(string(decodeKey)))

	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Println(err)
	}
	return string(plaintext), nil
}
