package runtime

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/roysmanfo/rexy/internal/configs"
)

func BuildRouterPage(domainMapping map[string]*configs.RexyConfDomain, htmlPageTemplate string) string {
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

	return strings.Replace(htmlPageTemplate, "{{ .Domains }}", sb.String(), 1)
}
