package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/olekukonko/tablewriter"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
)

func translateOverallStatus(status string) string {
	switch status {
	case "gray":
		return "未知(可能是裸机状态)"
	case "green":
		return "正常"
	case "yellow":
		return "警告"
	case "red":
		return "异常"
	default:
		return status
	}
}

func translatePowerState(state string) string {
	switch state {
	case "poweredOn":
		return "开机"
	case "poweredOff":
		return "关机"
	case "standBy":
		return "待机"
	default:
		return state
	}
}

// 创建 ESXi URL 链接实例
func CreateESXiURL(esxiHost string, esxiUser string, esxiPassword string) *url.URL {
	esxiURLString := fmt.Sprintf("https://%s/sdk", esxiHost)

	esxiURL, esxiURLParseError := url.Parse(esxiURLString)
	if esxiURLParseError != nil {
		fmt.Println("URL 解析失败:", esxiURLParseError)
		os.Exit(1) // TODO: 强制退出是有点问题的, 以后再把错误集中处理, 先等项目搭建完毕
	}

	esxiURL.User = url.UserPassword(esxiUser, esxiPassword)

	return esxiURL
}

// 打印主机表格
func PrintHostInfoTable(esxiHost string, esxiUser string, esxiPassword string) {
	// 创建上下文实例
	contextRoot := context.Background()

	// 创建 ESXi URL 链接实例
	esxiURL := CreateESXiURL(esxiHost, esxiUser, esxiPassword)

	// 创建 Govmomi 客户端实例
	govmomiClient, govmomiClientError := govmomi.NewClient(contextRoot, esxiURL, true)
	if govmomiClientError != nil {
		fmt.Println("连接失败:", govmomiClientError)
		os.Exit(1)
	}
	defer govmomiClient.Logout(contextRoot)

	// 创建 Finder 资源查找器实例
	resourceFinder := find.NewFinder(govmomiClient.Client, true)
	datacenter, datacenterError := resourceFinder.Datacenter(contextRoot, "ha-datacenter") // 裸 ESXi 默认数据中心名称
	if datacenterError != nil {
		fmt.Println("找不到默认数据中心:", datacenterError)
		os.Exit(1)
	}
	resourceFinder.SetDatacenter(datacenter)

	// 获取默认主机实例
	hostSystem, hostSystemError := resourceFinder.DefaultHostSystem(contextRoot)
	if hostSystemError != nil {
		fmt.Println("主机查找失败:", hostSystemError)
		os.Exit(1)
	}

	// 读取主机属性
	hostSystemMO := mo.HostSystem{}
	hostSystemPropertyError := govmomiClient.RetrieveOne(
		contextRoot,
		hostSystem.Reference(),
		[]string{
			"name",      // 主机名字
			"summary",   // 主机摘要
			"datastore", // 挂在存储
		},
		&hostSystemMO,
	)
	if hostSystemPropertyError != nil {
		fmt.Println("读取主机属性失败:", hostSystemPropertyError)
		os.Exit(1)
	}

	// 打印主机状态表格
	data := [][]string{
		{"主机名称", hostSystemMO.Name},
		{"总体状态", translateOverallStatus(string(hostSystemMO.Summary.OverallStatus))},
		{"电源状态", translatePowerState(string(hostSystemMO.Summary.Runtime.PowerState))},
		{"中央处理", strconv.Itoa(int(hostSystemMO.Summary.Hardware.CpuMhz)) + " MHz"},
		{"内存大小", strconv.FormatInt(hostSystemMO.Summary.Hardware.MemorySize/1024/1024/1024, 10) + " GB"},
		{"存储数量", strconv.Itoa(len(hostSystemMO.Datastore)) + " 个磁盘"},
	}

	for _, dsRef := range hostSystemMO.Datastore {
		ds := mo.Datastore{}
		err := govmomiClient.RetrieveOne(contextRoot, dsRef, []string{"summary"}, &ds)
		if err != nil {
			fmt.Println("获取 datastore 信息失败:", err)
			continue
		}
		capacityGB := ds.Summary.Capacity / 1024 / 1024 / 1024
		freeGB := ds.Summary.FreeSpace / 1024 / 1024 / 1024
		uncommittedGB := capacityGB - freeGB
		diskInfo := fmt.Sprintf("总容量: %d GB, 可用空间: %d GB, 已用空间: %d GB", capacityGB, freeGB, uncommittedGB)
		data = append(data, []string{ds.Summary.Name, diskInfo})
	}

	table := tablewriter.NewWriter(os.Stdout)

	table.Header([]string{"字段", "对应值"})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

// 获取环境变量中的密码
func getEnvsPassword() string {
	// 加载当前目录下的 .env 文件
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("加载 .env 文件失败:", err)
		os.Exit(1)
	}

	esxiPassword := os.Getenv("ESXI_PASSWORD") // ESXi 密码
	if esxiPassword == "" {
		fmt.Println("ESXI_PASSWORD 未设置")
		os.Exit(1)
	}

	return esxiPassword
}

func main() {
	// 设置 ESXi 主机基本信息

	esxiHost := "10.10.174.151"       // ESXi 主机地址
	esxiUser := "root"                // ESXi 用户名
	esxiPassword := getEnvsPassword() // ESXi 密码
	PrintHostInfoTable(esxiHost, esxiUser, esxiPassword)
}

// TODO: 构建一个完整的架构过程

/**
 * 操作系统控制器接口
 */
type SystemController interface {
	// 获取系统信息
	GetSystemInfo()
}

/**
 * 虚拟机控制器接口
 */
type VMController interface {
	// 创建虚拟机
	CreateVMSystem()
}

/**
 * 工具接口
 */
type Tools interface {
	// 获取某个环境变量
	GetEnv()
}
