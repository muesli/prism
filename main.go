package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	// TODO: switch to joy5?
	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
)

var (
	bind        = flag.String("bind", ":1935", "bind address")
	config_file = flag.String("config", "config.json", "config file")
)

type URLConfig struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Bitrate int    `json:"bitrate"`
}

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

func readConfigFromFile(filename string) ([]URLConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var configs []URLConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configs)
	if err != nil {
		return nil, err
	}

	return configs, nil
}

func main() {
	flag.Parse()
	var config []URLConfig
	var err error
	if *config_file != "" {
		if _, err := os.Stat(*config_file); err == nil {
			config, err = readConfigFromFile(*config_file)
			fmt.Println("Read", len(config), "outputs from", *config_file)
		}
	}
	for _, arg := range flag.Args() {
		config = append(config, URLConfig{Enabled: true, URL: arg})
	}

	if len(config) < 1 {
		config = append(config, URLConfig{Enabled: true, URL: "rtmp://localhost/live/test", Width: 1920, Height: 1080, Bitrate: 6000})
		file, err := os.Create(*config_file)
		if err != nil {
			fmt.Println("Error creating config file:", err)
			os.Exit(1)
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(config)

		if err != nil {
			fmt.Println("Error writing config file:", err)
			os.Exit(1)
		}
		fmt.Println("Created default config file", *config_file, "with one example output")
		os.Exit(1)
	}

	fmt.Println("Found", len(config), "outputs")

	fmt.Println("Starting RTMP server...")
	rtmp_config := &rtmp.Config{
		ChunkSize:  128,
		BufferSize: 0,
	}
	server := rtmp.NewServer(rtmp_config)
	server.Addr = *bind

	conns := make([]*RTMPConnection, len(config))
	for i, u := range config {
		// print u object
		fmt.Println(u)
		conns[i] = NewRTMPConnection(u.URL)
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
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
