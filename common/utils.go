package common

import (
	"bytes"
	"compress/gzip"
	"io"
	"math/rand"
	"time"

	"github.com/pkg/errors"
)

func GzipData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Flush(); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func RandStr(prefix string, length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	randomSeed := 100
	byteArr := []byte(str)
	result := []byte{}
	/* #nosec */
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(randomSeed)))
	for i := 0; i < length; i++ {
		/* #nosec */
		result = append(result, byteArr[rand.Intn(len(byteArr))])
	}
	return prefix + string(result)
}

func GunzipData(data []byte) ([]byte, error) {
	var r io.Reader
	var err error
	b := bytes.NewBuffer(data)
	r, err = gzip.NewReader(b)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return resB.Bytes(), nil
}
