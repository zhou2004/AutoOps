// 启动程序
// author xiaoRui

package main

import (
	"context"
	"dodevops-api/api/cmdb/controller"
	databaseService "dodevops-api/api/database/service"
	monitorService "dodevops-api/api/monitor/service"
	"dodevops-api/api/task/service"
	webService "dodevops-api/api/webservice/service"
	"dodevops-api/common"
	"dodevops-api/common/config"
	_ "dodevops-api/docs"
	"dodevops-api/pkg/db"
	"dodevops-api/pkg/log"
	"dodevops-api/pkg/redis"
	"dodevops-api/pkg/websecret"
	"dodevops-api/router"
	"dodevops-api/scheduler"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// @title devops运维管理系统
// @version 1.0
// @description devops运维管理系统API接口文档
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// 解析命令行参数
	var configPath string
	var mode string
	var databaseWorker string
	flag.StringVar(&mode, "mode", "api", "api、database-worker、database-collector、web-worker、web-collector 或 web-key-init")
	flag.StringVar(&databaseWorker, "database-worker", os.Getenv("DATABASE_WORKER_MODE"), "数据库任务执行器：auto（debug 模式内置）、embedded 或 external")
	flag.StringVar(&configPath, "c", "", "配置文件路径 (默认: ./config.yaml)")
	flag.Parse()
	if mode == "web-key-init" {
		location, created, err := websecret.Initialize()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("Web key ready: %s (created=%t); back up this key separately from the database.\n", location, created)
		return
	}

	// 加载配置文件
	if err := config.LoadConfig(configPath); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	if mode == "web-worker" || mode == "web-collector" {
		_ = common.InitRedis()
		if err := db.SetupDBLink(); err != nil {
			panic(err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		host, _ := os.Hostname()
		if mode == "web-collector" {
			webService.New(common.GetDB()).RunCollector(ctx)
		} else {
			webService.New(common.GetDB()).RunWorker(ctx, fmt.Sprintf("%s-%d", host, os.Getpid()))
		}
		return
	}
	if mode == "database-worker" || mode == "database-collector" {
		if err := db.SetupDBLink(); err != nil {
			panic(err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		host, _ := os.Hostname()
		if mode == "database-collector" {
			databaseService.New(common.GetDB()).RunCollector(ctx)
			return
		}
		databaseService.New(common.GetDB()).RunWorker(ctx, fmt.Sprintf("%s-%d", host, os.Getpid()))
		return
	}
	if mode != "api" {
		panic("unsupported mode")
	}
	embeddedWorker, err := config.EmbeddedDatabaseWorker(databaseWorker, config.Config.Server.Model)
	if err != nil {
		panic(err)
	}
	// 先占用监听端口，避免启动失败的实例继续迁移数据库和执行后台任务。
	listener, err := net.Listen("tcp", config.Config.Server.Address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "API 启动失败：无法监听 %s，请检查端口是否已被占用：%v\n", config.Config.Server.Address, err)
		os.Exit(1)
	}
	defer listener.Close()

	// 初始化其他服务
	if err := initServices(); err != nil {
		panic("Failed to initialize services: " + err.Error())
	}

	collectorCtx, stopCollector := context.WithCancel(context.Background())
	defer stopCollector()
	go monitorService.RunDomainCertChecks(collectorCtx)
	webEmbedded, err := config.EmbeddedDatabaseWorker(os.Getenv("WEB_WORKER_MODE"), config.Config.Server.Model)
	if err != nil {
		panic(err)
	}
	if webEmbedded {
		host, _ := os.Hostname()
		go webService.New(common.GetDB()).RunWorker(collectorCtx, fmt.Sprintf("web-dev-%s-%d", host, os.Getpid()))
	} else {
		go webService.New(common.GetDB()).RunCollector(collectorCtx)
	}
	if embeddedWorker {
		host, _ := os.Hostname()
		log.Log().Info("数据库任务执行器随开发 API 启动；排队任务将自动领取")
		go databaseService.New(common.GetDB()).RunWorker(collectorCtx, fmt.Sprintf("dev-%s-%d", host, os.Getpid()))
	} else {
		go databaseService.New(common.GetDB()).RunCollector(collectorCtx)
	}

	// 启动调度器管理器
	if err := scheduler.GetManager().Start(); err != nil {
		log.Log().Errorf("Failed to start scheduler manager: %v", err)
	}

	// 初始化任务队列系统
	if err := initTaskQueue(); err != nil {
		log.Log().Errorf("Failed to initialize task queue: %v", err)
	}

	log := log.Log()
	gin.SetMode(config.Config.Server.Model)
	router := router.InitRouter()
	srv := &http.Server{
		Addr:    config.Config.Server.Address,
		Handler: router,
	}
	// 启动服务
	serveErrors := make(chan error, 1)
	go func() {
		log.Info("Conflicting values for 'process.env.NODE_ENV'")
		log.Info("")
		log.Info("  App running at:")
		log.Info("  - Local:   http://" + config.Config.Server.Address)
		log.Info("  - Network: http://" + config.Config.Server.Address)
		log.Info("")
		log.Info("  请注意，开发版本尚未优化")
		log.Info("  要创建生产环境构建，请运行 go run main.go")
		log.Info("")
		if config.Config.Server.EnableSwagger {
			log.Info("API文档地址: http://" + config.Config.Server.Address + "/swagger/index.html")
		}
		serveErrors <- srv.Serve(listener)
	}()
	quit := make(chan os.Signal, 1)
	//监听消息
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)
	var serveErr error
	select {
	case <-quit:
	case serveErr = <-serveErrors:
		if serveErr != nil && serveErr != http.ErrServerClosed {
			log.Errorf("API 服务异常退出: %v", serveErr)
		}
	}
	log.Info("Shutdown Server ...")
	stopCollector()

	// 停止调度器管理器
	scheduler.GetManager().Stop()

	// 停止任务队列系统
	stopTaskQueue()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Infof("Server Shutdown: %v", err)
	}
	log.Info("Server exiting")
	if serveErr != nil && serveErr != http.ErrServerClosed {
		os.Exit(1)
	}
}

// initServices 初始化各种服务
func initServices() error {
	// 初始化数据库连接和自动迁移
	if err := db.SetupDBLink(); err != nil {
		return err
	}

	// 初始化Redis
	if err := common.InitRedis(); err != nil {
		log.Log().Warnf("Redis initialization failed: %v", err)
	}
	redis.SetupRedisDb()

	// 初始化SQL记录控制器
	controller.InitCmdbSQLRecordController(common.GetDB())

	return nil
}

// initTaskQueue 初始化任务队列系统
func initTaskQueue() error {
	// 导入任务服务包
	taskService := service.NewTaskWorkService()
	_ = taskService // 避免未使用警告

	// 初始化全局调度器
	if err := service.InitGlobalScheduler(); err != nil {
		return fmt.Errorf("初始化全局调度器失败: %v", err)
	}

	// 初始化任务队列
	config := service.DefaultConfig()
	if err := service.InitTaskQueue(config); err != nil {
		return fmt.Errorf("初始化任务队列失败: %v", err)
	}

	log.Log().Info("任务队列系统初始化成功")
	return nil
}

// stopTaskQueue 停止任务队列系统
func stopTaskQueue() {
	log.Log().Info("停止任务队列系统...")

	// 停止全局调度器
	service.StopGlobalScheduler()

	// 停止任务队列
	service.StopTaskQueue()

	log.Log().Info("任务队列系统已停止")
}
