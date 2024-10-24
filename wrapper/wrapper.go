package wrapper

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/core/ylog"
	"github.com/yomorun/yomo/serverless"
)

type SFNWrapper interface {
	WorkDir() string
	Build() error
	Run() error
}

type Header struct {
	Tags               []uint32 `json:"tags"`
	FunctionDefinition string   `json:"function_definition"`
}

func Run(name, zipperAddr, credential string, wrapper SFNWrapper) error {
	if err := wrapper.Build(); err != nil {
		return err
	}

	sockPath := filepath.Join(wrapper.WorkDir(), "sfn.sock")
	_ = os.Remove(sockPath)

	addr, err := net.ResolveUnixAddr("unix", sockPath)
	if err != nil {
		return err
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	errch := make(chan error)

	go func() {
		if err := wrapper.Run(); err != nil {
			errch <- err
		}
	}()

	conn, err := listener.Accept()
	if err != nil {
		return err
	}

	headerBytes, err := ReadHeader(conn)
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
		if err := serveSFN(name, zipperAddr, credential, header.FunctionDefinition, header.Tags, conn); err != nil {
			errch <- err
		}
	}()

	return <-errch
}

func serveSFN(name, zipperAddr, credential, functionDefinition string, tags []uint32, conn io.ReadWriter) error {
	sfn := yomo.NewStreamFunction(
		name,
		zipperAddr,
		yomo.WithSfnReConnect(),
		yomo.WithSfnCredential(credential),
		yomo.WithAIFunctionJsonDefinition(functionDefinition),
		yomo.WithSfnLogger(ylog.NewFromConfig(ylog.Config{Level: "error"})),
	)

	var once sync.Once

	sfn.SetObserveDataTags(tags...)
	sfn.SetHandler(func(ctx serverless.Context) {
		var (
			tag  = ctx.Tag()
			data = ctx.Data()
		)

		fmt.Println("Input:", string(data))
		WriteTagData(conn, tag, data)

		once.Do(func() {
			go func() {
				for {
					tag, data, err := ReadTagData(conn)
					if err == io.EOF {
						return
					}
					fmt.Println("Output:", string(data))
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

func ReadTagData(r io.Reader) (uint32, []byte, error) {
	var tag uint32
	if err := binary.Read(r, binary.LittleEndian, &tag); err != nil {
		return 0, nil, err
	}

	lengthBytes := make([]byte, 4)
	if err := binary.Read(r, binary.LittleEndian, &lengthBytes); err != nil {
		return 0, nil, err
	}

	data := make([]byte, binary.LittleEndian.Uint32(lengthBytes))
	if _, err := io.ReadFull(r, data); err != nil {
		return 0, nil, err
	}

	return tag, data, nil
}

func WriteTagData(w io.Writer, tag uint32, data []byte) error {
	err := binary.Write(w, binary.LittleEndian, tag)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, uint32(len(data)))
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func ReadHeader(conn io.Reader) ([]byte, error) {
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
