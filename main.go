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
	"github.com/chennqqi/goutils/consul.v2"
	"github.com/chennqqi/goutils/yamlconfig"
	"github.com/immortal/logrotate"
)

var (
	gconsul *consul.ConsulApp
)

func main() {
	var conf string
	flag.StringVar(&conf, "conf", "consul://127.0.0.1:8500", "set configure path or consul uri")
	flag.Parse()

	var cfg Config
	var app *consul.ConsulApp
	var err error
	if strings.HasPrefix(conf, "consul://") {
		app, err = consul.NewConsulAppWithCfg(&cfg, conf)
		if err != nil {
			logrus.Errorf("[main.go::main] NewConsulAppWithCfg error: %v", err)
			return
		}
	} else {
		err := yamlconfig.Load(&cfg, conf)
		if err != nil {
			logrus.Errorf("[main.go::main] yamlconfig.Load error: %v", err)
			return
		}

		//ignore ping error
		app, _ = consul.NewConsulApp("consul://127.0.0.1:8500")
		app.Fix()
		app.Ping()
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
	logrus.Println("CFG:", cfg)

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
