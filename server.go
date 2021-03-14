package main

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gocarina/gocsv"
)

var (
	URL = "http://www.vpngate.net/api/iphone"
)

type Server struct {
	HostName                string `csv:"#HostName"`
	IP                      string `csv:"IP"`
	Score                   int    `csv:"Score"`
	Ping                    string `csv:"Ping"`
	Speed                   int    `csv:"Speed"`
	CountryLong             string `csv:"CountryLong"`
	CountryShort            string `csv:"CountryShort"`
	NumVPNSessions          string `csv:"NumVpnSessions"`
	Uptime                  int    `csv:"Uptime"`
	TotalUsers              int    `csv:"TotalUsers"`
	TotalTraffic            int    `csv:"TotalTraffic"`
	LogType                 string `csv:"LogType"`
	Operator                string `csv:"Operator"`
	Message                 string `csv:"Message"`
	OpenVPNConfigDataBase64 string `csv:"OpenVPN_ConfigData_Base64"`
}

func GetServers() ([]Server, error) {
	res, err := http.Get(URL)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	// // テスト用
	// b, err := ioutil.ReadFile("test.csv")
	// if err != nil {
	// 	panic(err)
	// }

	b = bytes.TrimPrefix(b, []byte("*vpn_servers\r\n"))
	b = bytes.TrimSuffix(b, []byte("*\r\n"))

	var servers []Server
	if err := gocsv.Unmarshal(bytes.NewReader(b), &servers); err != nil {
		panic(err)
	}

	return servers, nil
}
