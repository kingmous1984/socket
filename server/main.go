package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

const (
	DirType int8 = iota
	FileType
	DataType
	EOFType
)

type DataStruct struct {
	FileType int8
	Name     string
	Data     []byte
	IsEOF    bool
}

func main() {
	var (
		oldFile string
		dStruct DataStruct
	)
	bufferSize := flag.Int("b", 4096, "请输入缓存大小")
	port := flag.Int("p", 9999, "请输入端口号")
	flag.Parse() // 解析参数

	currPath, err := os.Getwd()
	if err != nil {
		log.Fatal("os.Getwd,", err)
	}
	toPath := filepath.Join(currPath, "data")
	host := fmt.Sprintf("localhost:%d", *port)
	netListen, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatal("listen start error")
	}
	defer netListen.Close()
	log.Printf("listen start,port:%d\n", *port)
	buffer := make([]byte, getBufferSize(*bufferSize))

	for {
		conn, err := netListen.Accept()
		if err != nil {
			log.Printf("%s connect error\n", conn.RemoteAddr())
			continue
		}
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("conn.Read file data error,", err)
				break
			}

			if len(buffer) == 0 {
				continue
			}
			err = json.Unmarshal(buffer[:n], &dStruct)
			if err != nil {
				log.Println("json.Unmarshal, error:", err)
				continue
			}

			if dStruct.FileType == EOFType {
			} else if dStruct.FileType == DirType {
				tmpPath := filepath.Join(toPath, dStruct.Name)
				err := os.MkdirAll(tmpPath, 0766)
				if err != nil {
					fmt.Println("os.MkdirAll error,", err)
					continue
				}
			} else {
				// global.DataType
				filePath := filepath.Join(toPath, dStruct.Name)
				tmpPath := filepath.Dir(filePath)
				if err = os.MkdirAll(tmpPath, 0766); err != nil {
					fmt.Println("os.MkdirAll error,", err)
					continue
				}
				if oldFile != filePath {
					oldFile = filePath
				}
				fileD, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE, 0666)
				if err != nil {
					fmt.Println("os.OpenFile error,", err)
					continue
				}

				// if string(dStruct.Data) == io.EOF.Error() {
				// 	goto LABEL
				// }
				_, err = fileD.Write(dStruct.Data)
				if err != nil {
					fmt.Println("fileD.Write file data error,", err)
					continue
				}
				fileD.Close()
			}
			// LABEL:
			if err = sendToClient(conn, "OK"); err != nil {
				fmt.Println("sendToClient error,", err)
				continue
			}
		}

	}

}

func getBufferSize(bufferSize int) int {
	if bufferSize < 1024 {
		return 1024 * 2
	}
	if bufferSize <= 4096 {
		return bufferSize * 2
	}
	return bufferSize + 4096
}
func sendToClient(conn net.Conn, info string) error {
	_, err := conn.Write([]byte(info))
	return err
}
