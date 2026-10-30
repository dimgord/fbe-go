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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/dimgord/fbe-go/internal/fb2/export/html"
	"github.com/dimgord/fbe-go/internal/fb2/fsutil"
	"github.com/dimgord/fbe-go/internal/fb2/parser"
	"github.com/dimgord/fbe-go/internal/fb2/thumb"
	"github.com/dimgord/fbe-go/internal/fb2/xsd"
	"github.com/dimgord/fbe-go/internal/fb2/zipfb2"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// errInvalid means the command ran and its answer is "this book is not valid".
var errInvalid = errors.New("invalid")

// run is the whole CLI, returning the exit code: 0 ok, 1 failure, 2 bad usage.
func run(argv []string, stdout, stderr io.Writer) int {
	if len(argv) < 1 {
		usage(stderr)
		return 2
	}
	cmd, args := argv[0], argv[1:]

	var err error
	switch cmd {
	case "validate":
		err = cmdValidate(args, stdout)
	case "thumb":
		err = cmdThumb(args)
	case "unpack":
		err = cmdUnpack(args)
	case "pack":
		err = cmdPack(args)
	case "info":
		err = cmdInfo(args, stdout)
	case "export":
		err = cmdExport(args)
	case "help", "-h", "--help":
		usage(stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", cmd)
		usage(stderr)
		return 2
	}
	if errors.Is(err, errInvalid) {
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `fbe — FictionBook Editor (Go) CLI

Usage:
  fbe validate FILE.fb2
  fbe thumb    FILE.fb2 OUT.jpg
  fbe unpack   FILE.fb2.zip [OUT.fb2]
  fbe pack     FILE.fb2 [OUT.fb2.zip]
  fbe info     FILE.fb2
  fbe export   html FILE.fb2 OUT.html`)
}

func cmdValidate(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fbe validate FILE.fb2")
	}
	in, err := parser.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	errs, err := xsd.ValidateAll(in)
	if err != nil {
		return err
	}
	if len(errs) == 0 {
		fmt.Fprintln(stdout, "VALID")
		return nil
	}
	fmt.Fprintf(stdout, "INVALID: %d error(s)\n", len(errs))
	for _, e := range errs {
		fmt.Fprintf(stdout, "  %d:%d %s\n", e.Line, e.Column, e.Message)
	}
	return errInvalid
}

func cmdThumb(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fbe thumb FILE.fb2 OUT.jpg")
	}
	fb, err := parser.ParseFile(args[0])
	if err != nil {
		return err
	}
	data, _, err := thumb.Extract(fb)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(args[1], data)
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
	return fsutil.WriteAtomic(out, func(w io.Writer) error {
		_, err := io.Copy(w, rc)
		return err
	})
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
	return fsutil.WriteAtomic(out, func(w io.Writer) error { return zipfb2.Pack(w, args[0], in) })
}

func cmdInfo(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: fbe info FILE.fb2")
	}
	fb, err := parser.ParseFile(args[0])
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(fb.Description)
}

func cmdExport(args []string) error {
	if len(args) < 3 || args[0] != "html" {
		return fmt.Errorf("usage: fbe export html FILE.fb2 OUT.html")
	}
	fb, err := parser.ParseFile(args[1])
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(args[2], func(w io.Writer) error { return html.Export(w, fb) })
}
