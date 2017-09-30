package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// InputTable ...
type InputTable struct {
	String string       `json:"name"`
	Alias  string       `json:"alias"`
	Fields []InputField `json:"fields"`
}

// InputField ...
type InputField struct {
	String   string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"null"`
	ReadOnly bool   `json:"read_only"`
	Default  bool   `json:"default"`
	DataType string `json:"data_type"`
	Size     int    `json:"size"`
}

// Table ...
type Table struct {
	Table       string
	TableString string
	Alias       string
	Fields      []Field
}

// Field ...
type Field struct { //nolint: aligncheck
	Name       string
	String     string
	Type       string
	FieldType  string
	ReadOnly   bool
	HasDefault bool
	DataType   dataType
}

var fieldTypes = map[string]string{
	`time`:  `time.Time`,
	`bytes`: `[]byte`,
	`float`: `float64`,
}

var fullUpperList = []string{
	`acl`,
	`api`,
	`ascii`,
	`cpu`,
	`css`,
	`dns`,
	`eof`,
	`guid`,
	`html`,
	`http`,
	`https`,
	`id`,
	`ip`,
	`json`,
	`lhs`,
	`qps`,
	`ram`,
	`rhs`,
	`rpc`,
	`sla`,
	`smtp`,
	`sql`,
	`ssh`,
	`tcp`,
	`tls`,
	`ttl`,
	`udp`,
	`ui`,
	`uid`,
	`uuid`,
	`uri`,
	`url`,
	`utf8`,
	`vm`,
	`xml`,
	`xmpp`,
	`xsrf`,
	`xss`,
}

func getType(t string, null bool) string {
	p := ``
	if null {
		p = `*`
	}
	if v, ok := fieldTypes[t]; ok {
		return p + v
	}
	return p + t
}

type dataType struct {
	Name string
	Size int
	Null bool
}

var dataTypes = map[string]dataType{
	`char`:      {`String`, 0, false},
	`varchar`:   {`String`, 0, false},
	`tinyint`:   {`Int`, 8, false},
	`smallint`:  {`Int`, 16, false},
	`int`:       {`Int`, 32, false},
	`integer`:   {`Int`, 32, false},
	`bigint`:    {`Int`, 64, false},
	`real`:      {`Float`, 32, false},
	`float`:     {`Float`, 64, false},
	`double`:    {`Float`, 64, false},
	`time`:      {`Time`, 0, false},
	`date`:      {`Date`, 0, false},
	`datetime`:  {`Time`, 0, false},
	`timestamp`: {`Time`, 0, false},
	`boolean`:   {`Bool`, 0, false},
	`bool`:      {`Bool`, 0, false},
}

func getDataType(t string, size int, null bool) dataType {
	if v, ok := dataTypes[strings.Split(t, ` `)[0]]; ok {
		if v.Size == 0 {
			v.Size = size
		}
		v.Null = null
		return v
	}
	return dataType{t, size, null}
}

func newField(f InputField) Field {
	return Field{cleanName(f.String), f.String, f.Type, getType(f.Type, f.Nullable), f.ReadOnly, f.Default, getDataType(f.DataType, f.Size, f.Nullable)}
}

func cleanName(s string) string {
	parts := strings.Split(s, `.`)
	parts = strings.Split(parts[len(parts)-1], `_`)
	for k := range parts {
		upper := false
		for _, v := range fullUpperList {
			if v == parts[k] {
				upper = true
				break
			}
		}

		if upper || len(parts[k]) <= 1 {
			parts[k] = strings.ToUpper(parts[k])
			continue
		}

		parts[k] = strings.ToUpper(string(parts[k][0])) + parts[k][1:]
	}
	return strings.Join(parts, ``)
}

var pkg string

func init() {
	log.SetFlags(0)

	flag.StringVar(&pkg, `package`, `model`, `The package name for the output file`)
	flag.Parse()

	if len(flag.Args()) != 2 {
		log.Println(`Usage: qbgenerate [options] input.json output.go`)
		os.Exit(2)
	}
}

func main() {
	in, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(`Failed to open input file. `, err)
	}

	input := []InputTable{}

	err = json.NewDecoder(in).Decode(&input)
	if err != nil {
		log.Fatal(`Failed to parse input file. `, err)
	}

	out, err := os.OpenFile(flag.Arg(1), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(`Failed to open output file. `, err)
	}

	err = generateCode(out, input)
	if err != nil {
		log.Fatal(`Failed to generate code. `, err)
	}

	_ = out.Close()
	err = exec.Command(`goimports`, `-w`, out.Name()).Run()
	if err != nil {
		log.Fatal(`Failed to exectue goimports. `, err)
	}
}

func generateCode(out io.Writer, input []InputTable) error {
	tables := make([]Table, len(input))
	for k, v := range input {
		t := &tables[k]
		t.Table = cleanName(v.String)
		t.Alias = v.Alias
		t.TableString = v.String

		for _, f := range v.Fields {
			t.Fields = append(t.Fields, newField(f))
		}
	}

	t, err := template.New(`code`).Parse(codeTemplate)
	if err != nil {
		return err
	}

	_, _ = io.WriteString(out, `package `+pkg+"\n\n")
	for _, v := range tables {
		if err := t.Execute(out, v); err != nil {
			return err
		}
	}
	return nil
}
