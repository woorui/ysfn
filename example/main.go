package main

import (
	"log"
	"os"

	_ "embed"

	"github.com/joho/godotenv"
	"github.com/woorui/ysfn/wrapper"
	"github.com/woorui/ysfn/wrapper/nodejs"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	entryFile := "app.ts"

	var (
		functionName = os.Getenv("YOMO_SFN_NAME")
		zipperAddr   = os.Getenv("YOMO_SFN_ZIPPER")
		credential   = os.Getenv("YOMO_SFN_CREDENTIAL")
	)

	nw, err := nodejs.NewWrapper(functionName, entryFile)
	if err != nil {
		log.Fatalln(err)
	}

	wrapper.BuildAndRun(functionName, zipperAddr, credential, nw)
}
