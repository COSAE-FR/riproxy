# Proxy utility for RIP

## Features

- Logging proxy
- Simple HTTP-only reverse proxy
- WPAD server

## Configuration

Example configuration file
```yaml
logging:                       # configure logging properties
  level: debug                 # log level (trace,debug,info,warning,error)
  file: /var/log/riproxy.log   # log file (stderr if not set)
interfaces:                    # Configure interfaces
  - name: wlp59s0              # interface name
    bind: :80                  # connection string for the WPAD/Reverse proxy server
    proxy: 192.168.1.23:3128   # external proxy connection string, replace the IP with `self` to enable internal proxy feature                
    networks:                  # WPAD: network with DIRECT access (no proxy)
      - 10.0.0.0/8
    direct: true               # WPAD: this interface subnet will be added to the networks list
    reverse_proxy:                # Reverse proxy configuration
      test.example.org:           # reverse proxy domain
        source_interface: virbr0  # source interface for the server side connection
        destination: 192.168.1.73:8000  # server side destination
```