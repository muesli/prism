package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	// TODO: switch to joy5?
	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
)

var (
	bind         = flag.String("bind", ":1935", "bind address")
	outputs_file = flag.String("outputs-file", "outputs.txt", "output URLs file")
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

func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

func main() {
	flag.Parse()
	var urls []string
	var err1 error
	if *outputs_file != "" {
		if _, err := os.Stat("outputs.txt"); err == nil {
			urls, err1 = readURLsFromFile(*outputs_file)
			fmt.Println("Read", len(urls), "output URLs from", *outputs_file)
		}
	}
	// if flag.args is over 0, add all args to urls
	if len(flag.Args()) > 0 {
		urls = append(urls, flag.Args()...)
	}

	if err1 != nil {
		fmt.Println("Error reading output URLs from file:", err1)
		os.Exit(1)
	}

	fmt.Println("Found", len(urls), "output URLs")

	fmt.Println("Starting RTMP server...")
	config := &rtmp.Config{
		ChunkSize:  128,
		BufferSize: 0,
	}
	server := rtmp.NewServer(config)
	server.Addr = *bind

	conns := make([]*RTMPConnection, len(urls))
	for i, u := range urls {
		fmt.Println(i, ":", u)
		conns[i] = NewRTMPConnection(u)
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
