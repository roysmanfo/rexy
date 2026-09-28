package configs

// these variables will be populated at build time
// or on application startup

var defaultHTMLRouterPage string = ""

func SetDefaultHTMLRouterPage(p string) {
	defaultHTMLRouterPage = p
}
