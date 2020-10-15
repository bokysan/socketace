# SocketAce [![Docker image](https://github.com/bokysan/socketace/workflows/Docker/badge.svg)](https://github.com/bokysan/socketace/actions?query=workflow%3A%22Docker%22)] [![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbokysan%2Fsocketace.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbokysan%2Fsocketace?ref=badge_shield)

**Your ultimate connection tunnel.** TCP websocket tunnel. TLS sockets tunnel. Serial connection socket tunnel. One 
executable for the client and the server. Multiple platforms supported. Written in [go](https://golang.org).

Ever had an issue with restrictive firewalls? Well, this tool will help out. Socketace allows you to tunnel *multiple* 
connections through:
- sockets
- TLS-encrypted sockets (direct replacement for [stunnel](https://www.stunnel.org/))
- websockets
- TLS-encrypted websockets

Socketace is mainly meant for restricted environments where the firewall won't allow you to open an SSH connection or
even won't allow any other traffic other than on port 80 and 443. Socketace is also able to tunnel the connection
through HTTP (websockets) so even firewalls that do deep packet inspection / proxy ports 80 and 443 should work fine.

Unlike other solutions which use [`HTTP CONNECT`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/CONNECT) to
establish connection, `socketace` will actually overlay the TCP over HTTP.

**DISCLAIMER:** Please note that you are using socketace at your own risk.

SocketAce will use *one pyhiscal connection* and overlay multiple *logical connections* within that connection:

```

   LAYER            CLIENT                        TUNNEL                          SERVER
+--------------+  
| LOGICAL      |    Map channels and listen on:                                   Set up channels forward to:
+--------------+    - TCP socket                                                  - TCP socket
                    - Unix socket                                                 - Unix socket
                    - standard input/output
+--------------+
| CHANNEL      |                       <- PICK PROPER CHANNEL / SERVICE ->
+--------------+
+--------------+
| MULTIPLEX    |              <- MULTIPLEX MULTIPLE CONNECTIONS OVER ONE CONNECTION  ->
+--------------+
+--------------+
| SECURITY     |                      <- SECURE CONNECTION VIA STARTTLS  ->
+--------------+
+--------------+
| CONNECTIVITY |                               Tunnel via via:
+--------------+                               - simple sockets (TCP or Unix)
                                               - TLS-encrypted sockets (TCP or Unix)
                                               - packet sockets (UDP or UnixPacket)
                                               - websockets on plain HTTP
                                               - websockets on TLS-encrpyted HTTPS
                                               - standard input/output

```

This allows you to do wild combinations, such as:
- listen on a local TCP socket, forward connection via SSH + standard in/out to remote server to a Unix socket 
  (e.g. `rsync -essh` works)
- listen on a local TCP socket, wrap the connection TLS and forward to a service on a remote server 
  (i.e. replicate what `stunnel` does) 
- listen on a local standard in/out, forward to remote service via websocket
  (i.e. "expose ssh via websockets")

## Contents

1. [Rationale](#rationale)
1. [Installation](#installation)
    1. [Install using docker](#install-using-docker)
    1. [Install using brew](#install-using-brew)
    1. [Install on Linux using a package manager](#install-on-linux-using-a-package-manager)
    1. [Manual install](#manual-install)
1. [Usage](#usage)
    1. [Name](#name)
    1. [Synopsis](#synopsis)
    1. [Description](#description)
    1. [Examples](#examples)
1. [Caveats](#caveats)
    1. [Connecting to a secure (TLS-enabled) service](#connecting-to-a-secure-tls-enabled-service)
1. [TO-DO](#to-do)
1. [Similar projects](#similar-projects)

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
  
## Installation

This software uses [goreleaser](https://goreleaser.com/) and [buildx](https://docs.docker.com/buildx/working-with-buildx/)
to create software distribution. There are several ways to install it:

### Install using docker

The simplest way to use SOCKETACE is by referencing a pre-build [docker image](https://hub.docker.com/repository/docker/boky/socketace), 
e.g.

```shell script
docker run --rm -it boky/socketace
```

### Install using brew

```shell script
brew tap bokysan/socketace https://github.com/bokysan/socketace-brew.git
brew install socketace
```

### Install on Linux using a package manager

The build system provides RPM, DEB and APK packages:

1. Go to [Releases](https://github.com/bokysan/socketace/releases) page.
2. Download the version appropriate for your system.
3. Execute install for your distribution, e.g. `dpkg -i <package>.deb`


### Manual install

To install manually:

1. Go to [Releases](https://github.com/bokysan/socketace/releases) page.
2. Download the version appropriate for your system into `$HOME/bin` or similar.


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
    [-s|--secure]
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

##### Channels

Channels are configured in the YAML `channels` section. They define the external services that will be accessible 
through this server setup.

```yaml
server:
  channels:
    - name: <service-name>
      address: <address>
    ...
```

You may define multiple channels (upstreams). Each cannel needs the following properties:

- `name` is the unique name given to this upstream server. This is then referenced later
  on in the `servers` section and on the client. A good example would be `ssh`, `web`, `oracle` etc.
- `address` is the address of the upstream. For `tcp` this is the host and the port, e.g. `tcp://127.0.0.1:22`,
  `tcp://www.google.com:80` or `tcp://[::1]:8080`, `unix:///var/sock/app.sock`, `unixpacket:///var/sock/app.sock`.
 
##### Servers
 
At this stage, the following `kinds` (protocols) are supported: `websocket`, `tcp`, `stdin` and `unix`, `unixpacket`
and `udp`.  To configure the server, add it to the `servers` section of the configuration.

```yaml
server:
  servers:
    - address: <address>
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

- `address` is the type of server and listening location. Can be `http`, `https`, `tcp`, `tcp+tls`, `stdin`
    `stdin+tls`, `unix` or `unix+tls`, `udp` and `unixpacket`.
  - Always use a valid url, e.g. `tcp://0.0.0.0:5000`, `https://0.0.0.0:8900`.
  - Address type will define the listening server style, e.g.
    - `http` and `https` will start an HTTP / websocket server, 
    - `tcp` and `unix` will start a standard socket server,
    - `udp` and `unixgram` will start a packet socket server,
    - `stdin` will start a stream on standard input/output.
  - `stdin` and `stdin+tls` listen to stdin/stdout. As expected, only one `stdin` server can be configured. This allows
    you to use SocketAce via `ssh` (like [rsync over `ssh`](https://en.wikipedia.org/wiki/Rsync)) or any other service
    which can stream via standard input and output (e.g. via `telnet` or `netcat` or even serial connection).
  - TLS-secured tunnels will need the certificate info.
  - You can also listen on a non-secured channel (e.g. HTTP) and provide certificate info. If provided, server will
    support the `StartTLS` command, which executes TLS handshake after connecting. Especially useful if you're proxying
    the connection over an existing HTTP server.
- `endpoints` is only required or `websocket` server. It defines the list of URLs the server should listen to.
  For example `/ws/all` or `/my/secret/connection`. You may listen on multiple URLs.
- `channels` defines a list of upstream channels that this connection proxies. If not defined, all channels are 
  proxied.
- Define `caCertificate` or `caCertificateFile` if you want to use mutual (client and server) certificate
  authentication. When defined, the server will accept client connections only if signed by the given CA certificate. 
- `certificate` or `certificateFile` is the server's certificate. Needed for `tls` connections. If provided for non-TLS
  connections, server will suggest to the client to switch to secure communication via `StartTLS`.  
- `privateKey`, `privateKeyFile`, `privateKeyPassword` and `privateKeyPasswordProgram` should be pretty 
  self-explanatory. They must be defined when `certificate` is set up. 

#### Client

Client configuration is a bit simple and can be done via a config file or via a command line. Basically, only
two options are important:

- `--upstream <url>` may be specified multiple times. Defines a list of upstream servers that the client will 
  try to connect to. The format is `<protocol>[://<host|path>]`. Protocol may be any of the following: `tcp`, 
  `tcp+tls`, `stdin`, `stdin+tls`, `unix`, `unix+tls`, `http`, `https`, `unixgram` or `udp`. Examples:
  - `tcp://127.0.0.1:9995` to connect to a socket server on `localhost` on `9995` 
  - `udp://127.0.0.1:9993` to connect to a UDP server on `localhost` on `9993` 
  - `tcp+tls://127.0.0.1:9995` to connect to a TLS-encrypted socket server on `localhost` on `9995` 
  - `stdin` to connect to server through standard input / output
- `--listen <channel>~<listen-url>[~<forward-url>]` will open a listening socket on the client. 
  - `channel` name must be the same as defined on the server. 
  - `listen-url` is the protocol and the host/path to listen on. Protocol may be `tcp`, `unix` and `stdin` 
  - `foward-url` is the optional direct address of the service. If specified, the client will try to connect
    to this service directly first and, failing that, start going through upstream services.
 
### Examples

#### Server setup

The easiest way to set up a server is with a YAML file. The `examples` directory contains a configuration which
provides different server setups.

#### Client setup

##### Use socketace as a simple telnet client  

```shell script
socketace client -k --upstream tcp+tls://server.example.com:80 --upstream https://server.example.com/proxy --listen smtp~stdin://
```

##### Use socketace to SSH to your server from anywhere

```shell script
ssh localhost -o ProxyCommand='socketace client --upstream http://127.0.0.1:9999/ws/all --listen ssh~stdin://'
```

##### Use socketace to proxy IMAP and SMTP

```shell script
socketace client -e tcp+tls://server.example.com:80 --listen imap~tcp://127.0.0.1:143 --listen imap~tcp://127.0.0.2:587
```

## Caveats

### Connecting to a secure (TLS-enabled) service

If you are trying to proxy a connection to a secure service, you will most likely run into certificate errors. E.g.

If you configure your server with the following:

```yaml
server:
  channels:
    - name: google
      address: tcp://www.google.com:443
  servers:
    # Simple socket proxy. No security.
    - address: tcp://127.0.0.1:9995
```

...and start the server like this:

```shell script
socketace server -c config.yml
```

...and start the client like this:

```shell script
socketace client --upstream tcp://localhost:9995 --listen google~tcp://127.0.0.1:9898
```

Then this will produce a certificate error:

```shell script
curl https://localhost:9898
```

You need to supply the correct host name (either by overriding your hostfile or supplying the host name, if possible).
With `curl`, this is trivial:

```shell script
curl -H "Host: www.google.com" https://localhost:9898
```

## TO-DO

There's still some things to be done. If anybody's willing to pick up issues, pull
requests are welcome:
- add proxying of UDP connections
- add functionality similar to [sslh](https://github.com/yrutschle/sslh) to be able to
  "hide" the proxy and share the port with other services
- add the possibility of proxying connections through DNS, similar to how 
  [iodine](https://github.com/yarrick/iodine) works 
  
## Similar projects

There's [Chisel](https://github.com/jpillora/chisel) which tries to achieve about the same goal, but goes about it 
in a bit of a different way.

## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbokysan%2Fsocketace.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbokysan%2Fsocketace?ref=badge_large)