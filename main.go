package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
	"github.com/yomorun/yomo"
	"github.com/yomorun/yomo/core/ylog"
	"github.com/yomorun/yomo/serverless"
)

type Header struct {
	Tags []uint32 `json:"tags"`
}

func serveSFN(name, zipperAddr, credential, functionDefinition string, tags []uint32, conn io.ReadWriter) error {
	sfn := yomo.NewStreamFunction(
		name,
		zipperAddr,
		yomo.WithSfnLogger(ylog.NewFromConfig(ylog.Config{Level: "error"})),
		yomo.WithSfnReConnect(),
		yomo.WithSfnCredential(credential),
		yomo.WithAIFunctionJsonDefinition(functionDefinition),
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

func Run(name, zipperAddr, credential string, wrapper SfnWrapper) error {
	if err := wrapper.Build(); err != nil {
		return err
	}

	sockPath := wrapper.WorkDir() + "/sfn.sock"

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

	headerBytes, err := readHeader(conn)
	if err != nil {
		return err
	}

	header := &Header{}
	err = json.Unmarshal(headerBytes, header)
	if err != nil {
		return err
	}

	fdString, err := wrapper.GetFunctionDefinition()
	if err != nil {
		return fmt.Errorf("cannot load function definition: %w", err)
	}

	fd := &FunctionDefinition{}
	err = json.Unmarshal([]byte(fdString), fd)
	if err != nil || fd.Name == "" {
		return errors.New("invalid jsonschema, please check your jsonschema file")
	}

	go func() {
		if err := serveSFN(name, zipperAddr, credential, fdString, header.Tags, conn); err != nil {
			errch <- err
		}
	}()

	return <-errch
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("usage: yomorun <entry file>")
	}
	entryFile := os.Args[1]

	wrapper, err := NewNodejsWrapper(entryFile)
	if err != nil {
		log.Fatalln(err)
	}

	_ = godotenv.Load()

	if err := Run(
		os.Getenv("YOMO_SFN_NAME"),
		os.Getenv("YOMO_SFN_ZIPPER"),
		os.Getenv("YOMO_SFN_CREDENTIAL"),
		wrapper,
	); err != nil {
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

type SfnWrapper interface {
	WorkDir() string
	Build() error
	GetFunctionDefinition() (string, error)
	Run() error
}

type NodejsWrapper struct {
	workDir     string
	entryTSFile string
	entryJSFile string
}

func NewNodejsWrapper(entryTSFile string) (SfnWrapper, error) {
	ext := filepath.Ext(entryTSFile)
	if ext != ".ts" {
		return nil, fmt.Errorf("only support typescript, got: %s", entryTSFile)
	}
	workdir := filepath.Dir(entryTSFile)

	entryJSFile := entryTSFile[:len(entryTSFile)-len(ext)] + ".js"

	w := &NodejsWrapper{
		workDir:     workdir,
		entryTSFile: entryTSFile,
		entryJSFile: entryJSFile,
	}

	return w, nil
}

func (w *NodejsWrapper) WorkDir() string {
	return w.workDir
}

func (w *NodejsWrapper) Build() error {
	cmd := exec.Command("npm", "install")
	cmd.Dir = w.workDir

	if err := cmd.Run(); err != nil {
		return err
	}

	cmd2 := exec.Command("tsc", w.entryTSFile)
	cmd2.Dir = w.workDir

	return cmd2.Run()
}

func (w *NodejsWrapper) GetFunctionDefinition() (string, error) {
	data, err := os.ReadFile(filepath.Join(w.workDir, "jsonschema.json"))
	return string(data), err
}

func (w *NodejsWrapper) Run() error {
	cmd := exec.Command("node", w.entryJSFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
