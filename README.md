# Proxy utility for RIP

## Features

- Logging proxy (no HTTPS interception, use CONNECT method)
- Simple HTTP-only reverse proxy
- WPAD server

## Configuration

The configuration file is in YAML format. The following sections are defined.

### Logging

Configure the logs.

```yaml
logging:                       # configure logging properties
  level: debug                 # log level (trace,debug,info,warning,error)
  file: /var/log/riproxy.log   # log file (stderr if not set)
```

#### Level (level)

The log level. Must be one of the following :

- error: used for errors that should definitely be noted. 
- warning (or warn): non-critical entries that deserve eyes.
- info: general operational entries about what's going on inside the application.
- debug: usually only enabled when debugging. Very verbose logging.
- trace: designates finer-grained informational events than the Debug.

#### File (file)

The log file. Use stderr if not set.

#### Log MAC address (log_mac_address)

A boolean (true/false). If true, the HTTP services and Proxy services will log client MAC address.

#### Log file max size (file_max_size)

Maximum size in megabytes of the log file before it gets rotated.

#### Log file max age (file_max_age)

Maximum number of days to retain old log files.

#### Log file max backups (file_max_backups)

Maximum number of old log files to retain.

### Defaults (defaults)

#### HTTP service (http)

Configures the default values for the HTTP service (WPAD and HTTP-only reverse proxy).

##### Port (port)

Listening TCP port of the HTTP service.

##### WPAD (wpad)

Configures the default values for the WPAD service.

###### External Proxy (external_proxy)

If an external proxy must be defined by the WPAD file, configure this key with a connection string: `IP or FQDN`:`port`

###### Direct networks (direct_networks)

A list of networks in CIDR format or local interface names that will bypass the proxy in the WPAD file.

###### Listening interface is direct (direct)

A boolean (true/false) that indicates if the WPAD service listening interface network should be added to the direct network list.

#### Proxy service (proxy)

Configures the default values for the Proxy service.

##### Port (port)

Listening TCP port of the Proxy service.

##### Block by Internationalized Domain Name (block_by_idn)

A boolean (true/false) that indicates if the list of blocked domains should be normalized in IDN format.

##### List of blocked domains (block)

A list of FQDN to block.

##### Allow high TCP ports (allow_high_port)

A boolean (true/false) that indicates if connections to TCP ports higher than 1024 should be allowed.

##### Allow low TCP ports (allow_low_port)

A boolean (true/false) that indicates if connections to TCP ports lower than (or equal to) 1024 should be allowed.

HTTP connections through port 80 will always be allowed.
HTTPS connections through port 443 will always be allowed.

##### Block raw IPs (block_ips)

A boolean (true/false) that indicates if direct connection to IP (not FQDNs) will be allowed.

##### Block local services (block_local_service)

Block access to servers exposed by the local computer through the proxy service.

##### Allowed HTTP methods (allowed_methods)

List of HTTP method allowed in proxy requests. If the CONNECT method is not allowed, no HTTPS connection will be allowed.

The default allowed methods are :
```
	// Standard methods
	GET,
	HEAD,
	POST,
	PUT,
	PATCH,
	DELETE,
	CONNECT,
	OPTIONS,
	// WebDAV methods
	COPY, // copy a resource from one URI to another
	LOCK, // put a lock on a resource
	MKCOL, // create collections (a.k.a. a directory)
	MOVE, // move a resource from one URI to another
	PROPFIND, // retrieve properties from a web resource
	PROPPATCH, // change and delete multiple properties on a resource in a single atomic act
	UNLOCK, // remove a lock from a resource
	// WebDAV ACL methods
	ACL, // modify the access control list of a resource
	// WebDAV versioning
	REPORT, // obtain information about a resource
	VERSION-CONTROL, // create a version-controlled resource
	CHECKOUT, // allow modifications to the content and dead properties of a checked-in version-controlled resource
	CHECKIN, // produce a new version whose content and dead properties are copied from the checked-out resource
	UNCHECKOUT, // cancel the CHECKOUT and restore the pre-CHECKOUT state of the version-controlled resource
	MKWORKSPACE, // create a new workspace resource
	UPDATE, // modify the content and dead properties of a checked-in version-controlled resource
	LABEL, // modify the labels that select a version
	MERGE, // perform the logical merge of a specified version into a specified version-controlled resource
	BASELINE-CONTROL, // place a collection under baseline control
	MKACTIVITY, // create a new activity resource
	SEARCH, // initiate a server-side search
	// WebDAV collection ordering
	ORDERPATCH, // change the ordering semantics of a collection
	// CalDAV methods
	MKCALENDAR, // create a new calendar collection resource
```

### Listening interfaces (interfaces)

Map of configurations of listening interface.

```yaml
- interfaces:
  eth0:
    http:
      port:80
    proxy:
      port: 3128
```

#### Name of the interface (name)

Name of the interface to listen on.

#### HTTP service configuration (http)

Configures the default values for the HTTP service (WPAD and HTTP-only reverse proxy).

##### Port (port)

Listening TCP port of the HTTP service.

##### WPAD (wpad)

Configures the WPAD service.

###### Enable (enable)

A boolean (true/false) that indicates if the WPAD service is enabled on this interface.

###### External Proxy (external_proxy)

If an external proxy must be defined by the WPAD file, configure this key with a connection string: `IP or FQDN`:`port`

###### Direct networks (direct_networks)

A list of networks in CIDR format or local interface names that will bypass the proxy in the WPAD file.

These networks will be added to the defaults if defined.

###### Listening interface is direct (direct)

A boolean (true/false) that indicates if the WPAD service listening interface network should be added to the direct network list.

##### HTTP-only reverse proxies (reverse_proxies)

Associative array of host names and reverse proxy configuration that will listen on this interface.

```yaml
reverse_proxies:
  www.example.com:
    peer_ip: 192.0.2.2
    peer_port: 80
    source_interface: eth0
```

###### Peer IP address (peer_ip)

The IP address of the destination IP.

###### Peer TCP port (peer_port)

The destination port. The default port is 80.

###### Source interface (source_interface)

(Optional). The source interface of the server side connection.

###### Allowed HTTP methods (allowed_methods)

List of HTTP method allowed in requests.

The default allowed methods are :
```
	GET,
	HEAD,
	POST,
	PUT,
	PATCH,
	DELETE,
	OPTIONS,
```

#### Proxy service (proxy)

Configures the Proxy service.

##### Enable (enable)

A boolean (true/false) that indicates if the Proxy service is enabled on this interface.

##### Port (port)

Listening TCP port of the Proxy service.

##### Block by Internationalized Domain Name (block_by_idn)

A boolean (true/false) that indicates if the list of blocked domains should be normalized in IDN format.

##### List of blocked domains (block)

A list of FQDN to block.

These domains will be added to the defaults if defined.

##### Allow high TCP ports (allow_high_port)

A boolean (true/false) that indicates if connections to TCP ports higher than 1024 should be allowed.

If this setting is false and the default is true, the resulting setting is true.

##### Allow low TCP ports (allow_low_port)

A boolean (true/false) that indicates if connections to TCP ports lower than (or equal to) 1024 should be allowed.

HTTP connections through port 80 will always be allowed.
HTTPS connections through port 443 will always be allowed.

If this setting is false and the default is true, the resulting setting is true.

##### Block raw IPs (block_ips)

A boolean (true/false) that indicates if direct connection to IP (not FQDNs) will be allowed.

If this setting is false and the default is true, the resulting setting is true.

##### Block local services (block_local_service)

Block access to servers exposed by the local computer through the proxy service.

##### Allowed HTTP methods (allowed_methods)

List of HTTP method allowed in proxy requests. If the CONNECT method is not allowed, no HTTPS connection will be allowed.

This setting replaces the default if defined.
