package main

import (
	"context"
	"fmt"
	"log"

	"ecommerce-demo/app/user/pb"

	// 重点：必须用原生 grpc 连接，不能用 zrpc 连接
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. 原生 gRPC 直连（适配你的 pb 文件）
	conn, err := grpc.Dial(
		"127.0.0.1:8080", // 你的 RPC 端口
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("连接 RPC 失败: %v", err)
	}
	defer conn.Close()

	// 2. 创建客户端（完全匹配你的 pb）
	client := pb.NewUserClient(conn)

	// ------------------------------
	// 3. 测试注册接口
	// ------------------------------
	fmt.Println("====== 正在测试注册接口 ======")
	regResp, err := client.Register(context.Background(), &pb.RegisterReq{
		Username: "test_user_01",
		Password: "password123",
	})
	if err != nil {
		log.Printf("注册失败 (可能已存在): %v\n", err)
	} else {
		fmt.Printf("注册成功! 用户ID: %d\n", regResp.Id)
	}

	// ------------------------------
	// 4. 测试登录接口
	// ------------------------------
	fmt.Println("\n====== 正在测试登录接口 ======")
	loginResp, err := client.Login(context.Background(), &pb.LoginReq{
		Username: "test_user_01",
		Password: "password123",
	})
	if err != nil {
		log.Fatalf("登录失败: %v\n", err)
	}
	fmt.Printf("登录成功! 用户ID: %d\n", loginResp.Id)

	// ------------------------------
	// 5. 测试错误密码
	// ------------------------------
	fmt.Println("\n====== 测试错误密码 ======")
	_, err = client.Login(context.Background(), &pb.LoginReq{
		Username: "test_user_01",
		Password: "wrong_password",
	})
	if err != nil {
		fmt.Printf("预期错误: %v\n", err)
	}
}
