package etcd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Header represents the nested "header" object in the JSON
type Header struct {
	ClusterID string `json:"cluster_id"`
	MemberID  string `json:"member_id"`
	Revision  string `json:"revision"`
	RaftTerm  string `json:"raft_term"`
}
type Member struct {
	ID         string   `json:"ID"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
	ClientURLs []string `json:"clientURLs"`
}

// MemberReply represents the entire JSON structure from /memberlist
type MemberReply struct {
	Header  Header   `json:"header"`
	Members []Member `json:"members"`
}

// LeaderReply represents the entire JSON structure from maintain/status
type LeaderReply struct {
	Header           Header `json:"header"`
	Version          string `json:"version"`
	DBSize           string `json:"dbSize"`
	Leader           string `json:"leader"`
	RaftIndex        string `json:"raftIndex"`
	RaftTerm         string `json:"raftTerm"`
	RaftAppliedIndex string `json:"raftAppliedIndex"`
	DBSizeInUse      string `json:"dbSizeInUse"`
}

func GetLeader() (string, error) {
	leaderId := getLeaderID()
	if leaderId != "" {
		m := getMembers()
		return GetLeaderIP(m, leaderId)
	}
	return "", fmt.Errorf("no leader found")
}

func GetLeaderIP(members []Member, leaderID string) (string, error) {
	for _, member := range members {
		if member.ID == leaderID {
			curl := member.ClientURLs[0]
			nurl := strings.Replace(curl, "https://", "", -1)
			strs := strings.Split(nurl, ":")
			if len(strs) == 2 {
				return strs[0], nil
			}
		}
	}
	return "", fmt.Errorf("leader IP not found")
}

func getMembers() []Member {

	memURL := "https://127.0.0.1:2379/v3/cluster/member/list"

	body := HttpsPOST(memURL)
	if body == nil {
		return nil
	}
	// 输出响应
	//fmt.Printf("Response: %s\n", body)

	member := MemberReply{}
	err := json.Unmarshal(body, &member)
	if err != nil {
		fmt.Printf("Error parse HTTP response to LeaderReply")
		return nil
	}
	fmt.Printf("members: %v\n", member.Members)
	return member.Members
}

func getLeaderID() string {

	// etcd集群的URL
	etcdURL := "https://127.0.0.1:2379/v3/maintenance/status"

	body := HttpsPOST(etcdURL)

	// 输出响应
	//fmt.Printf("Leader Response: %s\n", body)
	if body == nil {
		return ""
	}
	leader := LeaderReply{}
	err := json.Unmarshal(body, &leader)
	if err != nil {
		fmt.Printf("Error parse HTTP response to LeaderReply")
		return ""
	}
	fmt.Printf("Leader ID: %s\n", leader.Leader)
	return leader.Leader
}

func HttpsPOST(url string) []byte {
	caCertPath := "/var/lib/rancher/k3s/server/tls/etcd/server-ca.crt"
	// 客户端证书和私钥
	clientCertPath := "/var/lib/rancher/k3s/server/tls/etcd/client.crt"
	clientKeyPath := "/var/lib/rancher/k3s/server/tls/etcd/client.key"

	// 加载CA证书
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		fmt.Printf("Error loading CA certificate: %v\n", err)
		return nil
	}

	// 创建CA证书池并添加CA证书
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// 加载客户端证书和私钥
	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		fmt.Printf("Error loading client certificate/key: %v\n", err)
		return nil
	}

	// 创建HTTP客户端并设置TLS配置
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      caCertPool,
			},
		},
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return nil
	}

	// 发送HTTP请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending HTTP request: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading HTTP response: %v\n", err)
		return nil
	}
	return body
}
