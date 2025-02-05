package main

import (
	"fmt"
	"net"
	"os"
)

const socketPath = "/Users/tt/work/projects/wukongIM/go/go-pdk/examples/test/example.sock"

func main() {
	// 确保旧的 socket 文件被删除，防止端口冲突
	os.Remove(socketPath)

	// 监听 Unix Socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	// 设置 socket 文件权限，允许其他用户访问
	os.Chmod(socketPath, 0777)

	fmt.Println("Unix socket server listening at", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		// 处理连接
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	fmt.Println("Received:", string(buffer[:n]))

	// 回复客户端
	conn.Write([]byte("Message received"))
}
