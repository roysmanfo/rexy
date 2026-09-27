package main

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.yaml.in/yaml/v4"

	"github.com/fsnotify/fsnotify"

	cfgs "github.com/roysmanfo/rexy/internal/configs"
)

var (
	proxies    map[string]*httputil.ReverseProxy
	configs    *cfgs.RexyConf
	mu         sync.RWMutex
	routerPage string
)

//go:embed index.html
var defaultHTMLRouterPage string

func buildRouterPage(domainMapping map[string]*cfgs.RexyConfDomain) string {
	data, err := os.ReadFile("index.html")

	var page string
	if err != nil {
		page = defaultHTMLRouterPage
	} else {
		page = string(data)
	}

	sb := strings.Builder{}
	for domain, target := range domainMapping {
		targetUrl, _ := url.Parse(target.RedirectTo)
		sb.WriteString(strings.TrimSpace(fmt.Sprintf(`
<tr class="entry" data-domain="%s">
<td class="font-mono"><a href="https://%s/" target="_blank">%s</a></td>
	<td class="col-2"><span class="badge text-bg-primary font-mono">%s</span></td>                    
</tr>`, domain, domain, domain, targetUrl.Host)))
		sb.WriteString("\n")
	}

	return strings.Replace(page, "{{ .Domains }}", sb.String(), 1)
}

func buildProxies(domainMapping map[string]*cfgs.RexyConfDomain) (map[string]*httputil.ReverseProxy, error) {
	newProxies := make(map[string]*httputil.ReverseProxy)

	for incomingHost, internalURL := range domainMapping {
		targetURL, err := url.Parse(internalURL.RedirectTo)
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: configs.SSL.InsecureSkipVerify,
			},
		}

		target := targetURL

		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			if configs.RedirectAsHttps {
				req.URL.Scheme = "https"
			}

			req.URL.Host = target.Host

			req.Header.Set("Host", target.Host)
			// req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Real-IP", req.RemoteAddr)
			//req.Header.Set("Connection", "")
		}

		proxy.ModifyResponse = func(res *http.Response) error {
			location := res.Header.Get("Location")
			// overwrite the domain on the location
			if location != "" {
				locationUrl, err := url.Parse(location)
				if err != nil {
					return err
				}
				locationUrl.Host = incomingHost
				res.Header.Set("Location", locationUrl.String())
			}

			return nil
		}

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error for %s: %v", r.Host, err)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		}

		proxy.FlushInterval = -1

		newProxies[incomingHost] = proxy
	}

	log.Printf("loaded %d routes", len(domainMapping))
	for domain := range domainMapping {
		log.Println(" -", domain)
	}
	routerPage = buildRouterPage(domainMapping)
	return newProxies, nil
}

func loadConfigFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	var conf cfgs.RexyConf

	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &conf); err != nil {
			return fmt.Errorf("failed to parse config: %v", err)
		}
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(data, &conf); err != nil {
			return fmt.Errorf("failed to parse config: %v", err)
		}
	default:
		return fmt.Errorf("failed to parse config: unknown format based on file extension: '%s'", ext)
	}

	if ok, err := conf.IsValid(); !ok {
		return err
	}

	mu.Lock()
	configs = &conf
	mu.Unlock()

	newProxies, err := buildProxies(conf.Mapping)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	mu.Lock()
	proxies = newProxies
	mu.Unlock()

	port := conf.HttpPort
	if conf.UseHTTPS() {
		port = conf.HttpsPort
	}
	log.Printf("reverse proxy page available on %s (%s:%d)\n", conf.Domain, conf.BindToIp, port)

	return nil
}

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
					if err := loadConfigFromFile(path); err != nil {
						log.Println("error while loading new configs, running with old ones")
					}
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

func handlerReverseProxyPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" || r.URL.Path == "/" {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(routerPage))
		return
	}

	if r.URL.Path == "/cert" {

		if configs.SSL.CertificateFile == "" {
			w.WriteHeader(http.StatusNoContent)
			w.Write([]byte("no certificate found on this server"))
			return
		}

		filedata, err := os.ReadFile(configs.SSL.CertificateFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unable to fetch certificate, check server error log"))
			log.Printf("[ERR   ] Unable to read certificate file: %s", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/x-pem-file")
		w.Write([]byte(filedata))
		return
	}

	http.NotFound(w, r)
}

func main() {
	configPath := flag.String("conf", "conf.yml", "the path to the configuration file to use")
	noHotReload := flag.Bool("no-hot-reload", false, "disable automatic config file reload when modified")

	flag.Parse()

	// Initial load
	if err := loadConfigFromFile(*configPath); err != nil {
		log.Println(err)
		return
	}
	if !*noHotReload {
		// Start watcher
		watchConfig(*configPath)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := net.ParseIP(r.URL.Hostname())
		if r.Host == configs.Domain || r.Host == "localhost" || (ip != nil && ip.IsLoopback()) {
			handlerReverseProxyPage(w, r)
			return
		}

		mu.RLock()
		proxy, ok := proxies[r.Host]
		mu.RUnlock()

		if !ok {
			http.NotFound(w, r)
			log.Printf("[ERR   ] A client asked for a domain not registered on this reverse proxy: %s", r.Host)
			return
		}

		log.Printf("[%-6s][%-21s] %s%s", r.Method, r.RemoteAddr, r.Host, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	if configs.HttpsPort != 0 || configs.UseHTTPS() {
		httpsListener(handler)
	}

	if configs.HttpPort != 0 {
		httpListener(handler)
	}

}

func httpsListener(handler http.HandlerFunc) {
	if configs.HttpsPort == 0 {
		return
	}

	addr := net.JoinHostPort(configs.BindToIp, fmt.Sprint(configs.HttpsPort))
	log.Println("started HTTPS server on", addr)
	log.Fatalln(http.ListenAndServeTLS(
		addr,
		configs.SSL.CertificateFile,
		configs.SSL.PrivateKeyFile,
		handler,
	))
}

func httpListener(handler http.HandlerFunc) {
	if configs.HttpPort == 0 {
		return
	}

	h := http.NewServeMux()
	addr := net.JoinHostPort(configs.BindToIp, fmt.Sprint(configs.HttpPort))
	server := http.Server{
		Addr:    addr,
		Handler: h,
	}

	if configs.UseHTTPS() {
		if configs.SSL.ForceSecureConection {
			// force redirect to use https
			h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				url := "https://" + configs.Domain + r.URL.RequestURI()
				http.Redirect(w, r, url, http.StatusMovedPermanently)
			})
		}
	} else {
		h.Handle("/", handler)
	}

	log.Println("started HTTP server on", addr)
	log.Fatalln(server.ListenAndServe())
}
