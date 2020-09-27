package util

type ProtoAddress struct {
	Network string `json:"network" short:"p" long:"network"   description:"Address network" choice:"tcp" choice:"unix" choice:"unixpacket"`
	Address string `json:"address"  short:"a" long:"address"  description:"Address IP and port, e.g. '192.168.8.0:22' or '/var/run/unix.sock'"`
}

type ProtoName struct {
	Name string `json:"name"     short:"n" long:"name"     description:"Unique endpoint name. Must match on the client and the server. E.g. 'ssh'."`
}

func (p *ProtoAddress) String() string {
	return p.Network + "://" + p.Address
}