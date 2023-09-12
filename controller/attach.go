package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func init() {
	os.MkdirAll("./mnt", os.ModePerm)
}

func Upload(c *gin.Context) {
	UploadLocal(c)
}

//  1、存储位置  ./mnt  确保已经创建好
//  2、url格式 /mnt/xxxxx.png  需要确保网络能访问

func UploadLocal(c *gin.Context) {
	//  获取上传的源文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		RespFail(c.Writer, err.Error())
		return
	}
	defer file.Close()

	// 创建一个新文件
	suffix := filepath.Ext(header.Filename)
	filename := generateFileName(suffix)
	newFilePath := filepath.Join("./mnt", filename)
	newFile, err := os.Create(newFilePath)
	if err != nil {
		zap.L().Error(err.Error())
		return
	}
	defer newFile.Close()
	// 复制上传文件的内容到新文件
	_, err = io.Copy(newFile, file)
	if err != nil {
		zap.L().Error(err.Error())
		return
	}
	url := "mnt/" + filename
	RespOk(c.Writer, url, "")
}

func generateFileName(suffix string) string {
	return fmt.Sprintf("%d%04d%s", time.Now().Unix(), rand.Int31(), suffix)
}
