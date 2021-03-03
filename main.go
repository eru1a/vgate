package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"text/tabwriter"

	"github.com/gocarina/gocsv"
)

var (
	URL = "http://www.vpngate.net/api/iphone"
)

type Server struct {
	HostName                string `csv:"#HostName"`
	IP                      string `csv:"IP"`
	Score                   string `csv:"Score"`
	Ping                    string `csv:"Ping"`
	Speed                   string `csv:"Speed"`
	CountryLong             string `csv:"CountryLong"`
	CountryShort            string `csv:"CountryShort"`
	NumVPNSessions          string `csv:"NumVpnSessions"`
	Uptime                  string `csv:"Uptime"`
	TotalUsers              string `csv:"TotalUsers"`
	TotalTraffic            string `csv:"TotalTraffic"`
	LogType                 string `csv:"LogType"`
	Operator                string `csv:"Operator"`
	Message                 string `csv:"Message"`
	OpenVPNConfigDataBase64 string `csv:"OpenVPN_ConfigData_Base64"`
}

func main() {
	country := "JP"

	allServers, err := getServers()
	if err != nil {
		panic(err)
	}

	var servers []Server
	for _, server := range allServers {
		if server.CountryShort == country {
			servers = append(servers, server)
		}
	}

	server := choseServer(servers)

	if err := Connect(server.OpenVPNConfigDataBase64); err != nil {
		panic(err)
	}
}

func choseServer(servers []Server) Server {
	var index int
	for {
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
		w.Write([]byte("Index\tCountry\tIP\tPing\tSpeed\tScore\n"))
		for i, server := range servers {
			w.Write([]byte(fmt.Sprintf("%d\t%s\t%s\t%s\t%s\t%s\n", i,
				server.CountryShort, server.IP, server.Ping, server.Speed, server.Score)))
		}
		w.Flush()
		fmt.Printf("choose index: ")
		_, err := fmt.Scanln(&index)
		if err != nil {
			continue
		}
		if index >= 0 && index < len(servers) {
			break
		}
	}
	return servers[index]
}

func getServers() ([]Server, error) {
	// res, err := http.Get(URL)
	// if err != nil {
	// 	panic(err)
	// }
	// defer res.Body.Close()
	// b, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	panic(err)
	// }

	// テスト用
	b, err := ioutil.ReadFile("test.csv")
	if err != nil {
		panic(err)
	}

	b = bytes.TrimPrefix(b, []byte("*vpn_servers\r\n"))
	b = bytes.TrimSuffix(b, []byte("*\r\n"))

	var servers []Server
	if err := gocsv.Unmarshal(bytes.NewReader(b), &servers); err != nil {
		panic(err)
	}

	return servers, nil
}

func Connect(openVPNConfigDataBase64 string) error {
	conf, err := base64.StdEncoding.DecodeString(openVPNConfigDataBase64)
	if err != nil {
		return err
	}

	f, err := os.Create("/tmp/openvpnconf")
	if err != nil {
		return err
	}
	defer f.Close()
	defer os.Remove("/tmp/openvpnconf")

	_, err = f.Write(conf)
	if err != nil {
		return err
	}

	cmd := exec.Command("sudo", "openvpn", "/tmp/openvpnconf")
	cmd.Stdout = os.Stdout

	signal.Ignore(os.Interrupt)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
