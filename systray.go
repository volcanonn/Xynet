package main

import (
	"context"
	_ "embed"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/appicon.png
var trayIcon []byte

func (a *App) setupSystray(ctx context.Context) {
	onReady := func() {
		systray.SetIcon(trayIcon)
		systray.SetTitle("Xynet")
		systray.SetTooltip("Xynet - Visual VPN Manager")

		mShow := systray.AddMenuItem("Open Xynet", "Show the main window")
		systray.AddSeparator()
		mDisconnectAll := systray.AddMenuItem("Disconnect All Tunnels", "Stop all proxies")
		mKillSwitch := systray.AddMenuItem("Kill All Strict Apps", "Kill all vopono namespaces")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application completely")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					runtime.WindowShow(ctx)
				case <-mDisconnectAll.ClickedCh:
					a.Undeploy()
				case <-mKillSwitch.ClickedCh:
					procs := a.ListVoponoProcesses()
					for _, p := range procs {
						a.KillVopono(p.ID)
					}
				case <-mQuit.ClickedCh:
					systray.Quit()
					runtime.Quit(ctx)
				case <-ctx.Done():
					systray.Quit()
					return
				}
			}
		}()
	}

	onExit := func() {}

	go systray.Run(onReady, onExit)
}
