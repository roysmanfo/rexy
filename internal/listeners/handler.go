package listeners

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/roysmanfo/rexy/internal/configs"
)

func CreateProxyHandler(c *configs.RexyConf, proxies *sync.Map, handlerReverseProxyPage http.HandlerFunc) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := net.ParseIP(r.URL.Hostname())
		if r.Host == c.Domain || r.Host == "localhost" || (ip != nil && ip.IsLoopback()) {
			handlerReverseProxyPage(w, r)
			return
		}

		proxy, ok := proxies.Load(r.Host)

		if !ok {
			http.NotFound(w, r)
			log.Printf("[ERR   ] A client asked for a domain not registered on this reverse proxy: %s", r.Host)
			return
		}

		log.Printf("[%-6s][%-21s] %s%s", r.Method, r.RemoteAddr, r.Host, r.URL.Path)
		proxy.(*httputil.ReverseProxy).ServeHTTP(w, r)
	})
}
