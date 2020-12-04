package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	// TODO: switch to joy5?
	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
)

var (
	bind = flag.String("bind", ":1935", "bind address")
)

type RTMPConnection struct {
	url  string
	conn *rtmp.Conn

	header  []av.CodecData
	packets chan av.Packet
}

func NewRTMPConnection(u string) (*RTMPConnection, error) {
	r := &RTMPConnection{
		url:     u,
		packets: make(chan av.Packet, 2),
	}

	return r, nil // r.Dial()
}

func (r *RTMPConnection) Dial() error {
	c, err := rtmp.Dial(r.url)
	if err != nil {
		return err
	}

	if len(r.header) > 0 {
		err = c.WriteHeader(r.header)
		if err != nil {
			fmt.Println("can't write header:", err)
			return err
		}
	}

	fmt.Println("connection established:", r.url)
	r.conn = c
	return nil
}

func (r *RTMPConnection) Disconnect() error {
	err := r.conn.Close()
	if err != nil {
		return err
	}

	r.conn = nil
	r.header = nil
	close(r.packets)
	r.packets = make(chan av.Packet, 2)

	fmt.Println("connection closed:", r.url)
	return nil
}

func (r *RTMPConnection) WriteHeader(h []av.CodecData) error {
	r.header = h
	if r.conn == nil {
		return r.Dial()
	}

	return r.conn.WriteHeader(h)
}

func (r *RTMPConnection) WritePacket(p av.Packet) {
	if r.conn == nil {
		return
	}

	r.packets <- p
}

func (r *RTMPConnection) Loop() error {
	for p := range r.packets {
		if err := r.conn.WritePacket(p); err != nil {
			r.conn = nil
			fmt.Println(err)

			for {
				time.Sleep(time.Second)

				err := r.Dial()
				if err != nil {
					fmt.Println("can't re-connect:", err)
					continue
				}

				// successful re-connect
				break
			}
		}
	}

	return nil
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Println("usage: prism URL [URL...]")
		os.Exit(1)
	}

	fmt.Println("Starting RTMP server...")
	config := &rtmp.Config{
		ChunkSize:  128,
		BufferSize: 0,
	}
	server := rtmp.NewServer(config)
	server.Addr = *bind

	conns := make([]*RTMPConnection, len(flag.Args()))
	for i, u := range flag.Args() {
		c, err := NewRTMPConnection(u)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		conns[i] = c
	}

	server.HandlePublish = func(conn *rtmp.Conn) {
		fmt.Println("New connection!")
		streams, err := conn.Streams()
		if err != nil {
			fmt.Println("can't retrieve streams:", err)
			os.Exit(1)
		}

		for _, c := range conns {
			if err := c.WriteHeader(streams); err != nil {
				fmt.Println("can't write header:", err)
				// os.Exit(1)
			}
			go c.Loop()
		}

		lastTime := time.Now()
		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				fmt.Println("can't read packet:", err)
				break
			}

			if time.Since(lastTime) > time.Second {
				fmt.Println("Duration:", packet.Time)
				lastTime = time.Now()
			}

			for _, c := range conns {
				c.WritePacket(packet)
			}
		}

		for _, c := range conns {
			err := c.Disconnect()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}

	fmt.Println("Waiting for incoming connection...")
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
