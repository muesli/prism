package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	// TODO: switch to joy5?
	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
)

var (
	bind      = flag.String("bind", ":1935", "bind address")
	streamKey = flag.String("key", "", "stream key")
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
	r.packets = make(chan av.Packet, 2)
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

	close(r.packets)
	r.reset()

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

	if *streamKey == "" {
		uuid, err := newUUID()
		if err != nil {
			fmt.Println("Can't generate rtmp key:", err)
			os.Exit(1)
		}

		*streamKey = uuid
	}

	fmt.Println("RTMP stream key set to: ", *streamKey)

	fmt.Println("Starting RTMP server...")
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
		if !strings.HasSuffix(conn.URL.Path, *streamKey) {
			fmt.Println("Connection attempt made using incorrect key: ", conn.URL.Path)
			conn.Close()
			return
		}

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

// newUUID generates a random UUID according to the RFC 4122, https://play.golang.org/p/4FkNSiUDMg
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)

	if n != len(uuid) || err != nil {
		return "", err
	}

	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
