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
	if len(os.Args) < 2 {
		log.Fatalln("usage: yomorun <entry file>")
	}
	entryFile := os.Args[1]

	var (
		functionName = os.Getenv("YOMO_SFN_NAME")
		zipperAddr   = os.Getenv("YOMO_SFN_ZIPPER")
		credential   = os.Getenv("YOMO_SFN_CREDENTIAL")
	)

	nw, err := nodejs.NewWrapper(functionName, entryFile)
	if err != nil {
		log.Fatalln(err)
	}

	if err := wrapper.Run(functionName, zipperAddr, credential, nw); err != nil {
		log.Fatalln(err)
	}
}
