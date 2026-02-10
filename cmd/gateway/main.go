package main

import (
	"Flux-KV/internal/config"
	"Flux-KV/internal/gateway/handler"
	"Flux-KV/internal/gateway/router"
	"Flux-KV/pkg/client"
	"Flux-KV/pkg/discovery"
	"Flux-KV/pkg/logger"
	"Flux-KV/pkg/tracer"
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 引入 Pprof，自动注册路由
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	// 1. 初始化配置系统
	config.InitConfig()
	config.PrintConfig()

	// 2. 初始化日志
	logger.InitLogger()
	// 程序退出前刷新日志缓冲区，防止日志丢失
	defer logger.Log.Sync()

	// 初始化分布式链路追踪
	jaegerEndpoint := viper.GetString("jaeger.endpoint")
	tp, err := tracer.InitTracer("gateway-service", jaegerEndpoint)
	if err != nil {
		logger.Log.Error("❌ Failed to init tracer", zap.Error(err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Log.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}()

	// 获取全局 Logger 实例
	log := logger.Log
	log.Info("🚀 Gateway is starting...")

	// 3. 设置 Gin 的运行模式
	gin.SetMode(viper.GetString("server.mode"))

	// 4. 连接 Etcd 获取服务发现
	etcdEndpoints := viper.GetStringSlice("etcd.endpoints")
	log.Info("🔍 Connecting to Etcd...", zap.Strings("endpoints", etcdEndpoints))

	disco, err := discovery.NewDiscovery(etcdEndpoints)
	if err != nil {
		log.Fatal("❌ Failed to connect to Etcd", zap.Error(err))
	}
	defer disco.Close() // 退出时关闭 Etcd 连接

	// 5. 初始化支持负载均衡的 gRPC Client
	serviceName := "kv-service"
	log.Info("🔗 Initializing KV Client (Load Balanced)...", zap.String("service", serviceName))

	kvClient, err := client.NewClient(disco, serviceName)
	if err != nil {
		log.Fatal("❌ Failed to init KV client", zap.Error(err))
	}
	defer func() {
		log.Info("🔌 Closing gRPC client connections...")
		if err := kvClient.Close(); err != nil {
			log.Error("Failed to close gRPC connection", zap.Error(err))
		}
	}()

	// 6. 初始化 Handlers (控制层)
	kvHandler := handler.NewKVHandler(kvClient)
	healthHandler := handler.NewHealthHandler()

	// 7. 初始化 Router (路由层)
	r := router.NewRouter(kvHandler, healthHandler)

	// 8. 条件启动 Pprof 监控服务（通过环境变量/配置控制）
	if viper.GetBool("pprof.enabled") {
		pprofPort := viper.GetInt("pprof.port")
		pprofAddr := fmt.Sprintf("0.0.0.0:%d", pprofPort)
		go func() {
			log.Info("📈 Pprof Debug Server is running",
				zap.String("addr", fmt.Sprintf("http://localhost:%d/debug/pprof/", pprofPort)))

			// http.ListenAndServe 使用默认的 ServeMux
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				log.Error("❌ Pprof Server failed", zap.Error(err))
			}
		}()
	} else {
		log.Info("⚙️  Pprof Debug Server is disabled (set FLUX_PPROF_ENABLED=true to enable)")
	}

	// 9. 配置 HTTP Server
	gatewayPort := viper.GetInt("gateway.port")
	if gatewayPort == 0 {
		gatewayPort = viper.GetInt("server.port") // 回退到 server.port
	}
	portStr := fmt.Sprintf("%d", gatewayPort)
	srv := &http.Server{
		Addr:    ":" + portStr,
		Handler: r,
	}

	// 10. 启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("❌ Listen error", zap.Error(err))
		}
	}()
	log.Info("✅ Gateway running", zap.String("port", portStr))

	// 11. 优雅退出
	quit := make(chan os.Signal, 1)
	// 监听中断信号 (Ctrl+C, Docker stop)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞直到收到信号
	<-quit
	log.Info("⚠️ Shutting down gateway...")

	// 创建一个 5 秒超时的 Context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭服务器，处理完当前的请求
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ Server forced to shutdown", zap.Error(err))
	}

	log.Info("👋 Gateway exited properly")
}
