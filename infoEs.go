package main

import (
	_ "embed"
	"github.com/PuerkitoBio/goquery"
	"github.com/sqweek/dialog"
	"github.com/jg1vpp/winc"
	"strconv"
	"strings"
	"time"
)

var (
	JST       [8]string
	okinawa   [8]string
	yamakawa  [8]string
	kokubunji [8]string
	wakkanai  [8]string
	abort     chan struct{}
)

const winsize = "infoEswindow"

type EsView struct {
	list *winc.ListView
}

var esview EsView

type EsItem struct {
	Place string
	Power string
	State string
	Last  string
}

func (item EsItem) Text() (text []string) {
	text = append(text, item.Place)
	text = append(text, item.Power)
	text = append(text, item.State)
	text = append(text, item.Last)
	return
}

func (item EsItem) ImageIndex() int {
	return 0
}

func Escheck(area [8]string) (string, string, string) {
	Es_power := "不明"
	Es_state := "不明"
	Late_jst := JST[0]
	for index := 7; index >= 0; index = index - 1 {
		Es_power_num, err := strconv.ParseFloat(strings.TrimSpace(area[index]), 64)
		if err == nil {
			Late_jst = JST[index]
			Es_power = area[index]
			Es_state = "静穏"
			if Es_power_num > 7 {
				Es_state = "Es"
			}
			if Es_power_num > 8 {
				Es_state = "強いEs"
			}
			if Es_power_num > 9 {
				Es_state = "非常に強いEs"
			}
			break
		}
	}
	return Es_power, Es_state, Late_jst
}

func EsUpdate() {
	//listを消す
	esview.list.DeleteAllItems()

	//情報を取得
	get_url_info, err := goquery.NewDocument("https://wdc.nict.go.jp/IONO/fxEs/latest-fxEs.html")
	if err != nil {
		DisplayToast(err.Error())
	}

	//必要なところだけ切り出し
	result := get_url_info.Find("td")
	result.Each(func(index int, s *goquery.Selection) {
		if index > 9 && index%5 == 0 {
			JST[index/5-2] = s.Text()
		}
		if index > 9 && index%5 == 1 {
			okinawa[index/5-2] = s.Text()
		}
		if index > 9 && index%5 == 2 {
			yamakawa[index/5-2] = s.Text()
		}
		if index > 9 && index%5 == 3 {
			kokubunji[index/5-2] = s.Text()
		}
		if index > 9 && index%5 == 4 {
			wakkanai[index/5-2] = s.Text()
		}
	})

	//表示
	power, state, jst := Escheck(okinawa)
	esview.list.AddItem(EsItem{
		Place: "恩納/沖縄",
		Power: power,
		State: state,
		Last:  jst,
	})

	power, state, jst = Escheck(yamakawa)
	esview.list.AddItem(EsItem{
		Place: "山川/鹿児島",
		Power: power,
		State: state,
		Last:  jst,
	})

	power, state, jst = Escheck(kokubunji)
	esview.list.AddItem(EsItem{
		Place: "国分寺/東京",
		Power: power,
		State: state,
		Last:  jst,
	})

	power, state, jst = Escheck(wakkanai)
	esview.list.AddItem(EsItem{
		Place: "稚内/北海道",
		Power: power,
		State: state,
		Last:  jst,
	})
}

var mainWindow *winc.Form

func wndOnClose(arg *winc.Event) {
	x, y := mainWindow.Pos()
	w, h := mainWindow.Size()
	SetINI(winsize, "x", strconv.Itoa(x))
	SetINI(winsize, "y", strconv.Itoa(y))
	SetINI(winsize, "w", strconv.Itoa(w))
	SetINI(winsize, "h", strconv.Itoa(h))
	abort <- struct{}{}
	mainWindow.Close()
}

func makewindow() {
	// --- Make Window
	mainWindow = winc.NewForm(nil)

	x, _ := strconv.Atoi(GetINI(winsize, "x"))
	y, _ := strconv.Atoi(GetINI(winsize, "y"))
	w, _ := strconv.Atoi(GetINI(winsize, "w"))
	h, _ := strconv.Atoi(GetINI(winsize, "h"))
	if w <= 0 || h <= 0 {
		w = 520
		h = 140
	}

	mainWindow.SetSize(w, h)
	if x <= 0 || y <= 0 {
		mainWindow.Center()
	} else {
		mainWindow.SetPos(x, y)
	}
	mainWindow.SetText("Eスポ情報")

	esview.list = winc.NewListView(mainWindow)
	esview.list.EnableEditLabels(false)
	esview.list.AddColumn("観測地点", 120)
	esview.list.AddColumn("計測値", 120)
	esview.list.AddColumn("状態", 120)
	esview.list.AddColumn("最終観測時刻", 140)
	dock := winc.NewSimpleDock(mainWindow)
	dock.Dock(esview.list, winc.Fill)

	mainWindow.Show()

	mainWindow.OnClose().Bind(wndOnClose)
}

func UpdateLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	for {
		select {
		case <-ticker.C:
			EsUpdate()
		case <-abort:
			return
		}
	}
}

func init() {
	OnLaunchEvent = onLaunchEvent
	winc.DllName = "infoEs"
}

func onLaunchEvent() {
	RunDelphi(`PluginMenu.Add(op.Put(MainMenu.CreateMenuItem(), "Name", "PluginEsInfoWindow"))`)
	RunDelphi(`op.Put(MainMenu.FindComponent("PluginEsInfoWindow"), "Caption", "Es情報 ウィンドウ")`)

	RunDelphi(`PluginMenu.Add(op.Put(MainMenu.CreateMenuItem(), "Name", "PluginEsInfoHow"))`)
	RunDelphi(`op.Put(MainMenu.FindComponent("PluginEsInfoHow"), "Caption", "Es情報 利用方法")`)

	HandleButton("MainForm.MainMenu.PluginEsInfoWindow", func(num int){
		abort = make(chan struct{})
		makewindow()
		EsUpdate()
		go UpdateLoop()
	})

	HandleButton("MainForm.MainMenu.PluginEsInfoHow", func(num int){
		dialog.Message("%s", "このシステムはNICTのサイトから情報を取得しています。\n15分毎に自動更新しています。").Title("利用方法").Info()
	})	
}