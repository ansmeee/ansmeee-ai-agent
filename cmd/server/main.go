package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ansmeee-ai-agent/internal/agent"
	"ansmeee-ai-agent/internal/config"
	"ansmeee-ai-agent/internal/llm"
	"ansmeee-ai-agent/internal/memory"
	"ansmeee-ai-agent/internal/router"
	"ansmeee-ai-agent/internal/tool"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger.
	logger, err := zap.NewProduction()
	if cfg.Server.Mode == "debug" {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("starting server",
		zap.Int("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
	)

	// Initialize GORM DB with master/slave.
	gormDB, err := openGORM(cfg)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	logger.Info("database connected", zap.String("master", cfg.MySQL.Master.Host))

	// Initialize agent store.
	agentStore, err := agent.NewAgentStoreWithDB(gormDB)
	if err != nil {
		logger.Fatal("failed to init agent store", zap.Error(err))
	}
	logger.Info("agent store initialized")

	// Initialize session store (MySQL or fallback to memory/redis).
	sessionStore, err := initSessionStore(cfg, gormDB, logger)
	if err != nil {
		logger.Fatal("failed to init session store", zap.Error(err))
	}
	defer sessionStore.Close()

	// Initialize LLM provider.
	llmProvider, err := llm.New(&cfg.LLM)
	if err != nil {
		logger.Fatal("failed to init LLM provider", zap.Error(err))
	}

	// Initialize tool registry.
	reg := tool.NewRegistry()
	reg.Register(&tool.Calculator{})
	reg.Register(&tool.DateTime{})
	reg.Register(&tool.WebSearch{})
	logger.Info("tools registered", zap.Int("count", len(reg.List())))

	// Initialize agent engine.
	cb := agent.NewCallback(logger)
	engine := agent.New(llmProvider, reg, sessionStore, cb)
	logger.Info("agent engine initialized")

	// Initialize model config store.
	modelConfigStore := llm.NewModelConfigStore(gormDB)
	logger.Info("model config store initialized")

	// Setup router and start server.
	r := router.Setup(cfg, logger, sessionStore, engine, reg, agentStore, modelConfigStore, gormDB)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	go func() {
		logger.Info("listening", zap.String("addr", addr))
		if err := r.Run(addr); err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down", zap.String("signal", sig.String()))
}

func openGORM(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQL.Master.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open master: %w", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.MySQL.Master.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.Master.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MySQL.Master.ConnMaxLifetime)

	slaveDSN := cfg.MySQL.Slave.DSN()
	if slaveDSN != "" && slaveDSN != cfg.MySQL.Master.DSN() {
		if err := db.Use(dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{mysql.Open(cfg.MySQL.Master.DSN())},
			Replicas: []gorm.Dialector{mysql.Open(slaveDSN)},
			Policy:   dbresolver.RandomPolicy{},
		})); err != nil {
			return nil, fmt.Errorf("register dbresolver: %w", err)
		}
	}
	return db, nil
}

func initSessionStore(cfg *config.Config, gormDB *gorm.DB, logger *zap.Logger) (memory.SessionStore, error) {
	switch cfg.Memory.Type {
	case "mysql":
		logger.Info("using MySQL session store")
		return memory.NewMySQLStore(gormDB)
	case "redis":
		logger.Info("using Redis memory backend", zap.String("addr", cfg.Redis.Addr))
		store, err := memory.NewRedis(&cfg.Redis, &cfg.Memory)
		if err != nil {
			logger.Warn("Redis unavailable, falling back to in-memory", zap.Error(err))
			return memory.NewInMemory(&cfg.Memory), nil
		}
		return store, nil
	default:
		logger.Info("using in-memory session store")
		return memory.NewInMemory(&cfg.Memory), nil
	}
}
