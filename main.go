package main

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/gdamore/tcell/v2"
	"github.com/gocarina/gocsv"
	"github.com/rivo/tview"
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

type App struct {
	*tview.Application
	servers     []Server
	flex        *tview.Flex
	serversView *ServersView
	outputView  *tview.TextView
	cmd         *exec.Cmd
}

func NewApp(servers []Server) *App {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	serversView := NewServersView(servers)
	outputView := tview.NewTextView()
	outputView.SetTitle("output").SetTitleAlign(tview.AlignLeft).SetBorder(true)
	flex.AddItem(serversView, 0, 1, true).
		AddItem(outputView, 0, 1, false)

	app := tview.NewApplication()
	app.SetRoot(flex, true)

	return &App{
		Application: app,
		servers:     servers,
		flex:        flex,
		serversView: serversView,
		outputView:  outputView,
	}
}

func (a *App) setAction() {
	a.serversView.SetSelectedFunc(func(row, column int) {
		if row > len(a.servers) {
			return
		}
		if a.cmd != nil {
			a.cmd.Process.Kill()
			a.cmd = nil
			a.outputView.Clear()
		}

		server := a.servers[row-1]

		cmd, err := ConnectCmd(server.OpenVPNConfigDataBase64)
		if err != nil {
			log.Fatal(err)
		}
		a.cmd = cmd
		a.cmd.Stdout = a.outputView
		a.cmd.Stderr = a.outputView

		go func() {
			a.cmd.Run()
		}()
	})

	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		}
		switch event.Rune() {
		case 'q':
			a.Stop()
			return nil
		}
		return event
	})

	a.outputView.SetChangedFunc(func() {
		a.Draw()
	})
}

func (a *App) Run() {
	defer os.Remove("/tmp/openvpnconf")

	a.setAction()

	if err := a.Application.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *App) Stop() {
	if a.cmd != nil {
		a.cmd.Process.Kill()
		a.cmd = nil
	}
	a.Application.Stop()
}

type ServersView struct {
	*tview.Table
	Servers []Server
}

func NewServersView(servers []Server) *ServersView {
	table := tview.NewTable()

	table.SetCell(0, 0, tview.NewTableCell("Country"))
	table.SetCell(0, 1, tview.NewTableCell("IP"))
	table.SetCell(0, 2, tview.NewTableCell("Ping"))
	table.SetCell(0, 3, tview.NewTableCell("Speed"))
	table.SetCell(0, 4, tview.NewTableCell("Score"))
	table.SetCell(0, 5, tview.NewTableCell("TotalUsers"))
	table.SetCell(0, 6, tview.NewTableCell("TotalTraffic"))

	for i, server := range servers {
		table.SetCell(i+1, 0, tview.NewTableCell(server.CountryShort))
		table.SetCell(i+1, 1, tview.NewTableCell(server.IP))
		table.SetCell(i+1, 2, tview.NewTableCell(server.Ping))
		table.SetCell(i+1, 3, tview.NewTableCell(server.Speed))
		table.SetCell(i+1, 4, tview.NewTableCell(server.Score))
		table.SetCell(i+1, 5, tview.NewTableCell(server.TotalUsers))
		table.SetCell(i+1, 6, tview.NewTableCell(server.TotalTraffic))
	}

	table.Select(1, 0).SetFixed(1, 1).SetSelectable(true, false)
	table.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			table.Select(1, 0)
		}
	})
	table.SetTitle("servers").SetTitleAlign(tview.AlignLeft).SetBorder(true)

	return &ServersView{
		Table:   table,
		Servers: servers,
	}
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

	app := NewApp(servers)
	app.Run()
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

func ConnectCmd(openVPNConfigDataBase64 string) (*exec.Cmd, error) {
	conf, err := base64.StdEncoding.DecodeString(openVPNConfigDataBase64)
	if err != nil {
		return nil, err
	}

	f, err := os.Create("/tmp/openvpnconf")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.Write(conf)
	if err != nil {
		return nil, err
	}

	// cmd := exec.Command("sudo", "openvpn", "/tmp/openvpnconf")
	cmd := exec.Command("openvpn", "/tmp/openvpnconf")

	return cmd, nil
}
