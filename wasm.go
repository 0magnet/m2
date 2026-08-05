// Package main wasm.go
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitfield/script"
	"github.com/briandowns/spinner"
)

func compileWASM() {
	s := spinner.New(spinner.CharSets[14], 25*time.Millisecond)
	s.Suffix = " Compiling wasm..."
	for _, wasmSRC := range f.WasmSRC {
		ascend := strings.Repeat("../", len(strings.Split(wasmSRC, "/")))
		outputFile := strings.TrimSuffix(filepath.Base(wasmSRC), filepath.Ext(wasmSRC)) + ".wasm"
		compilecmd := fmt.Sprintf("bash -c 'cd %s || exit 1 ; time GOOS=js GOARCH=wasm %s -o %s %s -ldflags=\"-s -w\" %s && cd %s && du %s'", wasmSRC, f.Gobuild, ascend+outputFile, ldflags(wasmSRC), ".", ascend, outputFile)
		log.Println("Compiling wasm with:")
		log.Println(compilecmd)
		s.Start()
		_, err := script.Exec(compilecmd).Stdout()
		if err != nil {
			log.Fatal(err)
		}
		s.Stop()
		log.Println("Compiled wasm!")
	}
	if !f.UseTinygo {
		return
	}
	for _, wasmSRC := range f.WasmSRC {
		ascend := strings.Repeat("../", len(strings.Split(wasmSRC, "/")))
		outputFile := strings.TrimSuffix(filepath.Base(wasmSRC), filepath.Ext(wasmSRC)) + "-tiny.wasm"
		compilecmd := fmt.Sprintf("bash -c 'cd %s || exit 1 ; time GOOS=js GOARCH=wasm %s -o %s %s %s && cd %s && du %s'", wasmSRC, f.Tinygobuild, ascend+outputFile, ldflags(wasmSRC), ".", ascend, outputFile)
		log.Println("compiling wasm with:")
		log.Println(compilecmd)
		s.Start()
		_, err := script.Exec(compilecmd).Stdout()
		if err != nil {
			log.Fatal(err)
		}
		s.Stop()
		log.Println("Compiled wasm!")
	}
}

func ldflags(s string) (ss string) {
	checkFiles, err := script.FindFiles(s).Slice()
	if err != nil {
		log.Fatal(err)
	}
	if f.LDFlagsX != "" {
		for _, s := range checkFiles {
			res, err := script.File(s).Match(strings.Split(f.LDFlagsX, "=")[0]).String()
			if err != nil {
				log.Fatal(err)
			}
			if res != "" {
				ss += fmt.Sprintf(` -X 'main.%s' `, f.LDFlagsX)
				break
			}
		}
	}
	for _, s := range checkFiles {
		res, err := script.File(s).Match("wasmName").String()
		if err != nil {
			log.Fatal(err)
		}
		if res != "" {
			ss += fmt.Sprintf(` -X 'main.wasmName=%s' `, strings.TrimSuffix(filepath.Base(s), filepath.Ext(s))+".wasm")
			break
		}
	}
	if ss != "" {
		ss = `-ldflags="` + ss + `"`
	}
	return ss
}
