package main

import (
	"context"
	"errors"
	"flag"
	"github.com/gin-gonic/gin"
	"log"
	"minioa/internal/config"
	"minioa/internal/database"
	"minioa/internal/service"
	transport "minioa/internal/transport/http"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	c, err := config.Load(os.Args[1:], os.Getenv, os.Stdout)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("MINIOA_JWT_SECRET") == "" {
		log.Printf("警告: 未设置 MINIOA_JWT_SECRET；实际密钥来源: %s。正式部署建议通过环境变量设置随机密钥。", c.SecretSource)
	}
	db, err := database.Open(c.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	reader, err := database.OpenReader(c.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	readPool, _ := reader.DB()
	defer readPool.Close()
	gin.SetMode(gin.ReleaseMode)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	s := &service.Service{DB: db, Reader: reader, Config: c}
	if err := s.RecoverAIQuota(ctx); err != nil {
		log.Fatalf("恢复 AI 周额度账本失败: %v", err)
	}
	s.StartUsageWriter()
	defer s.StopUsageWriter()
	s.StartResourceMonitor(ctx)
	server := &http.Server{Addr: c.Addr, Handler: transport.Router(s), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	shutdownDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		defer close(shutdownDone)
		const waitForRequests = 10 * time.Second
		shutdown, cancel := context.WithTimeout(context.Background(), waitForRequests)
		defer cancel()
		switch err := server.Shutdown(shutdown); {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("MiniOA 关闭超时：等待现有 HTTP 请求结束超过 %s", waitForRequests)
		case err != nil:
			log.Printf("MiniOA 关闭服务失败：%v", err)
		}
	}()
	log.Printf("MiniOA 已启动: http://%s；数据目录: %s", c.Addr, c.DataDir)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	if ctx.Err() != nil {
		<-shutdownDone
	}
}
