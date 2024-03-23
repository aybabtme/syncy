package main

import (
	"encoding/json"
	"io"

	"github.com/fatih/color"
	"github.com/kr/pretty"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type printer interface {
	Emit(any) int
	Error(string)
}

var _ printer = (*jsonPrinter)(nil)

type jsonPrinter struct {
	out io.Writer
}

func newJSONPrinter(out io.Writer) *jsonPrinter {

	return &jsonPrinter{out: out}
}

func (pp *jsonPrinter) Emit(v any) int {
	return pp.encode(v)
}

func (pp *jsonPrinter) Error(msg string) {
	type error struct {
		Error string `json:"error"`
	}
	pp.encode(error{Error: msg})
}

func (pp *jsonPrinter) encode(v any) int {
	var (
		out []byte
		err error
	)
	if pv, ok := v.(proto.Message); ok {
		out, err = protojson.Marshal(pv)
	} else {
		out, err = json.Marshal(v)
	}
	if err != nil {
		panic(err)
	}
	if n, err := pp.out.Write(out); err != nil {
		panic(err)
	} else {
		return n
	}
}

var _ printer = (*textPrinter)(nil)

type textPrinter struct {
	out io.Writer
}

func newTextPrinter(out io.Writer) *textPrinter {
	return &textPrinter{out: out}
}

func (pp *textPrinter) Emit(v any) int {
	return pp.print(pretty.Sprint(v) + "\n")
}

func (pp *textPrinter) Error(msg string) {
	pp.print(color.RedString("error: %v\n", msg))
}

func (pp *textPrinter) print(v string) int {
	if len(v) == 0 {
		return 0
	}
	n, err := pp.out.Write([]byte(v))
	if err != nil {
		panic(err)
	}
	if v[len(v)-1] != '\n' {
		m, err := pp.out.Write([]byte{'\n'})
		if err != nil {
			panic(err)
		}
		return n + m
	}
	return n
}

var _ printer = (*jsonPrinter)(nil)

type protoPrinter struct {
	out io.Writer
}

func newProtoPrinter(out io.Writer) *protoPrinter {
	return &protoPrinter{out: out}
}

func (pp *protoPrinter) Emit(v any) int {
	return pp.encode(v)
}

func (pp *protoPrinter) Error(msg string) {
	type error struct {
		Error string `json:"error"`
	}
	pp.encode(error{Error: msg})
}

func (pp *protoPrinter) encode(v any) int {
	var (
		out []byte
		err error
	)
	if pv, ok := v.(proto.Message); ok {
		out, err = proto.Marshal(pv)
	} else {
		out, err = json.Marshal(v)
	}
	if err != nil {
		panic(err)
	}
	if n, err := pp.out.Write(out); err != nil {
		panic(err)
	} else {
		return n
	}
}
