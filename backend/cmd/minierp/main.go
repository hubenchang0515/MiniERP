package main

import (
	"context"
	"errors"
	"flag"
	"github.com/gin-gonic/gin"
	"log"
	"minierp/internal/config"
	"minierp/internal/database"
	"minierp/internal/service"
	transport "minierp/internal/transport/http"
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
	if os.Getenv("MINIERP_JWT_SECRET") == "" {
		log.Printf("警告: 未设置 MINIERP_JWT_SECRET；实际密钥来源: %s。正式部署建议通过环境变量设置随机密钥。", c.SecretSource)
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
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Print(err)
		}
	}()
	log.Printf("MiniERP 已启动: http://%s；数据目录: %s", c.Addr, c.DataDir)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	if ctx.Err() != nil {
		<-shutdownDone
	}
}
