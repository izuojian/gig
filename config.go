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
	System  SysConfig
}

// 系统配置
type SysConfig struct {
	AccessLog string `ini:"access_log"`
	ErrorLog  string `ini:"error_log"`
}

var (
	// 系统配置
	GigConfig *Config
	// 项目根目录
	AppPath string
	// 配置文件目录
	ConfigPath string

	AccessLogger *logs.Logger
	ErrorLogger  *logs.Logger
)

func init() {
	GigConfig = newGigConfig()

	var err error
	AppPath, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	ConfigPath = filepath.Join(AppPath, "config")
	appConfigFile := filepath.Join(ConfigPath, "app.ini")
	if utils.FileExists(appConfigFile) {
		err = config.IniFileToMap(GigConfig, appConfigFile)
		if err != nil {
			fmt.Printf("Read config error: %v\n", err)
			os.Exit(1)
		}

		// 读取不同环境的配置
		envConfigFile := filepath.Join(ConfigPath, GigConfig.RunMode+".app.ini")
		if utils.FileExists(envConfigFile) {
			config.AllConfigs, err = config.ParseIniFile(envConfigFile)
			if err != nil {
				fmt.Printf("Read config error: %v\n", err)
				os.Exit(1)
			}

			// 系统配置
			err = config.Map("system", &GigConfig.System)
			if err != nil {
				fmt.Printf("Read system config error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// 设置mode
	SetMode(GigConfig.RunMode)

	// 设置系统默认的日志工具
	_initSystemLogger(GigConfig.System.AccessLog, GigConfig.System.ErrorLog)
}

// newGigConfig 默认配置
func newGigConfig() *Config {
	return &Config{
		AppName: "gig-app",
		RunMode: ProdMode,
		System: SysConfig{
			AccessLog: "access.log",
			ErrorLog:  "error.log",
		},
	}
}

// _initSystemLogger 初始化系统使用的日志工具
func _initSystemLogger(accessLogFile, errorLogFile string) {
	file, err := os.Create(accessLogFile)
	if err != nil {
		fmt.Printf("InitAccessLogger failed, err:%v\n", err)
		os.Exit(1)
	}
	AccessLogger = logs.NewLogger(file)

	file, err = os.Create(errorLogFile)
	if err != nil {
		fmt.Printf("InitErrorLogger failed, err:%v\n", err)
		os.Exit(1)
	}
	ErrorLogger = logs.NewLogger(file)
}
