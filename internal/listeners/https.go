package listeners

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/roysmanfo/rexy/internal/configs"
)

func ServeHTTPS(c *configs.RexyConf, handler http.HandlerFunc) {
	if c.HttpsPort == 0 {
		return
	}

	addr := net.JoinHostPort(c.BindToIp, fmt.Sprint(c.HttpsPort))
	log.Println("started HTTPS server on", addr)
	log.Fatalln(http.ListenAndServeTLS(
		addr,
		c.SSL.CertificateFile,
		c.SSL.PrivateKeyFile,
		handler,
	))
}
