package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/orangbus/m3u8/dl"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	url      string
	output   string
	chanSize int
	name     string
	fileName string
)

func init() {
	flag.StringVar(&url, "u", "", "M3U8 URL, required")
	flag.IntVar(&chanSize, "c", 25, "Maximum number of occurrences")
	flag.StringVar(&output, "o", "out", "Output folder, required")
	flag.StringVar(&name, "n", "", "out filename,default timestamp")
	flag.StringVar(&fileName, "f", "", "download filename format: http:xxx.m3u8 filename.mp4")
}

func main() {
	flag.Parse()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[error]", r)
			os.Exit(-1)
		}
	}()

	if fileName != "" {
		if err := downloadFile(); err != nil {
			log.Printf("下载错误：%s", err.Error())
			return
		}
		log.Println("下载完成")
		return
	}
	downloadOne()
}

func panicParameter(name string) {
	panic("parameter '" + name + "' is required")
}

func downloadOne() {
	if url == "" {
		panicParameter("u")
	}
	if name != "" {
		// 获取文件明
		ext := filepath.Ext(name)
		if ext == "" {
			name = fmt.Sprintf("%s.mp4", name)
		}
	} else {
		name = fmt.Sprintf("%d.mp4", time.Now().Unix())
	}
	if chanSize <= 0 {
		panic("parameter 'c' must be greater than 0")
	}
	downloader, err := dl.NewTask(output, url)
	if err != nil {
		panic(err)
	}
	if err := downloader.Start(chanSize, name); err != nil {
		panic(err)
	}
	fmt.Println("Done!")
}

func downloadFile() error {
	if !strings.HasPrefix(fileName, "/") {
		basePath, _ := os.Getwd()
		fileName = filepath.Join(basePath, fileName)
	}

	info, err := os.Stat(fileName)
	if err != nil {
		return err
	}
	ext := filepath.Ext(info.Name())
	ouPath := info.Name()[:len(info.Name())-len(ext)] + "_out"
	if _, err := os.Stat(ouPath); err != nil {
		if err2 := os.MkdirAll(ouPath, os.ModePerm); err2 != nil {
			return err2
		}
	}

	// 获取文件
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var start int64 // 开始
	tmpFile, err := os.OpenFile(fileName+".tmp", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	numStr, err := io.ReadAll(tmpFile)
	if err != nil {
		start = 0
	}
	if len(numStr) > 0 {
		num, err := strconv.Atoi(string(numStr))
		if err == nil {
			start = int64(num)
		}
	}
	if start > 0 {
		log.Printf("下载开始位置第 %d 行", start)
	}

	// 读取每一行
	var line string
	var total int64
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024) // 初始缓冲区大小为 64KB
	scanner.Buffer(buf, 1024*1024)  // 最大缓冲区设为 1MB
	for scanner.Scan() {
		total++
		if total <= start {
			continue
		}
		line = scanner.Text()
		line = strings.TrimSpace(line)
		items := strings.Split(line, ".m3u8")
		if len(items) == 2 {
			downloadName := strings.TrimSpace(items[1])
			downloadUrl := strings.TrimSpace(items[0] + ".m3u8")
			if err := startDownload(downloadUrl, downloadName, ouPath); err != nil {
				log.Printf("下载错误:%s", err.Error())
			}
		}
		tmpFile.Seek(0, 0)
		tmpFile.WriteString(fmt.Sprintf("%d", total))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func startDownload(url, name, outDir string) error {
	downloader, err := dl.NewTask(outDir, url)
	if err != nil {
		return err
	}
	if err := downloader.Start(chanSize, name); err != nil {
		return err
	}
	return nil
}
