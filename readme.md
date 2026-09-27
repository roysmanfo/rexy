# Rexy
Simple, light and fast YAML based reverse proxy.  
Easily redirect requests to internal services, or private domains.

## Features

- **HTTPS support**
  Rexy supports https if an SSL certificate is provided
- **Hot Reload**
  Rexy supports https if an SSL certificate is provided


## Usage
1. Start a simple http server (ex. `python3 -m http.server` or `bunx serve`) in the [sample website folder](./website/)
2. Complete the [`conf.yml`](./conf.yml) file with your own domain mappings
3. Run `go run ./cmd/ -conf ./conf.yml` to start Rexy and monitor the logs
4. Visit the configured URL in your browser (you may need to configure your DNS to point to Rexy)

## Important notice
On linux, Rexy might not be able to bind to well-known ports by default, unless elevated provileges are given.  
To bind to those ports, the recomended way is to only allow this access, without giving it root privileges
```
sudo setcap 'cap_net_bind_service=+ep' /path/to/rexy
```
or for a service, you can append this string to your `rexy.service` file 
```
AmbientCapabilities=CAP_NET_BIND_SERVICE
```



