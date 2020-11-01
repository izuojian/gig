package config

import (
	"gopkg.in/ini.v1"
)

type File = ini.File

// 当前环境的全部配置
var AllConfigs *File

// SetConfigs
func SetConfigs(c *File) {
	AllConfigs = c
}

// ParseIniFile 解析Ini文件
func ParseIniFile(file string) (*File, error) {
	return ini.Load(file)
}

// IniFileToMap 把文件内容直接解析到Struct
func IniFileToMap(v, source interface{}, others ...interface{}) error {
	return ini.MapTo(v, source, others...)
}

// Map 把Section内容解析到Struct
func Map(section string, v interface{}) error {
	if AllConfigs == nil {
		return nil
	}
	return AllConfigs.Section(section).MapTo(v)
}

// String 获取string配置值
func String(section, key string) string {
	if AllConfigs == nil {
		return ""
	}
	return AllConfigs.Section(section).Key(key).String()
}

// Int 获取int配置值
func Int(section, key string) (val int) {
	if AllConfigs == nil {
		return 0
	}
	val, _ = AllConfigs.Section(section).Key(key).Int()
	return
}
