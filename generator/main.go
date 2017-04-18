package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
)

// InputTable ...
type InputTable struct {
	String string       `json:"name"`
	Fields []InputField `json:"fields"`
}

// InputField ...
type InputField struct {
	String   string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"null"`
	ReadOnly bool   `json:"read_only"`
}

// Table ...
type Table struct {
	Table       string
	TableString string
	Fields      []Field
}

// Field ...
type Field struct {
	Name      string
	String    string
	Type      string
	FieldType string
	ReadOnly  bool
	Default   string
}

var fieldTypes = map[string]string{
	`string`:  `StringField`,
	`int`:     `IntField`,
	`int64`:   `Int64Field`,
	`int32`:   `Int32Field`,
	`float64`: `Float64Field`,
	`float32`: `Float32Field`,
	`float`:   `FloatField`,
	`bool`:    `BoolField`,
	`time`:    `TimeField`,
}

var defaultTypes = map[string]string{
	`string`:  "``",
	`int`:     `0`,
	`int64`:   `0`,
	`int32`:   `0`,
	`float64`: `0`,
	`float32`: `0`,
	`bool`:    `false`,
	`time`:    `time.Time{}`,
}

func newField(name string, t string, nullable bool, readOnly bool) Field {
	var prefix string
	def := defaultTypes[t]
	if nullable {
		prefix = `Null`
		def = `nil`
	}
	return Field{strings.Replace(name, `_`, ``, -1), name, t, prefix + fieldTypes[t], readOnly, def}
}

func cleanName(s string) string {
	parts := strings.Split(s, `.`)
	parts = strings.Split(parts[len(parts)-1], `_`)
	for k := range parts {
		parts[k] = strings.ToUpper(string(parts[k][0])) + parts[k][1:]
	}
	return strings.Join(parts, ``)
}

func main() {
	p := flag.String(`package`, `model`, `The package name for the output file`)
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Println(`Usage: qbgenerate [options] input.json output.go`)
		os.Exit(2)
	}

	in, err := os.Open(args[0])
	if err != nil {
		fmt.Println(`Failed to open input file`)
		os.Exit(2)
	}
	defer in.Close()

	input := []InputTable{}

	err = json.NewDecoder(in).Decode(&input)
	if err != nil {
		fmt.Println(`Failed to parse input file`)
		os.Exit(2)
	}

	in.Close()

	out, err := os.OpenFile(args[1], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println(`Failed to open output file`)
		os.Exit(2)
	}
	defer out.Close()

	tables := []Table{}

	for _, v := range input {
		t := Table{
			Table:       cleanName(v.String),
			TableString: v.String,
		}

		for _, f := range v.Fields {
			t.Fields = append(t.Fields, newField(f.String, f.Type, f.Nullable, f.ReadOnly))
		}

		tables = append(tables, t)
	}

	fmt.Fprint(out, `package `, *p, "\n\n")

	t, err := template.New(`code`).Parse(codeTemplate)
	if err != nil {
		panic(err)
	}

	for _, v := range tables {
		err = t.Execute(out, v)
		if err != nil {
			panic(err)
		}
	}

	out.Close()

	exec.Command(`goimports`, `-w`, out.Name()).Run()
}