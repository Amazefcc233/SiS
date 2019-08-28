package data

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/Tnze/CoolQ-Golang-SDK/cqp"
	"path/filepath"
	"time"
)

// Init 初始化插件的数据源，包括读取配置文件、建立数据库连接
func Init() error {
	// 读取配置文件
	err := readConfig()
	if err != nil {
		return fmt.Errorf("读配置文件出错: %v", err)
	}

	// 连接数据库
	err = openDB(filepath.Join(cqp.GetAppDir(), "data.db"))
	if err != nil {
		return fmt.Errorf("打开数据库出错: %v", err)
	}

	// 连接MC服务器
	err = openRCON(Config.RCON.Address, Config.RCON.Password)
	if err != nil {
		return fmt.Errorf("连接RCON出错: %v", err)
	}

	return nil
}

var Config struct {
	// 游戏群
	GroupID int64
	// 管理群
	AdminID int64
	// MC服务器远程控制台
	RCON struct {
		Address  string
		Password string
	}
	// Ping工具配置
	Ping struct {
		DefaultServer string
		Timeout       duration
	}
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func readConfig() error {
	md, err := toml.DecodeFile(filepath.Join(cqp.GetAppDir(), "conf.toml"), &Config)
	if err != nil {
		return err
	}

	// 检查配置文件是否有多余数据，抛警告⚠️
	if uk := md.Undecoded(); len(uk) > 0 {
		cqp.AddLog(cqp.Warning, "Conf", fmt.Sprintf("配置文件中有未知数据: %q", uk))
	}

	return nil
}
