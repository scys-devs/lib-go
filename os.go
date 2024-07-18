package lib

import (
	"archive/zip"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func FileName(s string) string {
	return strings.TrimSuffix(path.Base(s), path.Ext(s))
}

func Bash(cli string) ([]byte, error) {
	cmd := exec.Command("/bin/bash", "-c", cli)
	return cmd.CombinedOutput()
}

// 根据文件修改时间，遍历指定目录，根据os.ReadDir，修改了排序
func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ModTime().Unix() < list[j].ModTime().Unix()
	})
	return list, nil
}

// TODO 兼容下 绝对路径
func Zip(src string, target string) {
	// 预防：旧文件无法覆盖
	os.RemoveAll(target)

	// 创建：zip文件
	zipfile, _ := os.Create(target)
	defer zipfile.Close()

	// 打开：zip文件
	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	// 遍历路径信息
	_ = filepath.Walk(src, func(path string, info os.FileInfo, _ error) error {
		// 如果是win，那么替换下路径吧
		if runtime.GOOS == "windows" {
			path = strings.ReplaceAll(path, `\`, `/`)
		}
		// 如果是源路径，提前进行下一个遍历
		if path == src {
			return nil
		}
		// 获取：文件头信息
		header, _ := zip.FileInfoHeader(info)
		header.Name = strings.TrimPrefix(path, src+`/`)
		// 判断：文件是不是文件夹
		if info.IsDir() {
			header.Name += `/`
		} else {
			// 设置：zip的文件压缩算法
			header.Method = zip.Deflate
		}

		// 创建：压缩包头部信息
		writer, _ := archive.CreateHeader(header)
		if !info.IsDir() {
			file, _ := os.Open(path)
			defer file.Close()
			io.Copy(writer, file)
		}
		return nil
	})
}

func FileExist(path string) bool {
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}
