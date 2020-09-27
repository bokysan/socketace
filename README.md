# SocketAce

**Your ultimate connection proxy.** websocket proxy. secure sockets proxy.

Ever had an issue with restrictive firewalls? Well, this tool will help out. This tool
allows you to proxy *multiple* connections through:
- sockets
- TLS-encrypted sockets (direct replacement for [stunnel](https://www.stunnel.org/))
- websockets
- TLS-encrypted websockets

## Contents


1. [Rationale](#rationale)
1. [Usage](#usage)
    1. [Name](#name)
    1. [Synopsis](#synopsis)
    1. [Description](#description)
    1. [Examples](#examples)
    1. [See also](#see-also)
1. [TO-DO](#to-do)

## Rationale

There are several use cases where SocektAce might come in handy:

- **Encrypting connections** If your protocol does not support encryption, you can simply
  wrap the connection with SocketAce and pass it over the Internet. 

- **Restrictive firewalls** Sometimes you might find yourself behind a quite restrictive 
  firewall. The firewall might:
  - let through only specific ports (80, 443)
  - use deep packet inspection and block non-HTTP / non-HTTPs traffic
  
- **Expose Unix sockets as TCP streams** If your service is only available as a Unix socket,
  you can use SocketAce to expose it on a host and access it from other (even Windows) servers
  
  
## Usage

### Name

**socketace** - A tool for tunneling connections over the internet
    
### Synopsis

For the server:

```
socketace server 
    [--help] 
    [-v[v[v[v[v[v]]]]]]
    [-c|--config <yaml-config-file>]
    [-l|--log-file <log-file>]
    [-f|--log-format text|json]
    [-C|--log-color yes|no|true|false|auto]
    [--log-full-timestamp]
    [--log-report-caller]
```

For the client:
```
socketace client
    [--help] 
    [-v[v[v[v[v[v]]]]]]
    [-c|--config <yaml-config-file>]
    [-l|--log-file <log-file>]
    [-f|--log-format text|json]
    [-C|--log-color yes|no|true|false|auto]
    [--log-full-timestamp]
    [--log-report-caller]
    [--ca-certificate <string> | --ca-certificate-file=<file>]
    [--certificate <string> | --certificate-file=<file>]
    [--private-key <string> | --private-key-file=<file>]
    [--private-key-password <string> | --private-key-password-program=<string>]
    [-k|--insecure]
    [-l|--listen <string>]...
    [-u|--upstream <string>...
```

### Description

SocketAce can proxy multiple protocol across a single connection. You need to pick the
right protocol when setting up a connection on the client.

#### Server

The server can listen on multiple ports / protocols at the same time. To configure
the server, you need to set up:
- [Upstreams](#channels-upstreams)
- [Severs](#servers)

##### Channels (upstreams)

Upstreams are configured in the YAML `upstreams` section. They define the external services
that will be accessible through this server setup.

```yaml
server:
  channels:
    - name: <service-name>
      network: <network>
      address: <address>
    ...
```

You may define multiple channels (upstreams). Each cannel needs the following properties:

- `name` is the unique name given to this upstream server. This is then referenced later
  on in the `servers` section and on the client. A good example would be `ssh`, `web`, `oracle` etc.
- `network` is the protocol used to connect to the backend. Possible options are: `tcp` for Internet,
  and `unix` and `unixgram` for unix sockets.
- `address` is the address of the upstream. For `tcp` this is the host and the port, e.g. `127.0.0.1:22`,
  `www.google.com:80` or ˙[::1]:8080`. For unix sockets, this is the path to the socket.
 
##### Servers
 
At this stage, the following `kinds` (protocols) are supported: `websocket`, `tcp`, `stdin` and `unix`. 
To configure the server, add it to the `servers` section of the configuration.

```yaml
server:
  servers:
    - kind: <kind>
      [listen: <address>]
      [
      endpoints:
        - endpoint: <url-part>
        ...
      ]
      [channels: [list of channels]]
      [caCertificate: <ca-certificate>]
      [caCertificateFile: <ca-file>]
      [certificate: <certificate>]
      [certificateFile: <certificate-file>]
      [privateKey: <private-key>]
      [privateKeyFile: <private-key-file>]
      [privateKeyPassword: <private-key-password>]
      [privateKeyPasswordProgram: <private-key-password-program>]
```

Where:

- `kind` is the type of server to run. Can be `websocket`, `tcp`, `stdin` and `unix`.
  - All services other than `stdin` require the listening configuration
  - `stdin` listens on stdin/stdout. As expected, only one `stdin` server can be configured. This allows you to
    use SocketAce via `ssh` (like [rsync](https://en.wikipedia.org/wiki/Rsync) over `ssh`) or any other service
    that can stream standard input and output
- `endpoints` is only required or `websocket` server. It defines the list of URLs the server should listen to.
  For example `/ws/all` or `/my/secret/connection`. You may listen on multiple URLs.
- `channels` defines a list of upstream channels that this connection proxies. If not defined, all channels are 
  proxied.
- `caCertificate` or `caCertificateFile` may be defined if you want to use mutual (client and server) certificate
  authentication. When defined, the server will accept client connections only if signed by the given CA certificate. 
- `certificate` or `certificateFile` is server's certificate. When defined, the server automatically switches to
  TLS connection. Applicable for all server kinds. 
- `privateKey`, `privateKeyFile`, `privateKeyPassword` and `privateKeyPasswordProgram` should be pretty 
  self-explanatory. They must be defined when `certificate` is set up. 

#### Client

Client configuration is a bit simple and can be done via a config file or via a command line. Basically, only
two options are important:

- `--upstream <url>` may be specified multiple times. Defines a list of upstream servers that the client will 
  try to connect to. The format is `<protocol>[://<host|path>]`. Protocol may be any of the following: `tcp`, 
  `tcp+tls`, `stdin`, `stdin+tls`, `unix`, `unix+tls`, `http`, `https`. Examples:
  - `tcp://127.0.0.1:9995` to connect to a socket server on `localhost` on `9995` 
  - `tcp+tls://127.0.0.1:9995` to connect to a TLS-encrypted socket server on `localhost` on `9995` 
  - `stdin` to connect to server through standard input / output
- `--listen <channel>~<listen-url>[~<forward-url>]` will open a listening socket on the client. 
  - `channel` name must be the same as defined on the server. 
  - `listen-url` is the protocol and the host/path to listen on. Protocol may be `tcp`, `unix` and `stdin` 
  - `foward-url` is the optional direct address of the service. If specified, the client will try to connect
    to this service directly first and, failing that, start going through upstream services.

 
### Examples

#### Server setup

The easiest way to setup a server is with a YAML file. The `examples` directory contains a configuration which
provides different server setups.

#### Client setup

##### Use socketace as a simple telnet client  

```sh
socketace -k --upstream tcp+tls://server.example.com:80 --upstream https://server.example.com/proxy --listen smtp~stdin
```

##### Use socketace to SSH to your server from anywhere

```sh
ssh localhost -o ProxyCommand='socketace --upstream http://127.0.0.1:9999/ws/all --listen ssh~stdin'
```

##### Use socketace to proxy IMAP and SMTP

```sh
socketace -e tcp+tls://server.example.com:80 --listen imap~127.0.0.1:143 --listen imap~127.0.0.2:587
```

## TO-DO

There's still some things to be done. If anybody's willing to pick up issues, pull
requests are welcome:
- add proxying of UDP connections
- add functionality similar to [sslh](https://github.com/yrutschle/sslh) to be able to
  "hide" the proxy and share the port with other services
- add the possibility of proxying connections through DNS, similar to [iodine](https://github.com/yarrick/iodine)
  works 