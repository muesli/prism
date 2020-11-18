package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	// TODO: switch to joy5?
	rtmp "github.com/notedit/rtmp-lib"
)

var (
	bind = flag.String("bind", ":1935", "bind address")
)

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Println("usage: prism URL [URL...]")
		os.Exit(1)
	}

	config := &rtmp.Config{
		ChunkSize:  128,
		BufferSize: 0,
	}
	server := rtmp.NewServer(config)
	server.Addr = *bind

	conns := make([]*rtmp.Conn, len(flag.Args()))
	for i, u := range flag.Args() {
		c, err := rtmp.Dial(u)
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
			fmt.Println(err)
			os.Exit(1)
		}

		for _, c := range conns {
			if err := c.WriteHeader(streams); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		lastTime := time.Now()
		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				fmt.Println(err)
				break
			}

			if time.Since(lastTime) > time.Second {
				fmt.Println("Duration:", packet.Time)
				lastTime = time.Now()
			}

			for _, c := range conns {
				if err := c.WritePacket(packet); err != nil {
					// TODO: re-establish connection
					// can we just cache the streams?
					// buffer and continue where we left off?
					fmt.Println(err)
					break
				}
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
