package main

import (
	"fmt"
	"image"
	"image/color"

	//"log"
	"os"

	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	//"fyne.io/fyne/v2/container"

	//"fyne.io/fyne/v2/layout"

	r2 "github.com/golang/geo/r2"
	//heatmap "github.com/markus-wa/go-heatmap/v2"
	//schemes "github.com/markus-wa/go-heatmap/v2/schemes"

	ex "github.com/markus-wa/demoinfocs-golang/v3/examples"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v3/pkg/demoinfocs/msg"
)

var (
	timespeed     time.Duration = time.Millisecond * 16
	demoPlayState bool          = true
	hasNext       bool
)

// Run like this: go run heatmap.go -demo /path/to/demo.dem > out.jpg
func main() {
	//
	// Parsing
	//

	demoPath := "C:/demostest/esea_match_17012266.dem"

	getROI(demoPath)

	//f, err := os.Open(ex.DemoPathFromArgs())
	f, err := os.Open(demoPath)
	checkError(err)
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	// Parse header (contains map-name etc.)
	header, err := p.ParseHeader()
	checkError(err)

	var (
		mapMetadata ex.Map
		mapRadarImg image.Image
	)

	p.RegisterNetMessageHandler(func(msg *msg.CSVCMsg_ServerInfo) {
		// Get metadata for the map that the game was played on for coordinate translations
		mapMetadata = ex.GetMapMetadata(header.MapName, msg.GetMapCrc())

		//Load map overview image
		mapRadarImg = ex.GetMapRadar(header.MapName, msg.GetMapCrc())
	})

	a := app.New()
	w := a.NewWindow("Image")

	hasNext, err = p.ParseNextFrame()
	checkError(err)

	content := container.NewWithoutLayout()

	radar := canvas.NewImageFromImage(mapRadarImg)
	radar.Resize(fyne.NewSize(1024, 1024))
	radar.FillMode = canvas.ImageFillOriginal

	roundLabel := widget.NewLabel("Round 0")

	startButton := widget.NewButton("Start Demo", func() {

		p.RegisterEventHandler(func(e events.PlayerConnect) {
			c := createPlayerTracker(e.Player, p, mapMetadata)

			content.Add(c)
		})

		go func() {
			for hasNext {
				if demoPlayState {
					for range time.Tick(timespeed) {
						hasNext, err = p.ParseNextFrame()
						checkError(err)

						roundLabel.SetText(fmt.Sprintf("Round: %d, Tick: %d", p.GameState().TotalRoundsPlayed()+1, p.GameState().IngameTick()))

						break
					}
				}
			}
		}()
	})

	speedButtonRealTime := widget.NewButton("Real-Time", func() {
		timespeed = time.Millisecond * 16
	})

	speedButton16 := widget.NewButton("16x", func() {
		timespeed = time.Millisecond
	})

	pauseButton := widget.NewButton("Pause/Resume", func() {
		if demoPlayState {
			demoPlayState = false
		} else {
			demoPlayState = true
		}
	})

	content.Add(radar)
	content.Add(container.NewVBox(startButton, speedButton16, speedButtonRealTime, pauseButton))
	content.Add(roundLabel)
	roundLabel.Move(fyne.NewPos(512, 0))

	w.SetContent(content)

	w.ShowAndRun()
}

func createPlayerTracker(player *common.Player, parser demoinfocs.Parser, mapMetadata ex.Map) (c *fyne.Container) {

	var obj *canvas.Circle

	if player.Team == 3 {
		obj = canvas.NewCircle(color.RGBA{173, 216, 230, 255})
	} else if player.Team == 2 {
		obj = canvas.NewCircle(color.RGBA{250, 93, 83, 255})
	} else {
		obj = canvas.NewCircle(color.Black)
	}

	obj.Resize(fyne.NewSize(10, 10))

	name := widget.NewLabel(player.Name)

	tracker := container.NewWithoutLayout(obj, name)

	parser.RegisterEventHandler(func(e events.FrameDone) {
		// Translate positions from in-game coordinates to radar overview image pixels
		x, y := mapMetadata.TranslateScale(player.Position().X, player.Position().Y)

		point := r2.Point{X: x, Y: y}

		movePlayer(tracker, fyne.NewPos(float32(point.X)-(obj.Size().Height/2), float32(point.Y)-(obj.Size().Width/2)))
	})

	return tracker
}

func movePlayer(player fyne.CanvasObject, pos fyne.Position) {
	canvas.NewPositionAnimation(player.Position(), pos, timespeed, func(p fyne.Position) {
		player.Move(p)
		player.Refresh()
	}).Start()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func getROI(demoPath string) {

	f, err := os.Open(demoPath)
	checkError(err)
	defer f.Close()

	p := demoinfocs.NewParser(f)
	defer p.Close()

	var tradeAvailable bool
	var deathTime int
	var deathTimeTwo int
	var playerDead string
	p.RegisterEventHandler(func(e events.Kill) {

		deathTimeTwo = p.GameState().IngameTick()

		if e.Victim.TeamState.ClanName() == "8bit gamers" {
			tradeAvailable = true
			deathTime = p.GameState().IngameTick()
			playerDead = e.Victim.Name

		} else if tradeAvailable && (deathTimeTwo-deathTime) > 128 {
			print("Death Untraded At: ", deathTime, " Victim: ", playerDead, "\n")
			tradeAvailable = false

		} else if tradeAvailable && (deathTimeTwo-deathTime) < 128 {
			tradeAvailable = false
		}
	})

	p.ParseToEnd()

	/* Rounds of Interest:
	Clutches thrown (losing a 1v5 should never happen)
	Retake situations, for visualizing setups (teams are even or we have advantage)
	Kills gone untraded
	Teamflashes
	Spacing?




	*/

	f.Close()
	p.Close()

}
