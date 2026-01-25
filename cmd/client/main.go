package main

import (
	"Go-AI-KV-System/pkg/client"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// 1. 连接服务器
	serverAddr := "127.0.0.1:50051"
	fmt.Printf("🚀 正在连接 gRPC 服务端 [%s]...\n", serverAddr)

	cli, err := client.NewClient(serverAddr)
	if err != nil {
		fmt.Printf("❌ 连接失败: %v\n", err)
		return
	}
	defer cli.Close()

	fmt.Println("✅ 连接成功! (输入 'exit' 或 'quit' 退出)")
	fmt.Println("------------------------------------------------")

	// 2. 启动交互式循环
	reader := bufio.NewReader(os.Stdin)

	for {
		// 打印提示符
		fmt.Print("Go-KV> ")

		// 读取用户输入的一行
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		// 处理空输入
		if text == "" {
			continue
		}

		// 解析命令
		parts := strings.Fields(text)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "SET":
			if len(parts) < 3 {
				fmt.Println("⚠️  用法: SET <key> <value>")
				continue
			}
			key := parts[1]
			// 支持 value 中带空格 (例如: SET msg hello world)
			val := strings.Join(parts[2:], " ")
			
			err := cli.Set(key, val)
			if err != nil {
				fmt.Printf("❌ SET 错误: %v\n", err)
			} else {
				fmt.Println("OK")
			}
		
		case "GET":
			if len(parts) != 2 {
				fmt.Println("⚠️  用法: GET <key>")
				continue
			}
			key := parts[1]
			val, err := cli.Get(key)
			if err != nil {
				fmt.Printf("❌ GET 错误: %v\n", err)
			} else {
				// 模仿 Redis，输出加上引号
				fmt.Printf("\"%s\"\n", val)
			}

		case "DEL":
			if len(parts) != 2 {
				fmt.Println("⚠️  用法: DEL <key>")
				continue
			}
			key := parts[1]
			err := cli.Del(key)
			if err != nil {
				fmt.Printf("❌ DEL 错误: %v\n", err)
			} else {
				fmt.Println("(integer) 1") // 模仿 Redis 风格
			}

		case "EXIT", "QUIT":
			fmt.Println("👋 Bye!")
			return

		default:
			fmt.Printf("❌ 未知命令: %s\n", cmd)
		}
	}
}