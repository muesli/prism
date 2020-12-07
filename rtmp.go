package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	// TODO: switch to joy5?
	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
)

type RTMPConnection struct {
	url  string
	conn *rtmp.Conn

	header  []av.CodecData
	packets chan av.Packet
}

func NewRTMPConnection(u string) *RTMPConnection {
	r := &RTMPConnection{
		url: u,
	}
	r.reset()

	return r
}

func (r *RTMPConnection) reset() {
	r.packets = make(chan av.Packet)
	r.conn = nil
	r.header = nil
}

func (r *RTMPConnection) Dial() error {
	c, err := rtmp.Dial(r.url)
	if err != nil {
		return err
	}

	if len(r.header) > 0 {
		err = c.WriteHeader(r.header)
		if err != nil {
			log.Log(1, fmt.Sprintf("Failed writing header to %s: %s", shortenURL(r.url), err))
			return err
		}
	}

	log.Log(3, "Connection established to: "+shortenURL(r.url))
	r.conn = c
	return nil
}

func (r *RTMPConnection) Disconnect() error {
	if r.conn != nil {
		err := r.conn.Close()
		if err != nil {
			return err
		}

		log.Log(3, "Connection closed: "+shortenURL(r.url))
	}

	close(r.packets)
	r.reset()

	return nil
}

func (r *RTMPConnection) WriteHeader(h []av.CodecData) {
	r.header = h
}

func (r *RTMPConnection) WritePacket(p av.Packet) {
	select {
	case r.packets <- p:
	default:
		// non-blocking send
	}
}

func (r *RTMPConnection) Loop() {
	for p := range r.packets {
		if r.conn == nil {
			if err := r.Dial(); err != nil {
				log.Log(1, fmt.Sprintf("Connection to %s failed: %s", shortenURL(r.url), err))
				time.Sleep(time.Second)
				continue
			}
		}

		if err := r.conn.WritePacket(p); err != nil {
			r.conn = nil
			log.Log(1, fmt.Sprintf("Sending stream to %s failed: %s", shortenURL(r.url), err))

			for {
				time.Sleep(time.Second)

				err := r.Dial()
				if err != nil {
					log.Log(1, fmt.Sprintf("Connection to %s failed: %s", shortenURL(r.url), err))
					time.Sleep(time.Second)
					continue
				}

				// successful re-connect
				break
			}
		}
	}
}

func shortenURL(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	return u.Host
}

func setupRTMP() {
	config := &rtmp.Config{
		ChunkSize:  128,
		BufferSize: 0,
	}
	server := rtmp.NewServer(config)
	server.Addr = *bind

	conns := make([]*RTMPConnection, len(flag.Args()))
	for i, u := range flag.Args() {
		conns[i] = NewRTMPConnection(u)
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		log.Log(2, fmt.Sprintf("Incoming RTMP connection from: %s", conn.URL.Host))

		streams, err := conn.Streams()
		if err != nil {
			log.Log(1, fmt.Sprintf("Can't retrieve stream headers: %s", err))
			return
		}

		for _, c := range conns {
			c.WriteHeader(streams)
			go c.Loop()
		}

		lastTime := time.Now()
		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				if err == io.EOF {
					log.Log(2, "Incoming connection closed: "+err.Error())
				} else {
					log.Log(1, "Incoming connection aborted: "+err.Error())
				}
				break
			}

			if time.Since(lastTime) > time.Second {
				log.Replace(4, "Stream duration: "+packet.Time.String())
				lastTime = time.Now()
			}

			for _, c := range conns {
				c.WritePacket(packet)
			}
		}

		for _, c := range conns {
			err := c.Disconnect()
			if err != nil {
				log.Log(1, fmt.Sprintf("Error disconnecting from %s: %s", shortenURL(c.url), err))
				os.Exit(1)
			}
		}

		log.Log(2, "Waiting for incoming connection...")
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Log(1, fmt.Sprintf("Failed starting RTMP server: %s", err))
			os.Exit(1)
		}
	}()

	log.Log(2, "Waiting for incoming connection...")
}
