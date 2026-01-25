package client

import (
	pb "Go-AI-KV-System/api/proto"
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client 封装了 gRPC 的连接和客户端存根
type Client struct {
	conn *grpc.ClientConn
	rpcClient pb.KVServiceClient
}

// NewClient 创建了一个新的 gRPC 客户端连接
func NewClient(address string) (*Client, error) {
	// 1. 建立 gRPC 连接
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	// 2. 创建生成的客户端存根 (Stub)
	c := pb.NewKVServiceClient(conn)

	// 3. 返回封装后的 Client 实例
	return &Client{
		conn: conn,
		rpcClient: c,
	}, nil
}

// Close 关闭底层连接
func (c *Client) Close() error {
	return c.conn.Close()
}

// Set 封装 Set 请求
func (c *Client) Set(key, value string) error {
	// 设置一秒超时，防止网络卡死
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	req := &pb.SetRequest{
		Key: key,
		Value: value,
	}

	// 调用远程方法
	_, err := c.rpcClient.Set(ctx, req)
	return err
}

// Get 封装 Get 请求
func (c *Client) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	req := &pb.GetRequest{
		Key: key,
	}

	resp, err := c.rpcClient.Get(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

// Del 封装 Del 请求
func (c *Client) Del(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	req := &pb.DelRequest{
		Key: key,
	}

	_, err := c.rpcClient.Del(ctx, req)
	return err
}

