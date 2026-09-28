package configs

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

func buildProxies(c *RexyConf) (*sync.Map, error) {

	// newProxies := make(map[string]*httputil.ReverseProxy)
	newProxies := sync.Map{}

	for incomingHost, internalURL := range c.Mapping {
		targetURL, err := url.Parse(internalURL.RedirectTo)
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.SSL.InsecureSkipVerify,
			},
		}

		target := targetURL

		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			if c.RedirectAsHttps {
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

		newProxies.Store(incomingHost, proxy)
	}

	log.Printf("loaded %d routes", len(c.Mapping))
	for domain := range c.Mapping {
		log.Println(" -", domain)
	}

	return &newProxies, nil
}
