package gig

import (
	"fmt"
	"github.com/izuojian/gig/config"
	"github.com/izuojian/gig/internal/utils"
	"github.com/izuojian/gig/logs"
	"os"
	"path/filepath"
)

// Gig默认配置
type Config struct {
	AppName string `ini:"app_name"`
	RunMode string `ini:"run_mode"`
	Log     LogConfig
}

// 日志文件配置
type LogConfig struct {
	Dir        string `ini:"dir"`
	File       string `ini:"file"`
	MaxSize    int    `ini:"max_size"`
	MaxAge     int    `ini:"max_age"`
	MaxBackups int    `ini:"max_backups"`
	Compress   bool   `ini:"compress"`
}

var (
	// 系统配置
	GigConfig *Config
	// 项目根目录
	AppPath string
	// 配置文件目录
	LogPath string
)

func init() {
	GigConfig = newGigConfig()

	var err error
	AppPath, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	LogPath = filepath.Join(AppPath, "config")
	appConfigName := "app.ini"
	appConfigFile := filepath.Join(LogPath, appConfigName)
	if utils.FileExists(appConfigFile) {
		err = config.IniFileToMap(GigConfig, appConfigFile)
		if err != nil {
			fmt.Printf("Read config error: %v\n", err)
			os.Exit(1)
		}

		// 读取不同环境的配置
		envConfFile := filepath.Join(LogPath, GigConfig.RunMode+".app.ini")
		if utils.FileExists(envConfFile) {
			config.AllConfigs, err = config.ParseIniFile(envConfFile)
			if err != nil {
				fmt.Printf("Read config error: %v\n", err)
				os.Exit(1)
			}

			// 日志配置
			err = config.Map("log", &GigConfig.Log)
			if err != nil {
				fmt.Printf("Read log config error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// 设置mode
	SetMode(GigConfig.RunMode)

	// 设置系统默认的日志工具
	logs.SetGigLoggger(logs.LogConfig{
		Dir:        GigConfig.Log.Dir,
		File:       GigConfig.Log.File,
		MaxSize:    GigConfig.Log.MaxSize,
		MaxAge:     GigConfig.Log.MaxAge,
		MaxBackups: GigConfig.Log.MaxBackups,
		Compress:   GigConfig.Log.Compress,
	})
}

// 默认配置
func newGigConfig() *Config {
	return &Config{
		AppName: "gig-app",
		RunMode: ProdMode,
		Log: LogConfig{
			Dir:        "",
			File:       "app.log",
			MaxSize:    1024,
			MaxAge:     7,
			MaxBackups: 1,
			Compress:   true,
		},
	}
}
