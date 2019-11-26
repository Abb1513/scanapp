// @Time    :  2019/11/20
// @Software:  GoLand
// @File    :  main.go
// @Author  :  Abb1513

package main

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"reflect"
	"scanapp/conf"
	"strconv"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/fsnotify/fsnotify"

	"github.com/prometheus/common/log"
)

type Applications struct {
	Application []struct {
		Name     string `xml:"name"`
		Instance []struct {
			App    string `xml:"app"`
			IpAddr string `xml:"ipAddr"`
			Port   struct {
				Text string `xml:",chardata"`
			} `xml:"port"`
		} `xml:"instance"`
	} `xml:"application"`
}

var con conf.Config

func main() {
	con = conf.GetConfig()
	log.Infof("  配置获取 ===> eurekaUrl: %s, consulUrl: %s, mtList: %s, ftList: %s,", con.EurekaUrl, con.ConsulUrl, con.MtList, con.FtList)
	//设置监听回调函数
	conf.V.OnConfigChange(func(e fsnotify.Event) {
		con = conf.GetConfig()
		log.Infof(" 配置变更 ===> eurekaUrl: %s, consulUrl: %s, mtList: %s, ftList: %s,", con.EurekaUrl, con.ConsulUrl, con.MtList, con.FtList)
	})
	//开始监听
	conf.V.WatchConfig()
	body := getEurekaApi(con.EurekaUrl)
	getApp(body)
	log.Info("Start Ing")
	ticker := time.Tick(time.Hour * 6)
	for {
		select {
		case <-ticker:
			body := getEurekaApi(con.EurekaUrl)
			getApp(body)
		}
	}
}

func getEurekaApi(urls string) []byte {
	// 创建请求对象
	log.Debug("getEurekaApi, ", urls)
	response, err := http.Get(urls)
	if err != nil {
		// handle error
	}
	//程序在使用完回复后必须关闭回复的主体。
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	return body
}

func getApp(x []byte) {
	var app Applications
	err := xml.Unmarshal(x, &app)
	if err != nil {
		log.Fatal("解析xml 错误", err)
	}
	application := app.Application
	//log.Info(application)
	for _, v := range application {
		// 并发注册
		for _, k := range v.Instance {
			log.Debugf("name:%s IP: %s, Port: %s", k.App, k.IpAddr, k.Port.Text)
			go registerConsul(k.App, k.IpAddr, k.Port.Text)
		}
	}
}

func registerConsul(App string, ipAddr string, Port string) {
	//"id": "mysql",
	//"name": "mysql",
	//"address": "192.168.1.50",
	//"port": 3306,
	//"tags": ["blackbox"],
	var tag []string
	log.Debug("registerConsul 获取到mtList: ", con.MtList)
	log.Debug("registerConsul 获取到ftList: ", con.FtList)
	log.Debugf("获取到Application ==>name: %s ip: %s:%s", App, ipAddr, Port)
	if Contain(ipAddr, con.MtList) {
		tag = append(tag, "mt")
	} else if Contain(ipAddr, con.FtList) {
		tag = append(tag, "ft")
	}
	// 创建实例
	c := consul.DefaultConfig()
	log.Debug("consul 地址: ", con.ConsulUrl)
	c.Address = con.ConsulUrl

	client, err := consul.NewClient(c)
	if err != nil {
		log.Fatal("consul 连接失败", err)
	}
	// 实例新的 注册对象
	reg := new(consul.AgentServiceRegistration)
	//  ip
	reg.Address = ipAddr
	// 名称
	reg.Name = "application"
	reg.ID = strings.ToLower(App)
	// 端口
	reg.Port, _ = strconv.Atoi(Port)
	// 标签
	reg.Tags = tag
	// 注册服务
	err = client.Agent().ServiceRegister(reg)
	if err != nil {
		log.Errorln(err)
		log.Errorf("name: %s, ip: %s, port: %d,  注册失败", reg.Name, reg.Address, reg.Port)
	}

}

// 判断obj是否在target中，target支持的类型arrary,slice,map
func Contain(obj interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}
	return false
}
