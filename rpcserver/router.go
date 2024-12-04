package rpcserver

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/didip/tollbooth/v7/limiter"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/indexer"
	"github.com/sat20-labs/indexer/rpcserver/base"
	"github.com/sat20-labs/indexer/rpcserver/bitcoind"
	"github.com/sat20-labs/indexer/rpcserver/extension"
	"github.com/sat20-labs/indexer/rpcserver/ord"
	"github.com/sat20-labs/indexer/rpcserver/ordx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	STRICT_TRANSPORT_SECURITY   = "strict-transport-security"
	CONTENT_SECURITY_POLICY     = "content-security-policy"
	CACHE_CONTROL               = "cache-control"
	VARY                        = "vary"
	ACCESS_CONTROL_ALLOW_ORIGIN = "access-control-allow-origin"
	TRANSFER_ENCODING           = "transfer-encoding"
	CONTENT_ENCODING            = "content-encoding"
)

const (
	CONTEXT_TYPE_TEXT = "text/html; charset=utf-8"
	CONTENT_TYPE_JSON = "application/json"
)

type RateLimit struct {
	limit    *limiter.Limiter
	reqCount int
}

type Rpc struct {
	basicService     *base.Service
	ordxService      *ordx.Service
	ordService       *ord.Service
	btcdService      *bitcoind.Service
	extensionService *extension.Service
	apidoc           *APIDoc
}

func NewRpc(baseIndexer *indexer.IndexerMgr, chain string) *Rpc {
	return &Rpc{
		basicService:     base.NewService(baseIndexer),
		ordxService:      ordx.NewService(baseIndexer),
		ordService:       ord.NewService(),
		btcdService:      bitcoind.NewService(),
		extensionService: extension.NewService(chain),
		apidoc:           &APIDoc{},
	}
}

func (s *Rpc) Start(rpcUrl, swaggerHost, swaggerSchemes, rpcProxy, rpcLogFile string, apiConf *config.API) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	var writers []io.Writer
	if rpcLogFile != "" {
		exePath, _ := os.Executable()
		executableName := filepath.Base(exePath)
		if strings.Contains(executableName, "debug") {
			executableName = "debug"
		}
		executableName += ".rpc"
		fileHook, err := rotatelogs.New(
			rpcLogFile+"/"+executableName+".%Y%m%d%H%M.log",
			rotatelogs.WithLinkName(rpcLogFile+"/"+executableName+".log"),
			rotatelogs.WithMaxAge(7*24*time.Hour),
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			return fmt.Errorf("failed to create RotateFile hook, error %s", err)
		}
		writers = append(writers, fileHook)
	}
	writers = append(writers, os.Stdout)
	gin.DefaultWriter = io.MultiWriter(writers...)
	r.Use(logger.SetLogger(
		logger.WithLogger(logger.Fn(func(c *gin.Context, l zerolog.Logger) zerolog.Logger {
			if c.Request.Header["Authorization"] == nil {
				return l
			}
			return l.With().
				Str("Authorization", c.Request.Header["Authorization"][0]).
				Logger()
		})),
	))

	config := cors.Config{
		AllowOrigins: []string{"*", "sat20.org", "ordx.market", "localhost"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		// ExposeHeaders:    []string{"Content-Length"},
		// AllowCredentials: true,
		MaxAge: 12 * time.Hour,
	}
	config.AllowOrigins = []string{"*"}
	config.OptionsResponseStatusCode = 200
	r.Use(cors.New(config))

	// doc
	InitApiDoc(swaggerHost, swaggerSchemes, rpcProxy)
	r.GET(rpcProxy+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// api config
	err := s.apidoc.InitApiConf(apiConf)
	if err != nil {
		return err
	}

	err = s.apidoc.ApplyApiConf(r, rpcProxy)
	if err != nil {
		return err
	}

	// common header
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set(VARY, "Origin")
		c.Writer.Header().Add(VARY, "Access-Control-Request-Method")
		c.Writer.Header().Add(VARY, "Access-Control-Request-Headers")

		c.Writer.Header().Del(CONTENT_SECURITY_POLICY)
		c.Writer.Header().Set(
			CONTENT_SECURITY_POLICY,
			"default-src 'self'",
		)

		c.Writer.Header().Set(
			STRICT_TRANSPORT_SECURITY,
			"max-age=31536000; includeSubDomains; preload",
		)

		c.Writer.Header().Set(
			ACCESS_CONTROL_ALLOW_ORIGIN,
			"*",
		)

		c.Next()
	})

	// // zip encoding
	// r.Use(
	// 	gzip.Gzip(gzip.DefaultCompression,
	// 		gzip.WithExcludedPathsRegexs(
	// 			[]string{
	// 				// `.*\/btc\/.*`,
	// 			},
	// 		),
	// 	),
	// )

	// Compression middleware
	r.Use(CompressionMiddleware())

	// router
	s.basicService.InitRouter(r, rpcProxy)
	s.ordxService.InitRouter(r, rpcProxy)
	s.ordService.InitRouter(r, rpcProxy)
	s.btcdService.InitRouter(r, rpcProxy)
	s.extensionService.InitRouter(r, rpcProxy)

	parts := strings.Split(rpcUrl, ":")
	var port string
	if len(parts) < 2 {
		rpcUrl += ":80"
		port = "80"
	} else {
		port = parts[1]
	}

	// 先检查端口
	if err := checkPort(port); err != nil {
		return err
	}

	go r.Run(rpcUrl)
	return nil
}

func checkPort(port string) error {
	// 方法1: 尝试监听该端口
	addr := fmt.Sprintf(":%s", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %s is in use: %v", port, err)
	}
	l.Close()
	return nil
}
