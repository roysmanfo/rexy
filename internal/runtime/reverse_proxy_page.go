package runtime

import (
	"log"
	"net/http"
	"os"

	"github.com/roysmanfo/rexy/internal/configs"
)

func CreateReverseProxyPageHandler(c *configs.RexyConf, routerPage *string) http.HandlerFunc {
	h := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(*routerPage))
			return
		}

		if r.URL.Path == "/cert" {

			if c.SSL.CertificateFile == "" {
				w.WriteHeader(http.StatusNoContent)
				w.Write([]byte("no certificate found on this server"))
				return
			}

			filedata, err := os.ReadFile(c.SSL.CertificateFile)
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
	return h
}
