package app

import (
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/encoding"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type App struct {
	ScreenX      int
	ScreenY      int
	screen       tcell.Screen
	spinnerIndex int32
	keyEvent     chan *tcell.EventKey
	drawables    []*Drawable
	systems      []*System
	// clear/add/remove is less accurate execution than clear.
	// so it makes sense to store keep-alive stuff like this, instead of having
	// separate arrays to iterate through
	keepAlive    map[interface{}]bool
	changeMutex  sync.Mutex
	routineMutex sync.Mutex
	quit         bool
	// re-render screen if true
	dirty           chan bool
	loading         bool
	runOnAppRoutine []func()
	spinner         *SpinnerTCell
	view            View
}

const (
	AppFPS             = 30
	AppFPSMilliseconds = time.Second / AppFPS
)

var (
	appInstance *App
	once        sync.Once
	//appBackground = tcell.ColorDefault // TODO - should be black?
	//appBackground = tcell.NewRGBColor(28, 36, 42)
	appBackground = tcell.NewRGBColor(22, 22, 22)
)

func CreateNewApp() *App {
	once.Do(initApp)
	return appInstance
}

func GetApp() *App {
	return appInstance
}

func initApp() {
	if os.Getenv("TERM") == "cygwin" {
		os.Setenv("TERM", "")
	}
	encoding.Register()
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatalln(err)
	}
	if err := screen.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	defStyle := tcell.StyleDefault.Background(appBackground).Foreground(tcell.ColorDefault)
	screen.SetStyle(defStyle)
	screen.EnableMouse()
	s := NewSimpleSpinner()
	x, y := screen.Size()
	appInstance = &App{
		screen:          screen,
		ScreenX:         x,
		ScreenY:         y,
		spinnerIndex:    0,
		keyEvent:        make(chan *tcell.EventKey),
		runOnAppRoutine: make([]func(), 0, 64),
		drawables:       make([]*Drawable, 0, 256),
		systems:         make([]*System, 0, 128),
		keepAlive:       make(map[interface{}]bool),
		dirty:           make(chan bool),
		spinner:         s,
	}
}

func (a *App) Start() {
	defer a.Close()
	go a.processTerminalEvents()
	go a.processOsSignals()
	for {
		if a.quit == true {
			return
		}
		// TODO - could be added as an potential performance improvement
		// it will reduce the render cycles
		//select {
		//case <-a.dirty:
		//}
		a.screen.Show()
		for _, system := range a.systems {
			(*system).Update()
		}
		a.screen.Clear()
		if a.loading {
			a.spinner.Draw(a.screen)
		}
		for _, drawable := range a.drawables {
			(*drawable).Draw(a.screen)
		}
		if len(a.runOnAppRoutine) == 0 {
			time.Sleep(AppFPSMilliseconds)
			continue
		}
		funcsToRun := len(a.runOnAppRoutine) - 1
		for i := funcsToRun; i >= 0; i-- {
			a.runOnAppRoutine[i]()
		}
		if len(a.runOnAppRoutine) > funcsToRun {
			a.runOnAppRoutine = a.runOnAppRoutine[funcsToRun+1:]
			continue
		}
		a.runOnAppRoutine = nil
	}
}

func (a *App) Close() {
	a.screen.Clear()
	a.screen.Show()
	a.screen.Fini()
	close(a.keyEvent)
}

func (a *App) Loading(flag bool) {
	a.spinner.text = "Fetching"
	a.loading = flag
}

func (a *App) LoadingWithText(flag bool, text string) {
	a.spinner.text = text
	a.loading = flag
}

func (a *App) SetView(view View) {
	if a.view != nil {
		a.view.Destroy()
		delete(a.keepAlive, a.view)
		a.RemoveDrawable(a.view.(Drawable))
		a.RemoveSystem(a.view.(System))
	}
	a.keepAlive[view] = true
	a.view = view
	a.ClearNow()
	a.AddDrawable(view.(Drawable))
	a.AddSystem(view.(System))
	view.Init()
}

func (a *App) KeepAlive(component interface{}) {
	a.keepAlive[component] = true
}

func (a *App) UnKeepAlive(component interface{}) {
	a.keepAlive[component] = false
}

func (a *App) AddDrawable(drawable Drawable) {
	a.changeMutex.Lock()
	a.drawables = append(a.drawables, &drawable)
	a.changeMutex.Unlock()
	if resizable, ok := drawable.(Resizable); ok {
		resizable.Resize(a.ScreenX, a.ScreenY)
	}
}

func (a *App) RemoveDrawable(drawable Drawable) {
	if a.keepAlive[drawable] == true {
		return
	}
	a.changeMutex.Lock()
	index := -1
	for i, _ := range a.drawables {
		if a.drawables[i] == &drawable {
			index = i
			break
		}
	}
	if index >= 0 {
		a.drawables = append(a.drawables[:index], a.drawables[index+1:]...)
	}
	a.changeMutex.Unlock()
}

func (a *App) AddSystem(system System) {
	a.changeMutex.Lock()
	a.systems = append(a.systems, &system)
	a.changeMutex.Unlock()
}

func (a *App) RemoveSystem(system System) {
	if a.keepAlive[system] == true {
		return
	}
	a.changeMutex.Lock()
	index := -1
	for i, _ := range a.systems {
		if a.systems[i] == &system {
			index = i
			break
		}
	}
	if index >= 0 {
		a.systems = append(a.systems[:index], a.systems[index+1:]...)
	}
	a.changeMutex.Unlock()
}

func (a *App) SetDirty() {
	a.dirty <- true
}

func (a *App) ClearNow() {
	a.clear()
	a.screen.Clear()
	//a.screen.Sync()
}

func (a *App) RunOnAppRoutine(f func()) {
	a.runOnAppRoutine = append(a.runOnAppRoutine, f)
}

func (a *App) Quit() {
	a.quit = true
}

func (a *App) clear() {
	a.changeMutex.Lock()
	a.drawables = nil
	a.systems = nil
	a.changeMutex.Unlock()
	if len(a.keepAlive) > 0 {
		for s, _ := range a.keepAlive {
			// TODO - without locking?
			a.AddSystem(s.(System))
			a.AddDrawable(s.(Drawable))
		}
	}
}

func (a *App) processTerminalEvents() {
	for {
		if a.quit {
			return
		}
		ev := a.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			a.screen.Sync()
			x, y := a.screen.Size()
			a.ScreenX = x
			a.ScreenY = y
			for _, s := range a.drawables {
				if ft, ok := (*s).(Resizable); ok {
					ft.Resize(x, y)
				}
			}
			break
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC {
				a.Quit()
				return
			}
			if len(a.systems) == 0 && ev.Key() == tcell.KeyEscape {
				a.quit = true
			}
			// TODO - should keep only one array with components?
			for _, s := range a.systems {
				if ft, ok := (*s).(KeyListener); ok {
					go ft.HandleKeyEvent(ev)
				}
			}
		default:
			continue
		}
	}
}

func (a *App) processOsSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	go func() {
		<-signals
		a.quit = true
	}()
}
