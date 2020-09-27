package server

import (
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/multiformats/go-multistream"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
)

func addMuxHandler(mux *multistream.MultistreamMuxer, upstream *Channel) {
	handler := func(protocol string, downstreamConnection io.ReadWriteCloser) error {
		switch upstream.Network {
		case "udp", "udp4", "udp6", "unixgram":
			return errors.Errorf("Packet connections (%v) are not yet supported", upstream.Network)
		}

		log.Debugf("Opening connection to upstream: %v", upstream)
		upstreamConnection, err := upstream.OpenConnection()
		if err != nil {
			return err
		}

		return util.PipeData(downstreamConnection, upstreamConnection)
	}
	mux.AddHandler("/"+upstream.Name, handler)

}

func MultiplexToUpstream(multiplexChannel io.ReadWriteCloser, upstreams []*Channel) error {
	mux := multistream.NewMultistreamMuxer()
	for _, u := range upstreams {
		addMuxHandler(mux, u)
	}

	defer func() {
		if err := multiplexChannel.Close(); err != nil {
			log.WithError(err).Warnf("Failed closing the multiplex channel connection: %+v", err)
		}
	}()

	if err := mux.Handle(multiplexChannel); err != nil {
		err = errors.Wrapf(err, "Could not handle multiplex channel: %+v", err)
		return err
	}
	return nil
}

