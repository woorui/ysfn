package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	_ "embed"

	"github.com/joho/godotenv"
	"github.com/woorui/ysfn/wrapper"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	sockPath := filepath.Join("", "debug.sock")
	_ = os.Remove(sockPath)

	addr, err := net.ResolveUnixAddr("unix", sockPath)
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go func() {
			headerBytes, err := wrapper.ReadHeader(conn)
			if err != nil {
				panic(err)
			}

			header := &wrapper.Header{}
			err = json.Unmarshal(headerBytes, header)
			if err != nil {
				panic(err)
			}

			fd := &wrapper.FunctionDefinition{}
			err = json.Unmarshal([]byte(header.FunctionDefinition), fd)
			if err != nil || fd.Name == "" {
				fmt.Println("invalid function definition")
			}
			fmt.Println(header.Tags, fd.Name)

			go func() {
				for {
					data := `{"tid":"kSF2ae3T_roMDGXLQAybkw7KHFJkfYMp","req_id":"jHGiEumCkJ-1WZHp","arguments":"{\"location\": \"江西\",\"unit\":\"celsius\"}","tool_call_id":"call_Q1rjCglaW4L3JRq6bkLjHsVJ","function_name":"get_weather","is_ok":false}`
					if err := wrapper.WriteTagData(conn, 0x33, []byte(data)); err != nil {
						return
					}
					fmt.Println("Send:", data)
					time.Sleep(2 * time.Second)
				}
			}()

			for {
				tag, data, err := wrapper.ReadTagData(conn)
				if err == io.EOF {
					break
				}
				if err != nil {
					panic(err)
				}

				fmt.Println("Receive:", tag, string(data))
			}
		}()
	}

}
