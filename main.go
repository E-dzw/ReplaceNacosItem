package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	utils "replaceNacos/utils/httpClient"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type NacosNsConfig struct {
	TotalCount     int                 `json:"totalCount"`
	PageNumber     int                 `json:"pageNumber"`
	PagesAvailable int                 `json:"pagesAvailable"`
	PageItems      []NacosNsConfigItem `json:"pageItems"`
}

// 每个ns下的配置项
type NacosNsConfigItem struct {
	ID      string `json:"id"`
	DataId  string `json:"dataId"`
	Group   string `json:"group"`
	Content string `json:"content"`
	Md5     string `json:"md5"`
	Tenant  string `json:"tenant"`
	AppName string `json:"appName"`
	Type    string `json:"type"`
}

// 配置详情
type NacosConfig struct {
	ID         string `json:"id"`
	DataId     string `json:"dataId"`
	Group      string `json:"group"`
	Content    string `json:"content"`
	Md5        string `json:"md5"`
	Tenant     string `json:"tenant"`
	AppName    string `json:"appName"`
	Type       string `json:"type"`
	CreateTime int64  `json:"createTime"`
	ModifyTime int64  `json:"modifyTime"`
	CreateUser string `json:"createUser"`
	CreateIp   string `json:"createIp"`
	Desc       string `json:"desc"`
}

func getNacosNameSpaces(url string) []string {
	apiUrl := "/v1/console/namespaces"
	requestUrl := url + apiUrl
	resp := utils.SendHttpRequest("GET", requestUrl, "", nil)
	var nsNames []string
	var data []map[string]interface{}
	err := json.Unmarshal([]byte(resp), &data)
	if err != nil {
		log.Fatalf("json序列化异常")
	}
	for _, namespace := range data {
		nsName := namespace["namespace"].(string)
		nsNames = append(nsNames, nsName)
	}
	return nsNames
}

// 获取nacos指定namespace下配置信息列表
func getNsConfigList(nacosUrl, tenant string) *NacosNsConfig {
	var apiUri string
	if tenant == "public" {
		apiUri = "/nacos/v1/cs/configs?dataId=&group=&appName=&config_tags=&pageNo=1&pageSize=1000&tenant=&search=accurate"
	} else {
		apiUri = fmt.Sprintf("/nacos/v1/cs/configs?dataId=&group=&appName=&config_tags=&pageNo=1&pageSize=1000&tenant=%s&search=accurate", tenant)
	}
	requestUrl := nacosUrl + apiUri
	resp := utils.SendHttpRequest("GET", requestUrl, "", nil)
	data := new(NacosNsConfig)
	err := json.Unmarshal([]byte(resp), &data)
	if err != nil {
		log.Fatalf("json序列化异常")
	}
	return data
}

// 获取单条配置的详细信息
func getDetailConfig(nacosUrl, dataId, group, tenant string) *NacosConfig {
	var apiUri string
	if tenant == "public" {
		apiUri = fmt.Sprintf("/nacos/v1/cs/configs?show=all&dataId=%s&group=%s&tenant=", dataId, group)
	} else {
		apiUri = fmt.Sprintf("/nacos/v1/cs/configs?show=all&dataId=%s&group=%s&tenant=%s", dataId, group, tenant)
	}
	requestUrl := nacosUrl + apiUri
	resp := utils.SendHttpRequest("GET", requestUrl, "", nil)
	data := new(NacosConfig)
	err := json.Unmarshal([]byte(resp), &data)
	if err != nil {
		log.Print("获取 % dataId下的配置信息异常", dataId)
	}
	return data
}

// 替换配置信息
func configReplace(configDetail *NacosConfig, srcConfig, targetConfig string) string {
	configContent := configDetail.Content
	// if strings.Contains(configContent, srcConfig) {
	// 	configContent = strings.ReplaceAll(configContent, srcConfig, targetConfig)
	// }
	// return configContent
	reg, _ := regexp.Compile(srcConfig)
	return reg.ReplaceAllString(configContent, targetConfig)
}

// 生成要替换的配置
func configDetailGenerate(configDetail *NacosConfig, srcConfig, targetConfig string) map[string]string {
	// 比对并替换B集群中需要特异化的配置信息
	configContent := configReplace(configDetail, srcConfig, targetConfig)
	updateParams := make(map[string]string)
	updateParams["dataId"] = configDetail.DataId
	updateParams["group"] = configDetail.Group
	updateParams["content"] = configContent
	updateParams["type"] = configDetail.Type
	updateParams["id"] = configDetail.ID
	updateParams["md5"] = configDetail.Md5
	updateParams["tenant"] = configDetail.Tenant
	updateParams["appName"] = configDetail.AppName
	return updateParams
}

// 更新并发布Nacos配置
func updateConfig(nacosUrl string, postData map[string]string) bool {
	apiUri := "/nacos/v1/cs/configs"
	requestUrl := nacosUrl + apiUri
	resp := utils.HttpPostWithFormData(requestUrl, postData)
	result, err := strconv.ParseBool(resp)
	if err != nil {
		log.Fatalf("数据类型断言异常")
	}
	return result
}

func main() {
	viper.SetConfigFile("config.toml")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("read config failed: ", err)
	}
	nacosUrl := viper.GetString("url")
	srcConfigString := viper.GetString("srcConfigString")
	targetConfigString := viper.GetString("targetConfigString")
	tenant := viper.GetString("tenant")
	configItemMap := make(map[string]NacosNsConfigItem)
	var count int
	nacosConfigList := getNsConfigList(nacosUrl, tenant)
	//获取当前nacos namespace 所有dataId的切片
	var nacosDataIdSlice []string
	for _, configItem := range nacosConfigList.PageItems {
		dataId := configItem.DataId
		nacosDataIdSlice = append(nacosDataIdSlice, dataId)
		configItemMap[dataId] = configItem
	}
	var dataIdList []string
	for _, dataId := range nacosDataIdSlice {
		group := configItemMap[dataId].Group
		nacosConfig := getDetailConfig(nacosUrl, dataId, group, tenant)
		if strings.Contains(nacosConfig.Content, srcConfigString) {
			updateParams := configDetailGenerate(nacosConfig, srcConfigString, targetConfigString)
			result := updateConfig(nacosUrl, updateParams)
			if result {
				count = count + 1
			}
			dataIdList = append(dataIdList, dataId)
		}
	}
	log.Printf("配置:%s共完成%d处替换,以下dataId发生了变更: %s", targetConfigString, count, dataIdList)
}
