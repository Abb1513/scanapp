// @Time    :  2019/11/21
// @Software:  GoLand
// @File    :  conf.go
// @Author  :  Abb1513

package conf

import (
	"github.com/prometheus/common/log"
	"github.com/spf13/viper"
)

var V = viper.New()

func init() {
	V.SetConfigName("config")
	V.AddConfigPath(".")
	V.SetConfigType("yml")
	if err := V.ReadInConfig(); err != nil {
		panic(err)
	}

}

type Config struct {
	EurekaUrl string
	ConsulUrl string
	MtList    []string
	FtList    []string
}

func GetConfig() Config {
	var Con Config
	err := V.Unmarshal(&Con)
	if err != nil {
		log.Error("解析配置失败, ", err)
	}
	return Con
}
