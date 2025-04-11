package services

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"net/url"
	"os"
	"os/signal"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/gorilla/websocket"
)

const (
	defaultWidth       = 700 // Увеличенная ширина окна по умолчанию
	defaultHeight      = 160 // Увеличенная высота окна по умолчанию
	defaultPointsCount = 50
)

type TrafficData struct {
	Up   int `json:"up"`
	Down int `json:"down"`
}

// WebSocketGraph структура для управления графиком
type WebSocketGraph struct {
	app         fyne.App
	window      fyne.Window
	plot        *fyne.Container
	dataUp      []float64
	dataDown    []float64
	pointsCount int
	width       int
	height      int
	wsURL       url.URL
}

func NewWebSocketGraph(wsURL url.URL, width, height, pointsCount int) *WebSocketGraph {
	return &WebSocketGraph{
		dataUp:      make([]float64, pointsCount),
		dataDown:    make([]float64, pointsCount),
		pointsCount: pointsCount,
		width:       width,
		height:      height,
		wsURL:       wsURL,
	}
}

// NewDefaultWebSocketGraph создает новый экземпляр WebSocketGraph с размерами по умолчанию
func NewClashAPIService(wsURL url.URL) *WebSocketGraph {
	return NewWebSocketGraph(wsURL, defaultWidth, defaultHeight, defaultPointsCount)
}

func (wg *WebSocketGraph) StartMonitoring(parent *fyne.Container) {
	wg.plot = container.NewWithoutLayout()
	parent.Add(wg.plot)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				wg.connectAndStream(done)
				log.Println("Disconnected, retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func (wg *WebSocketGraph) connectAndStream(done chan struct{}) {
	log.Printf("connecting to %s", wg.wsURL.String())
	c, _, err := websocket.DefaultDialer.Dial(wg.wsURL.String(), nil)
	if err != nil {
		log.Println("dial:", err)
		return
	}
	defer c.Close()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		default:
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			var data TrafficData
			err = json.Unmarshal(message, &data)
			if err != nil {
				log.Println("unmarshal:", err)
				continue
			}

			copy(wg.dataUp, wg.dataUp[1:])
			copy(wg.dataDown, wg.dataDown[1:])
			wg.dataUp[len(wg.dataUp)-1] = float64(data.Up)
			wg.dataDown[len(wg.dataDown)-1] = float64(data.Down)

			maxVal := wg.getOverallMaxValue()
			wg.plot.Objects = nil
			step := float64(wg.width) / float64(len(wg.dataUp)-1)

			wg.drawBrokenCurve(wg.plot, wg.dataUp, maxVal, step, color.RGBA{R: 0, G: 255, B: 0, A: 255})
			wg.drawBrokenCurve(wg.plot, wg.dataDown, maxVal, step, color.RGBA{R: 0, G: 0, B: 255, A: 255})
			wg.addScale(maxVal)

			wg.plot.Refresh()
		}
	}
}

func (wg *WebSocketGraph) drawBrokenCurve(plot *fyne.Container, data []float64, maxValue float64, step float64, lineColor color.RGBA) {
	allZero := true
	for _, val := range data {
		if val != 0 {
			allZero = false
			break
		}
	}

	if len(data) < 2 {
		return // Нужны как минимум две точки
	}

	// Собираем основные точки
	points := make([]fyne.Position, len(data))
	for i := 0; i < len(data); i++ {
		var y float64
		if !allZero && maxValue > 0 {
			y = float64(wg.height) - (data[i]/maxValue)*float64(wg.height)
		} else {
			y = float64(wg.height)
		}
		x := step * float64(i)
		points[i] = fyne.Position{X: float32(x), Y: float32(y)}
	}

	// Создаём сглаженные точки с помощью сплайна Catmull-Rom
	smoothPoints := wg.catmullRomSpline(points, 20, 1)

	// Определяем цвет заливки (полупрозрачный оттенок основного цвета линии)
	fillColor := color.RGBA{
		R: lineColor.R,
		G: lineColor.G,
		B: lineColor.B,
		A: 70, // Полупрозрачность (0-255)
	}

	// Закрашиваем область под кривой
	for i := 0; i < len(smoothPoints); i++ {
		point := smoothPoints[i]
		fillLine := canvas.NewLine(fillColor)
		fillLine.Position1 = point
		fillLine.Position2 = fyne.Position{X: point.X, Y: float32(wg.height)} // Нижняя граница
		fillLine.StrokeWidth = 1
		plot.Add(fillLine)
	}

	// Рисуем саму линию
	for i := 0; i < len(smoothPoints)-1; i++ {
		line := canvas.NewLine(lineColor)
		line.Position1 = smoothPoints[i]
		line.Position2 = smoothPoints[i+1]
		line.StrokeWidth = 1
		plot.Add(line)
	}
}

// catmullRomSpline функция для вычисления сплайна Catmull-Rom с параметром tension
func (wg *WebSocketGraph) catmullRomSpline(points []fyne.Position, steps int, tension float32) []fyne.Position {
	if len(points) < 2 {
		return points
	}

	var smoothPoints []fyne.Position
	smoothPoints = append(smoothPoints, points[0])

	for i := 0; i < len(points)-1; i++ {
		p0 := points[max(0, i-1)]
		p1 := points[i]
		p2 := points[i+1]
		p3 := points[min(len(points)-1, i+2)]

		for j := 1; j <= steps; j++ {
			t := float32(j) / float32(steps)
			t2 := t * t
			t3 := t2 * t

			// Коэффициенты Catmull-Rom с учётом tension
			c0 := (-tension * t3) + (2 * tension * t2) - (tension * t)
			c1 := ((2 - tension) * t3) + ((tension - 3) * t2) + 1
			c2 := ((tension - 2) * t3) + ((3 - 2*tension) * t2) + (tension * t)
			c3 := (tension * t3) - (tension * t2)

			x := c0*p0.X + c1*p1.X + c2*p2.X + c3*p3.X
			y := c0*p0.Y + c1*p1.Y + c2*p2.Y + c3*p3.Y

			// Ограничиваем Y сверху и снизу
			if y < 0 {
				y = 0
			}
			if y > float32(wg.height) {
				y = float32(wg.height)
			}
			if x < 0 {
				x = 0
			}
			if x > float32(wg.width) {
				x = float32(wg.width)
			}

			smoothPoints = append(smoothPoints, fyne.Position{X: x, Y: y})
		}
	}

	smoothPoints = append(smoothPoints, points[len(points)-1])
	return smoothPoints
}

// getOverallMaxValue функция для нахождения общего максимального значения для "up" и "down"
func (wg *WebSocketGraph) getOverallMaxValue() float64 {
	maxVal := 0.0
	for _, v := range wg.dataUp {
		if v > maxVal {
			maxVal = v
		}
	}
	for _, v := range wg.dataDown {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// addScale функция для добавления шкалы слева с "ровными" значениями и пунктирными линиями
func (wg *WebSocketGraph) addScale(maxValue float64) {
	if maxValue == 0 {
		return
	}

	maxBps := maxValue * 8

	// Определяем разумное максимальное значение шкалы и шаг
	var roundedMaxBps float64
	var scaleStepBps float64
	var unitDivisor float64
	var unitSuffix string

	const numScaleSteps = 5

	switch {
	case maxBps >= 10*1e9: // >= 10 Gbps
		unitDivisor = 1e9
		unitSuffix = " Gbps"
		roundedMaxBps = math.Ceil(maxBps/unitDivisor/10) * 10 * unitDivisor
		scaleStepBps = roundedMaxBps / numScaleSteps
	case maxBps >= 10*1e6: // >= 10 Mbps
		unitDivisor = 1e6
		unitSuffix = " Mbps"
		roundedMaxBps = math.Ceil(maxBps/unitDivisor/10) * 10 * unitDivisor
		scaleStepBps = roundedMaxBps / numScaleSteps
	case maxBps >= 10*1e3: // >= 10 Kbps
		unitDivisor = 1e3
		unitSuffix = " Kbps"
		roundedMaxBps = math.Ceil(maxBps/unitDivisor/10) * 10 * unitDivisor
		scaleStepBps = roundedMaxBps / numScaleSteps
	default:
		unitDivisor = 1
		unitSuffix = " bps"
		roundedMaxBps = math.Ceil(maxBps/10) * 10
		scaleStepBps = roundedMaxBps / numScaleSteps
		if roundedMaxBps == 0 {
			roundedMaxBps = 5
			scaleStepBps = 1
		}
	}

	// Рисуем шкалу
	for i := 0; i <= numScaleSteps; i++ {
		scaleValueBps := float64(i) * scaleStepBps
		scaleValueData := scaleValueBps / 8

		var y float64
		if maxValue > 0 {
			y = float64(wg.height) - (scaleValueData/maxValue)*float64(wg.height)
		} else {
			y = float64(wg.height)
		}

		scaleText := fmt.Sprintf("%.0f%s", math.Round(scaleValueBps/unitDivisor), unitSuffix)
		text := canvas.NewText(scaleText, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		text.Move(fyne.NewPos(10, float32(y)-text.MinSize().Height/2))
		wg.plot.Add(text)

		// Начало штрихов после текста с запасом
		dashStartX := float32(10) + text.MinSize().Width + 5 // 10 — начальная позиция текста, +5 — запас
		dashLength := float32(5)
		gapLength := float32(5)
		x := dashStartX

		for x < float32(wg.width) {
			startX := x
			endX := x + dashLength
			if endX > float32(wg.width) {
				endX = float32(wg.width)
			}

			dash := canvas.NewLine(color.RGBA{R: 0, G: 0, B: 0, A: 100})
			dash.Position1 = fyne.Position{X: startX, Y: float32(y)}
			dash.Position2 = fyne.Position{X: endX, Y: float32(y)}
			dash.StrokeWidth = 1
			wg.plot.Add(dash)

			x += dashLength + gapLength
		}
	}
}
