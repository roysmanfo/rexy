package listeners

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/roysmanfo/rexy/internal/configs"
)

func ServeHTTP(c *configs.RexyConf, handler http.HandlerFunc) {
	if c.HttpPort == 0 {
		return
	}

	h := http.NewServeMux()
	addr := net.JoinHostPort(c.BindToIp, fmt.Sprint(c.HttpPort))
	server := http.Server{
		Addr:    addr,
		Handler: h,
	}

	if c.UseHTTPS() {
		if c.SSL.ForceSecureConection {
			// force redirect to use https
			h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				url := "https://" + c.Domain + r.URL.RequestURI()
				http.Redirect(w, r, url, http.StatusMovedPermanently)
			})
		}
	} else {
		h.Handle("/", handler)
	}

	log.Println("started HTTP server on", addr)
	log.Fatalln(server.ListenAndServe())
}
