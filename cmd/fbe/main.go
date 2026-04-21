// Command fbe is a CLI companion to the desktop app.
//
// It covers batch and scripting workflows that do not need a GUI:
//
//   fbe validate FILE.fb2          — schema-validate against FictionBook.xsd
//   fbe thumb    FILE.fb2 OUT.jpg  — extract coverpage
//   fbe pack     FILE.fb2          — produce FILE.fb2.zip
//   fbe unpack   FILE.fb2.zip      — extract to FILE.fb2
//   fbe info     FILE.fb2          — print metadata as JSON
//   fbe export   html FILE.fb2 OUT.html
//
// Replaces FBV.exe (standalone C++ validator in FBV/) and centralizes the core
// library for scripting / library-management use cases.
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dimgord/fbe-go/internal/fb2/export/html"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/thumb"
	"github.com/dimgord/fbe-go/internal/fb2/writer"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
	"github.com/dimgord/fbe-go/internal/fb2/zipfb2"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]

	var err error
	switch cmd {
	case "validate":
		err = cmdValidate(args)
	case "thumb":
		err = cmdThumb(args)
	case "unpack":
		err = cmdUnpack(args)
	case "pack":
		err = cmdPack(args)
	case "info":
		err = cmdInfo(args)
	case "export":
		err = cmdExport(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `fbe — FictionBook Editor (Go) CLI

Usage:
  fbe validate FILE.fb2
  fbe thumb    FILE.fb2 OUT.jpg
  fbe unpack   FILE.fb2.zip [OUT.fb2]
  fbe pack     FILE.fb2 [OUT.fb2.zip]
  fbe info     FILE.fb2
  fbe export   html FILE.fb2 OUT.html`)
}

func cmdValidate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fbe validate FILE.fb2")
	}
	in, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	errs, err := xsd.Validate(in)
	if err != nil {
		return err
	}
	if len(errs) == 0 {
		fmt.Println("VALID")
		return nil
	}
	fmt.Printf("INVALID: %d error(s)\n", len(errs))
	for _, e := range errs {
		fmt.Printf("  %s\n", e.Message)
	}
	os.Exit(1)
	return nil
}

func cmdThumb(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fbe thumb FILE.fb2 OUT.jpg")
	}
	in, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	fb, err := parser.Parse(in)
	if err != nil {
		return err
	}
	data, _, err := thumb.Extract(fb)
	if err != nil {
		return err
	}
	return os.WriteFile(args[1], data, 0o644)
}

func cmdUnpack(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fbe unpack FILE.fb2.zip [OUT.fb2]")
	}
	zr, err := zip.OpenReader(args[0])
	if err != nil {
		return err
	}
	defer zr.Close()

	rc, err := zipfb2.Unpack(&zr.Reader)
	if err != nil {
		return err
	}
	defer rc.Close()

	out := "out.fb2"
	if len(args) >= 2 {
		out = args[1]
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = copyAll(f, rc)
	return err
}

func cmdPack(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fbe pack FILE.fb2 [OUT.fb2.zip]")
	}
	in, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	out := args[0] + ".zip"
	if len(args) >= 2 {
		out = args[1]
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	return zipfb2.Pack(f, args[0], in)
}

func cmdInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fbe info FILE.fb2")
	}
	in, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()
	fb, err := parser.Parse(in)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(fb.Description)
}

func cmdExport(args []string) error {
	if len(args) < 3 || args[0] != "html" {
		return fmt.Errorf("usage: fbe export html FILE.fb2 OUT.html")
	}
	in, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer in.Close()
	fb, err := parser.Parse(in)
	if err != nil {
		return err
	}
	out, err := os.Create(args[2])
	if err != nil {
		return err
	}
	defer out.Close()
	_ = writer.Write // keep writer imported for later
	return html.Export(out, fb)
}

// copyAll avoids importing io just for io.Copy.
func copyAll(dst interface{ Write([]byte) (int, error) }, src interface{ Read([]byte) (int, error) }) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total, werr
			}
			total += int64(n)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return total, nil
			}
			return total, err
		}
	}
}
