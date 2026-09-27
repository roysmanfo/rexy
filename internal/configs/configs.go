package configs

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	DOMAIN_HTTPS_AUTO  = "auto"
	DOMAIN_HTTPS_FORCE = "force"
	DOMAIN_HTTPS_OFF   = "off"
)

type RexyConfDomain struct {
	Name       string `json:"name" yaml:"name"`
	RedirectTo string `json:"redirect_to" yaml:"redirect_to"`
	Https      string `json:"https" yaml:"https"`
}

type RexyConfSLL struct {
	CertificateFile      string `json:"certificate_file" yaml:"certificate_file"`
	PrivateKeyFile       string `json:"private_key_file" yaml:"private_key_file"`
	ForceSecureConection bool   `json:"force_secure_connection" yaml:"force_secure_connection"` // redirect requests to the http_port to the https_port
	InsecureSkipVerify   bool   `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

// RexyConf tecnically also supports Json, and if the config file passed is .json, it will load,
// but for now YAML is the focus
type RexyConf struct {
	Domain          string                     `json:"domain" yaml:"domain"`
	BindToIp        string                     `json:"bind_to_ip" yaml:"bind_to_ip"`
	HttpPort        int                        `json:"http_port" yaml:"http_port"`
	HttpsPort       int                        `json:"https_port" yaml:"https_port"`
	RedirectAsHttps bool                       `json:"redirect_as_https" yaml:"redirect_as_https"`
	SSL             RexyConfSLL                `json:"ssl" yaml:"ssl"`
	Mapping         map[string]*RexyConfDomain `json:"mapping" yaml:"mapping"`
}

func (r *RexyConf) IsValid() (valid bool, reason error) {
	if _, err := url.Parse(r.Domain); err != nil {
		return false, err
	}

	if res := net.ParseIP(r.BindToIp); res == nil {
		return false, fmt.Errorf("invalid ip address %s", r.BindToIp)
	}

	if r.HttpPort < 0 || r.HttpPort >= 0xffff {
		return false, fmt.Errorf("port is invalid")
	}

	if r.HttpsPort < 0 || r.HttpsPort >= 0xffff {
		return false, fmt.Errorf("port is invalid")
	}

	for external, internal := range r.Mapping {
		if _, err := url.Parse(external); err != nil {
			return false, fmt.Errorf("unable to parse external domain `%s`: %v", external, err)
		}
		if _, err := url.Parse(internal.RedirectTo); err != nil {
			return false, fmt.Errorf("unable to parse internal domain `%s`: %v", internal, err)
		}

		switch internal.Https {
		case "":
			internal.Https = DOMAIN_HTTPS_AUTO
		case DOMAIN_HTTPS_AUTO, DOMAIN_HTTPS_FORCE, DOMAIN_HTTPS_OFF:
		default:
			return false, fmt.Errorf("invalid https field for domain `%s`: %s", external, internal.Https)
		}

		// normalize strings to be all lowercase
		delete(r.Mapping, external)
		normalizedName := strings.ToLower(external)
		normalizedRedirect := strings.ToLower(internal.RedirectTo)
		r.Mapping[normalizedName] = internal
		internal.Name = normalizedName
		internal.Name = normalizedRedirect
	}

	if r.SSL.CertificateFile != "" {
		var err error
		if r.SSL.CertificateFile, err = filepath.Abs(r.SSL.CertificateFile); err != nil {
			return false, fmt.Errorf("unable to expand path for %s: %s", r.SSL.CertificateFile, err)
		}
		if r.SSL.PrivateKeyFile, err = filepath.Abs(r.SSL.PrivateKeyFile); err != nil {
			return false, fmt.Errorf("unable to expand path for %s: %s", r.SSL.PrivateKeyFile, err)
		}

		if _, err := os.Stat(r.SSL.CertificateFile); err != nil {
			return false, fmt.Errorf("unable to load SSL certificate file: `%s`: %v", r.SSL.CertificateFile, err)
		}
		if _, err := os.Stat(r.SSL.PrivateKeyFile); err != nil {
			return false, fmt.Errorf("unable to load SSL private key file: `%s`: %v", r.SSL.PrivateKeyFile, err)
		}
	}

	return true, nil
}

// ResolveIncommingHost returns a new [url.URL] pointer to the mapped domain
//
// example:
// "dashboard.lan" maps to "https://localhost:8080"
//
//	ResolveIncommingHost("dashboard.lan")
//	/* will return  */
//	&url.URL{Scheme: "https", Host: "localhost:8080"}
func (r *RexyConf) ResolveIncommingHost(host string) *url.URL {
	res := r.Mapping[strings.ToLower(host)]
	if res == nil {
		return nil
	}
	u, _ := url.Parse(res.RedirectTo)
	return u
}

func (r *RexyConf) GetDomainConfForHost(host string) *RexyConfDomain {
	return r.Mapping[strings.ToLower(host)]
}

// ! NOT IMPLEMENTED
// func (r *RexyConf) ResolveProtocolForHost(host string) *RexyConfDomain {
// 	return r.Mapping[strings.ToLower(host)]
// }

func (r *RexyConf) UseHTTPS() bool {
	return r.HttpsPort != 0 && (r.SSL.ForceSecureConection || r.SSL.CertificateFile != "")
}
