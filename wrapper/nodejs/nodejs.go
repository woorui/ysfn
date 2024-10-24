package nodejs

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"

	_ "embed"

	"github.com/woorui/ysfn/wrapper"
)

//go:embed templates/wrapper_ts.tmpl
var WrapperTSTmpl string

var (
	wrapperTS = ".wrapper.ts"
	wrapperJS = ".wrapper.js"
)

type NodejsWrapper struct {
	functionName string
	workDir      string // eg. src/
	entryTSFile  string // eg. src/app.ts
	entryJSFile  string // eg. src/app.js
	fileName     string // eg. src/app
}

func NewWrapper(functionName, entryTSFile string) (wrapper.SFNWrapper, error) {
	ext := filepath.Ext(entryTSFile)
	if ext != ".ts" {
		return nil, fmt.Errorf("only support typescript, got: %s", entryTSFile)
	}
	workdir := filepath.Dir(entryTSFile)

	entryJSFile := entryTSFile[:len(entryTSFile)-len(ext)] + ".js"

	fileName := entryTSFile[:len(entryTSFile)-len(filepath.Ext(entryTSFile))]

	w := &NodejsWrapper{
		functionName: functionName,
		workDir:      workdir,
		entryTSFile:  entryTSFile,
		entryJSFile:  entryJSFile,
		fileName:     fileName,
	}

	return w, nil
}

func (w *NodejsWrapper) WorkDir() string {
	return w.workDir
}

func (w *NodejsWrapper) Build() error {
	// 1. generate .wrapper.ts file
	dstPath := filepath.Join(w.workDir, wrapperTS)
	_ = os.Remove(dstPath)

	if err := w.genWrapperTS(w.functionName, dstPath); err != nil {
		return err
	}

	// 2. install dependencies
	cmd := exec.Command("pnpm", "install")
	cmd.Dir = w.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// 3. compile .wrapper.ts file to .wrapper.js
	cmd2 := exec.Command("tsc", wrapperTS)
	cmd2.Dir = w.workDir
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	if err := cmd2.Run(); err != nil {
		return err
	}

	fmt.Println("Build (Install Dependencies & Compile) success")

	return nil
}

func (w *NodejsWrapper) Run() error {
	cmd := exec.Command("node", wrapperJS)
	cmd.Dir = w.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (w *NodejsWrapper) genWrapperTS(functionName, dstPath string) error {
	data := struct {
		WorkDir      string
		FunctionName string
		FileName     string
		FilePath     string
	}{
		WorkDir:      w.workDir,
		FunctionName: functionName,
		FileName:     w.fileName,
		FilePath:     w.entryTSFile,
	}

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	t := template.Must(template.New("wrapper").Parse(WrapperTSTmpl))
	if err := t.Execute(dst, data); err != nil {
		return err
	}

	return nil
}
