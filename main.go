package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/meixiezichuan/magent/etcd"
)

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
	fmt.Printf("os.Args: %v", os.Args)
	virtualIP := os.Args[1]
	strs := strings.Split(virtualIP, ":")
	if len(strs) != 2 {
		fmt.Printf("Virtual IP shoud in fomat ip:port ")
		return
	}
	port := strs[1]
	for {
		leaderIP, err := etcd.GetLeader()
		if err != nil {
			fmt.Printf("Error finding leader IP: %v\n", err)
			continue
		}
		fmt.Println("Get leaderIP: ", leaderIP)
		realIP := leaderIP + ":" + port
		fmt.Println("realIP: ", realIP)
		err = createOrUpdateIPVSService(virtualIP, realIP)
		if err != nil {
			fmt.Printf("Error creating IPVS service: %v\n", err)
		} else {
			fmt.Println("IPVS service updated successfully")
		}

		// 设定检查间隔，例如每5秒检查一次
		time.Sleep(30 * time.Second)
	}
}
