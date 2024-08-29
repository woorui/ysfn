package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/core/ylog"
	"github.com/yomorun/yomo/serverless"
)

type Header struct {
	Tags               []uint32 `json:"tags"`
	FunctionDefinition string   `json:"function_definition"`
}

func serveSFN(name, zipperAddr, credential, tool string, tags []uint32, conn io.ReadWriter) error {
	sfn := yomo.NewStreamFunction(
		name,
		zipperAddr,
		yomo.WithSfnLogger(ylog.NewFromConfig(ylog.Config{Level: "error"})),
		yomo.WithSfnReConnect(),
		yomo.WithSfnCredential(credential),
		yomo.WithSfnAIFunctionDefinitionInJsonSchema(tool),
	)

	var once sync.Once

	sfn.SetObserveDataTags(tags...)
	sfn.SetHandler(func(ctx serverless.Context) {
		var (
			tag  = ctx.Tag()
			data = ctx.Data()
		)
		writeTagData(conn, tag, data)

		once.Do(func() {
			go func() {
				for {
					tag, data, err := readTagData(conn)
					if err == io.EOF {
						return
					}
					_ = ctx.Write(tag, data)
				}
			}()
		})
	})

	if err := sfn.Connect(); err != nil {
		return err
	}

	defer sfn.Close()

	sfn.Wait()

	return nil
}

func Start(zipperAddr, credential string, cmd *exec.Cmd) error {
	socketPath := "sfn.sock"
	defer os.Remove(socketPath)

	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return err
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	errch := make(chan error)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		if err := cmd.Run(); err != nil {
			errch <- err
		}
	}()

	conn, err := listener.Accept()
	if err != nil {
		return err
	}

	headerBytes, err := readHeader(conn)
	if err != nil {
		return err
	}

	header := &Header{}
	err = json.Unmarshal(headerBytes, header)
	if err != nil {
		return err
	}

	fd := &FunctionDefinition{}
	err = json.Unmarshal([]byte(header.FunctionDefinition), fd)
	if err != nil || fd.Name == "" {
		return errors.New("invalid jsonschema, please check your jsonschema file")
	}

	go func() {
		if err := serveSFN(fd.Name, zipperAddr, credential, header.FunctionDefinition, header.Tags, conn); err != nil {
			errch <- err
		}
	}()

	return <-errch
}

func main() {
	// go func() {
	// 	source := yomo.NewSource("hello-1", "localhost:9000")
	// 	err := source.Connect()
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}

	// 	for {
	// 		source.Write(0xe001, []byte(`{"trans_id":"12345","req_id":"67890","result":"Success","arguments":"{}","tool_call_id":"tool123","function_name":"exampleFunction","is_ok":true}`))
	// 		time.Sleep(time.Second)
	// 	}
	// }()

	cmd := exec.Command("npm", "run", "sfn")
	cmd.Dir = "./nodejs"

	// cmd := exec.Command("python", "example/example.py")
	// cmd.Dir = "./python"

	if err := Start("localhost:9000", "", cmd); err != nil {
		log.Fatalln(err)
	}
}

func readTagData(rw io.Reader) (uint32, []byte, error) {
	var tag uint32
	if err := binary.Read(rw, binary.LittleEndian, &tag); err != nil {
		return 0, nil, err
	}

	lengthBytes := make([]byte, 4)
	if err := binary.Read(rw, binary.LittleEndian, &lengthBytes); err != nil {
		return 0, nil, err
	}

	data := make([]byte, binary.LittleEndian.Uint32(lengthBytes))
	if _, err := io.ReadFull(rw, data); err != nil {
		return 0, nil, err
	}

	return tag, data, nil
}

func writeTagData(rw io.ReadWriter, tag uint32, data []byte) error {
	err := binary.Write(rw, binary.LittleEndian, tag)
	if err != nil {
		return err
	}

	err = binary.Write(rw, binary.LittleEndian, uint32(len(data)))
	if err != nil {
		return err
	}

	_, err = rw.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func readHeader(conn io.Reader) ([]byte, error) {
	len := make([]byte, 4)
	if err := binary.Read(conn, binary.LittleEndian, &len); err != nil {
		return nil, err
	}

	title := make([]byte, binary.LittleEndian.Uint32(len))
	if _, err := io.ReadFull(conn, title); err != nil {
		return nil, err
	}

	return title, nil
}

type FunctionDefinition struct {
	Name string `json:"name,omitempty"`
}
