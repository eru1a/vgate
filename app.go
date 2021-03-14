package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	*tview.Application
	servers     []Server
	flex        *tview.Flex
	serversView *ServersView
	outputView  *tview.TextView
	statusView  *tview.TextView
	connectCmd  *exec.Cmd
}

func NewApp(servers []Server) *App {
	serversView := NewServersView(servers)

	outputView := tview.NewTextView()
	outputView.SetTitle("output").SetTitleAlign(tview.AlignLeft).SetBorder(true)

	statusView := tview.NewTextView()

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)

	flex.AddItem(serversView, 0, 1, true).
		AddItem(outputView, 0, 1, false).
		AddItem(statusView, 1, 1, false)

	app := tview.NewApplication()
	app.SetRoot(flex, true)

	return &App{
		Application: app,
		servers:     servers,
		flex:        flex,
		serversView: serversView,
		outputView:  outputView,
		statusView:  statusView,
	}
}

func (a *App) setAction() {
	a.serversView.SetSelectedFunc(func(row, column int) {
		if os.Getegid() != 0 {
			a.outputView.SetText("root required")
			return
		}

		if row > len(a.servers) {
			return
		}

		a.Connect(a.servers[row-1])

	})

	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		}
		switch event.Rune() {
		case 'x':
			a.Disconnect()
			return nil
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

func (a *App) Connect(server Server) {
	a.Disconnect()

	cmd, err := ConnectCmd(server.OpenVPNConfigDataBase64)
	if err != nil {
		log.Fatal(err)
	}
	a.connectCmd = cmd
	a.connectCmd.Stdout = a.outputView
	a.connectCmd.Stderr = a.outputView

	// TODO: `$curl inet-ip.info` 等で実際に確認する
	a.statusView.SetText(fmt.Sprintf("connect to %s [x: disconnect]", server.IP))

	go func() {
		a.connectCmd.Run()
	}()
}

func (a *App) Disconnect() {
	if a.connectCmd != nil {
		a.connectCmd.Process.Kill()
		a.connectCmd = nil
		a.outputView.Clear()
		a.statusView.Clear()
	}
}

func (a *App) Stop() {
	a.Disconnect()
	a.Application.Stop()
}

func (a *App) Run() {
	defer os.Remove("/tmp/openvpnconf")

	a.setAction()

	if err := a.Application.Run(); err != nil {
		log.Fatal(err)
	}
}

type Order int

const (
	CountryD Order = iota
	CountryA
	PingD
	PingA
	SpeedD
	SpeedA
	ScoreD
	ScoreA
	UptimeD
	UptimeA
	TotalUsersD
	TotalUsersA
	TotalTrafficD
	TotalTrafficA
)

type ServersView struct {
	*tview.Table
	servers []Server
	order   Order
}

func NewServersView(servers []Server) *ServersView {
	table := tview.NewTable()
	table.Select(1, 0).SetFixed(1, 1).SetSelectable(true, false)
	table.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			table.Select(1, 0)
		}
	})
	table.SetTitle("servers").SetTitleAlign(tview.AlignLeft).SetBorder(true)

	serversView := &ServersView{
		Table:   table,
		servers: servers,
		order:   ScoreD,
	}

	serversView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'c':
			if serversView.order == CountryD {
				serversView.order = CountryA
			} else {
				serversView.order = CountryD
			}
		case 'p':
			if serversView.order == PingD {
				serversView.order = PingA
			} else {
				serversView.order = PingD
			}
		case 's':
			if serversView.order == SpeedD {
				serversView.order = SpeedA
			} else {
				serversView.order = SpeedD
			}
		case 'e':
			if serversView.order == ScoreD {
				serversView.order = ScoreA
			} else {
				serversView.order = ScoreD
			}
		case 'u':
			if serversView.order == UptimeD {
				serversView.order = UptimeA
			} else {
				serversView.order = UptimeD
			}
		case 't':
			if serversView.order == TotalUsersD {
				serversView.order = TotalUsersA
			} else {
				serversView.order = TotalUsersD
			}
		case 'f':
			if serversView.order == TotalTrafficD {
				serversView.order = TotalTrafficA
			} else {
				serversView.order = TotalTrafficD
			}
		default:
			return event
		}
		serversView.SetCells()
		return nil
	})
	serversView.SetCells()
	return serversView
}

func (s *ServersView) SetCells() {
	s.Sort()

	s.Table.Clear()

	s.Table.SetCell(0, 0, tview.NewTableCell("Country(c)"))
	s.Table.SetCell(0, 1, tview.NewTableCell("IP"))
	s.Table.SetCell(0, 2, tview.NewTableCell("Ping(p)"))
	s.Table.SetCell(0, 3, tview.NewTableCell("Speed(s)"))
	s.Table.SetCell(0, 4, tview.NewTableCell("Score(e)"))
	s.Table.SetCell(0, 5, tview.NewTableCell("Uptime(u)"))
	s.Table.SetCell(0, 6, tview.NewTableCell("TotalUsers(t)"))
	s.Table.SetCell(0, 7, tview.NewTableCell("TotalTraffic(f)"))

	switch s.order {
	case CountryD:
		s.Table.SetCell(0, 0, tview.NewTableCell("Country(c)▼"))
	case CountryA:
		s.Table.SetCell(0, 0, tview.NewTableCell("Country(c)▲"))
	case PingD:
		s.Table.SetCell(0, 2, tview.NewTableCell("Ping(p)▼"))
	case PingA:
		s.Table.SetCell(0, 2, tview.NewTableCell("Ping(p)▲"))
	case SpeedD:
		s.Table.SetCell(0, 3, tview.NewTableCell("Speed(s)▼"))
	case SpeedA:
		s.Table.SetCell(0, 3, tview.NewTableCell("Speed(s)▲"))
	case ScoreD:
		s.Table.SetCell(0, 4, tview.NewTableCell("Score(e)▼"))
	case ScoreA:
		s.Table.SetCell(0, 4, tview.NewTableCell("Score(e)▲"))
	case UptimeD:
		s.Table.SetCell(0, 5, tview.NewTableCell("Uptime(u)▼"))
	case UptimeA:
		s.Table.SetCell(0, 5, tview.NewTableCell("Uptime(u)▲"))
	case TotalUsersD:
		s.Table.SetCell(0, 6, tview.NewTableCell("TotalUsers(t)▼"))
	case TotalUsersA:
		s.Table.SetCell(0, 6, tview.NewTableCell("TotalUsers(t)▲"))
	case TotalTrafficD:
		s.Table.SetCell(0, 7, tview.NewTableCell("TotalTraffic(f)▼"))
	case TotalTrafficA:
		s.Table.SetCell(0, 7, tview.NewTableCell("TotalTraffic(f)▲"))
	}

	for i, server := range s.servers {
		s.Table.SetCell(i+1, 0, tview.NewTableCell(strings.ToUpper(server.CountryShort)).SetAlign(tview.AlignCenter))
		s.Table.SetCell(i+1, 1, tview.NewTableCell(server.IP).SetAlign(tview.AlignRight))
		s.Table.SetCell(i+1, 2, tview.NewTableCell(server.Ping).SetAlign(tview.AlignRight))
		s.Table.SetCell(i+1, 3, tview.NewTableCell(strconv.Itoa(server.Speed)).SetAlign(tview.AlignRight))
		s.Table.SetCell(i+1, 4, tview.NewTableCell(strconv.Itoa(server.Score)).SetAlign(tview.AlignRight))
		s.Table.SetCell(i+1, 5, tview.NewTableCell(uptimeToString(server.Uptime)).SetAlign(tview.AlignRight))
		s.Table.SetCell(i+1, 6, tview.NewTableCell(strconv.Itoa(server.TotalUsers)).SetAlign(tview.AlignRight))
		s.Table.SetCell(i+1, 7, tview.NewTableCell(trafficToString(server.TotalTraffic)).SetAlign(tview.AlignRight))
	}
}

func (s *ServersView) Sort() {
	comp := func(i, j int) bool {
		switch s.order {
		case CountryD:
			return s.servers[i].CountryShort > s.servers[j].CountryShort
		case CountryA:
			return s.servers[i].CountryShort < s.servers[j].CountryShort
		case PingD:
			ping1, _ := strconv.Atoi(s.servers[i].Ping)
			ping2, _ := strconv.Atoi(s.servers[j].Ping)
			return ping1 > ping2
		case PingA:
			ping1, _ := strconv.Atoi(s.servers[i].Ping)
			ping2, _ := strconv.Atoi(s.servers[j].Ping)
			return ping1 < ping2
		case SpeedD:
			return s.servers[i].Speed > s.servers[j].Speed
		case SpeedA:
			return s.servers[i].Speed < s.servers[j].Speed
		case ScoreD:
			return s.servers[i].Score > s.servers[j].Score
		case ScoreA:
			return s.servers[i].Score < s.servers[j].Score
		case UptimeD:
			return s.servers[i].Uptime > s.servers[j].Uptime
		case UptimeA:
			return s.servers[i].Uptime < s.servers[j].Uptime
		case TotalUsersD:
			return s.servers[i].TotalUsers > s.servers[j].TotalUsers
		case TotalUsersA:
			return s.servers[i].TotalUsers < s.servers[j].TotalUsers
		case TotalTrafficD:
			return s.servers[i].TotalTraffic > s.servers[j].TotalTraffic
		case TotalTrafficA:
			return s.servers[i].TotalTraffic < s.servers[j].TotalTraffic
		default:
			panic("unrecheable!")
		}
	}

	sort.SliceStable(s.servers, comp)
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

	cmd := exec.Command("openvpn", "/tmp/openvpnconf")

	return cmd, nil
}

func uptimeToString(uptime int) string {
	days := uptime / (60 * 60 * 24 * 1000)
	return fmt.Sprintf("%d days", days)
}

func trafficToString(traffic int) string {
	gb := traffic / (1000 * 1000 * 1000 * 1000)
	return fmt.Sprintf("%d gb", gb)
}
