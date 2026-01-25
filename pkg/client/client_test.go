package client

import (
	"Go-AI-KV-System/internal/config"
	"Go-AI-KV-System/internal/core"
	"Go-AI-KV-System/internal/service"
	pb "Go-AI-KV-System/api/proto"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// TestKVServiceFlow 会模拟启动一个服务器，然后创建一个客户端去连接它
func TestKVServiceFlow(t *testing.T) {
	// 1. 启动服务端

	// 1.1 准备基础设施：使用 :0 让系统自动分配空闲端口
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v ", err)
	}
	// 获取系统实际分配的端口
	port := lis.Addr().String()
	t.Logf("Test Server started on port: %s", port)

	// 1.2 创建 gRPC 服务器，注册 KV 服务
	s := grpc.NewServer()
	db := core.NewMemDB(&config.Config{})

	kvService := service.NewKVService(db)

	pb.RegisterKVServiceServer(s, kvService)

	// 1.3 goroutine 中启动服务（不阻塞测试主线程）
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("failed to serve: %v", err)
		}
	}()
	defer s.Stop()	// 测试结束自动停止服务，释放资源

	time.Sleep(50 * time.Millisecond)

	// 2. 启动客户端

	cli, err := NewClient(port)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer cli.Close()

// ==========================================
	// 3. 执行测试用例 (Assert)
	// ==========================================

	// 测试 SET
	key := "my_name"
	val := "Naato"
	t.Logf("Testing SET %s = %s", key, val)

	err = cli.Set(key, val)
	if err != nil {
		t.Fatalf("❌ SET command failed: %v", err)
	}

	// 测试 GET
	t.Logf("Testing GET %s", key)
	got, err := cli.Get(key)
	if err != nil {
		t.Fatalf("❌ GET command failed: %v", err)
	}

	// 验证结果
	if got != val {
		t.Errorf("❌ Verification Failed! Expected '%s', but got '%s'", val, got)
	} else {
		t.Logf("✅ Success! Got expected value: %s", got)
	}

	// 测试 DEL (新增，Day 7 完整性测试)
	t.Logf("Testing DEL %s", key)
	if err := cli.Del(key); err != nil {
		t.Fatalf("❌ DEL command failed: %v", err)
	}

	// 验证删除是否生效
	gotAfterDel, err := cli.Get(key)
	// gRPC 的 Get 如果找不到 key，可能会返回空字符串，或者 error，具体看你的 Server 实现
	// 这里假设返回空字符串表示没有找到
	if gotAfterDel != "" {
		t.Errorf("❌ DEL failed! Key still exists with value: %s", gotAfterDel)
	} else {
		t.Log("✅ DEL Success")
	}
}