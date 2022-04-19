package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	DirType int8 = iota
	FileType
	DataType
	EOFType
)

var (
	bufferSize int
)

type DataStruct struct {
	FileType int8
	Name     string
	Data     []byte
}

func main() {
	ip := flag.String("s", "127.0.0.1", "请输入服务端IP")
	port := flag.Int("p", 9999, "请输入端口号")
	path := flag.String("f", "", "请输入文件或文件夹路径")
	bSize := flag.Int("b", 4096, "请输入缓存大小")
	flag.Parse() // 解析参数

	setBufferSize(*bSize)
	server := fmt.Sprintf("%s:%d", *ip, *port)
	conn, err := getConn(server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s \n", err.Error())
		os.Exit(1)
	}

	if !PathExists(*path) {
		fmt.Println("file or director not exists! error")
		os.Exit(1)
	}

	if IsDir(*path) {
		if err = sendPath(*path, *path, conn); err != nil {
			log.Fatalln(err)
		}
	} else {
		basePath := filepath.Dir(*path)
		if err = sendFile(basePath, *path, conn); err != nil {
			log.Fatalln(err)
		}
	}
}
func setBufferSize(bsize int) {
	if bsize < 1024 {
		bufferSize = 1024
	}
	bufferSize = bsize
}
func getConn(server string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		return nil, errors.New("net.ResolveTCPAddr error," + err.Error())
	}
	return net.DialTCP("tcp", nil, tcpAddr)
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
func IsDir(f string) bool {
	fi, e := os.Stat(f)
	if e != nil {
		return false
	}
	return fi.IsDir()
}

func sendFile(basePath, path string, conn *net.TCPConn) error {
	fileName := strings.TrimLeft(path, basePath)
	buffer := make([]byte, bufferSize)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return err
			}
			data := DataStruct{
				FileType: EOFType,
				Name:     fileName,
			}
			if err = sendData(conn, data, 0); err != nil {
				return err
			}
			break
		}
		data := DataStruct{
			FileType: FileType,
			Name:     fileName,
			Data:     buffer[:n],
		}
		if err = sendData(conn, data, 0); err != nil {
			return err
		}
	}
	return nil
}
func reserve(conn *net.TCPConn) error {
	buffer := make([]byte, 4)
	n, err := conn.Read(buffer)
	if err != nil {
		return err
	}
	if string(buffer[:n]) == "OK" {
		return nil
	}
	return errors.New("retry")
}
func sendPath(basePath, path string, conn *net.TCPConn) error {
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalln("ioutil.ReadDir,path:", path, ",error:", err)
	}

	for _, fi := range fileInfo {
		fmt.Println(fi.Name())
		fileName := filepath.Join(path, fi.Name())
		fmt.Println("fileName=", fileName)
		if fi.IsDir() {
			data := DataStruct{
				FileType: DirType,
				Name:     strings.TrimLeft(fileName, basePath),
			}
			if err = sendData(conn, data, 0); err != nil {
				return err
			}
			if err = sendPath(basePath, fileName, conn); err != nil {
				return err
			}
		} else {
			if err = sendFile(basePath, fileName, conn); err != nil {
				return err
			}
		}
	}
	return nil
}

func sendData(conn *net.TCPConn, data DataStruct, num int8) error {
	dataJson, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte(dataJson))
	if err != nil {
		return err
	}
	if err = reserve(conn); err != nil {
		if err.Error() == "retry" && num < 3 {
			num++
			sendData(conn, data, num)
		}
		return err
	}
	return nil
}
