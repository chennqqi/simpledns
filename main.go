package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/chennqqi/goutils/consul"
	"github.com/immortal/logrotate"
)

var (
	gconsul *consul.ConsulApp
)

func main() {
	var consulUrl string
	flag.StringVar(&consulUrl, "conf", "consul://127.0.0.1:8300", "set configure path or consul uri")
	flag.Parse()

	var cfg Config
	app, err := consul.NewAppWithCfgEx(&cfg, "", consulUrl)
	if err != nil {
		logrus.Errorf("")
	}
	gconsul = app

	switch strings.ToUpper(cfg.LogLevel) {
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	case "WARN", "WARNING":
		logrus.SetLevel(logrus.WarnLevel)
	case "DEBUG", "DBG":
		logrus.SetLevel(logrus.DebugLevel)
	default:
		fallthrough
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	}

	logfile, err := logrotate.New(cfg.LogFile, 86400, 7, 0, false)
	if cfg.LogFile != strings.ToLower("console") && err == nil {
		logrus.SetOutput(logfile)
	}

	var server Server
	err = server.Init(&cfg)
	if err != nil {
		fmt.Println("Init Server ERROR:", err)
		return
	}
	go http.ListenAndServe(cfg.HealthHost, nil)
	go server.Run()

	app.Wait(func(s os.Signal) {
		server.Shutdown(context.Background())
		if logfile != nil {
			logfile.Close()
		}
	})
}
