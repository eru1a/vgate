package main

import (
	"os"
	"strings"
)

func main() {
	country := "JP"
	if len(os.Args) > 1 {
		country = os.Args[1]
	}

	serversAll, err := GetServers()
	if err != nil {
		panic(err)
	}

	var servers []Server
	for _, server := range serversAll {
		if server.Ping == "-" {
			continue
		}
		switch {
		case country == strings.ToLower("ALL"),
			strings.ToLower(country) == strings.ToLower(server.CountryShort),
			strings.ToLower(country) == strings.ToLower(server.CountryLong):
			servers = append(servers, server)
		}
	}

	app := NewApp(servers)
	app.Run()
}
