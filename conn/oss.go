package conn

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var bucketClient *oss.Bucket
var HostOSS = ""

func NewOss(endpoint, bucket string) {
	if len(HostOSS) == 0 {
		HostOSS = fmt.Sprintf("https://%v.%v.aliyuncs.com/", bucket, endpoint)
	}

	if ENV == "local" || ENV == "local-docker" {
		endpoint = endpoint + ".aliyuncs.com"
	} else {
		endpoint = endpoint + "-internal.aliyuncs.com"
	}

	client, err := oss.New(endpoint, AliyunID, AliyunSecret)
	if err == nil {
		bucketClient, err = client.Bucket(bucket)
		if err == nil {
			return
		}
	}
	panic("conn oss failed")
}

func NewPublicOSS(endpoint, bucket string) (bucketClient *oss.Bucket) {
	if len(HostOSS) == 0 {
		HostOSS = fmt.Sprintf("https://%v.%v.aliyuncs.com/", bucket, endpoint)
	}

	endpoint = endpoint + ".aliyuncs.com"

	client, err := oss.New(endpoint, AliyunID, AliyunSecret)
	if err == nil {
		bucketClient, err = client.Bucket(bucket)
		if err == nil {
			return
		}
	}
	panic("conn oss failed")
}

func GetOSS() *oss.Bucket {
	return bucketClient
}

func GetOssObject(key string) ([]byte, error) {
	r, err := GetOSS().GetObject(key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func SaveOssObject(key string, b []byte) string {
	_ = bucketClient.PutObject(key, bytes.NewReader(b))
	return HostOSS + key
}
