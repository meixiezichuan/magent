package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/meixiezichuan/magent/etcd"
)

type Addr struct {
	IP   string
	Port int
}

func parseAddressList(input string) ([]Addr, error) {
	// 分割逗号分隔的地址
	rawAddresses := strings.Split(input, ",")
	var results []Addr

	for _, rawAddr := range rawAddresses {
		// 清理空白字符
		trimmed := strings.TrimSpace(rawAddr)
		if trimmed == "" {
			continue
		}

		// 解析单个地址
		addr, err := parseSingleAddress(trimmed)
		if err != nil {
			return nil, fmt.Errorf("无效地址 %q: %w", trimmed, err)
		}

		results = append(results, addr)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到有效地址")
	}

	return results, nil
}

// parseSingleAddress 解析单个 IP:port 地址
func parseSingleAddress(addrStr string) (Addr, error) {
	// 尝试分割最后一次出现的冒号（支持IPv6地址）
	lastColon := strings.LastIndex(addrStr, ":")
	if lastColon == -1 {
		return Addr{}, fmt.Errorf("缺少端口分隔符")
	}

	host := addrStr[:lastColon]
	portStr := addrStr[lastColon+1:]

	// 解析端口号
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return Addr{}, fmt.Errorf("无效端口 %q: %w", portStr, err)
	}

	// 验证端口范围
	if port < 1 || port > 65535 {
		return Addr{}, fmt.Errorf("端口 %d 超出范围 (1-65535)", port)
	}

	// 验证IP地址格式
	if ip := net.ParseIP(host); ip == nil {
		// 尝试解析主机名
		if _, err := net.LookupHost(host); err != nil {
			return Addr{}, fmt.Errorf("无效IP/主机名 %q", host)
		}
	}

	return Addr{IP: host, Port: port}, nil
}

func createOrUpdateIPVSService(virtualIP, realIP string) error {
	// 清除现有的IPVS服务
	cmd := exec.Command("ipvsadm", "-C")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear existing IPVS rules: %v", err)
	}

	// 添加虚拟IPVS服务
	cmd = exec.Command("ipvsadm", "-A", "-t", virtualIP, "-s", "rr")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add IPVS service: %v", err)
	}

	// 添加真实服务器
	cmd = exec.Command("ipvsadm", "-a", "-t", virtualIP, "-r", realIP, "-m")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add real server: %v", err)
	}

	return nil
}

func main() {
	//fmt.Printf("os.Args: %v", os.Args)
	if len(os.Args) < 2 {
		fmt.Println("Usage: program <ip:port,ip:port,...>")
		return
	}
	virtualIPs := os.Args[1]

	addrs, err := parseAddressList(virtualIPs)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
		return
	}

	//fmt.Println("成功解析的地址列表:")
	//for i, addr := range addrs {
	//	fmt.Printf("%d. IP: %s, Port: %d\n", i+1, addr.IP, addr.Port)
	//}

	for {
		leaderIP, err := etcd.GetLeader()
		if err != nil {
			fmt.Printf("Error finding leader IP: %v\n", err)
			continue
		}
		fmt.Println("Get leaderIP: ", leaderIP)
		for _, addr := range addrs {
			virtualIP := addr.IP + ":" + strconv.Itoa(addr.Port)
			realIP := leaderIP + ":" + strconv.Itoa(addr.Port)
			fmt.Println("realIP: ", realIP)
			err = createOrUpdateIPVSService(virtualIP, realIP)
			if err != nil {
				fmt.Printf("Error creating IPVS service: %v\n", err)
			} else {
				fmt.Println("IPVS service updated successfully")
			}
		}

		// 设定检查间隔，例如每5秒检查一次
		time.Sleep(30 * time.Second)
	}
}
