package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"

	cfgs "github.com/roysmanfo/rexy/internal/configs"
	"github.com/roysmanfo/rexy/internal/listeners"
	"github.com/roysmanfo/rexy/internal/runtime"
)

var (
	proxies = sync.Map{} // map[string]*httputil.ReverseProxy{}
	configs = &cfgs.RexyConf{}

	mu         sync.RWMutex
	routerPage string
)

// This variable will be overwritten at build time
var version = "1.0-dev"

//go:embed index.html
var defaultHTMLRouterPage string

func watchConfig(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					log.Println("config changed, reloading...")

					mu.Lock()
					if err := cfgs.LoadConfigFromFile(path, configs, &proxies); err != nil {
						log.Println("error while loading new configs, running with old ones")
					} else {
						routerPage = runtime.BuildRouterPage(configs.Mapping, defaultHTMLRouterPage)
					}
					mu.Unlock()
				}
			case err := <-watcher.Errors:
				log.Println("watcher error:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	configPath := flag.String("conf", "conf.yml", "the path to the configuration file to use")
	noHotReload := flag.Bool("no-hot-reload", false, "disable automatic config file reload when modified")
	versionFlag := flag.Bool("version", false, "print the current version and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("rexy %s\n", version)
		return
	}

	// Initial load
	if err := cfgs.LoadConfigFromFile(*configPath, configs, &proxies); err != nil {
		log.Println(err)
		return
	}
	if !*noHotReload {
		// Start watcher
		watchConfig(*configPath)
	}

	handlerReverseProxyPage := runtime.CreateReverseProxyPageHandler(configs, &routerPage)
	handler := listeners.CreateProxyHandler(configs, &proxies, handlerReverseProxyPage)

	if configs.HttpsPort != 0 || configs.UseHTTPS() {
		listeners.ServeHTTPS(configs, handler)
	}

	if configs.HttpPort != 0 {
		listeners.ServeHTTP(configs, handler)
	}

}
