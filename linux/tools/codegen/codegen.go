package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

var (
	out  = flag.String("out", "", "help message for flagname")
	tmpl = flag.String("tmpl", "", "help message for flagname")
)

var cnt = 0

var funcMap = template.FuncMap{
	"esc": func(s string) string {
		s = strings.Replace(s, " ", "", -1)
		s = strings.Replace(s, "/", "", -1)
		s = strings.Replace(s, "_", "", -1)
		return s
	},
	"reset": func() string {
		cnt = 0
		return ""
	},
	"roy": func(n, c, k, v string) string {
		var s string
		switch v {
		case "uint8":
			s += fmt.Sprintf("// %s ...\n", k)
			s += fmt.Sprintf("func (r %s) %s () %s { return r[%d]}\n", n, k, v, cnt)
			s += fmt.Sprintf("// Set%s ...\n", k)
			if k == "AttributeOpcode" {
				s += fmt.Sprintf("func (r %s) Set%s () { r[%d] = %s}", n, k, cnt, c)
			} else {
				s += fmt.Sprintf("func (r %s) Set%s (v %s) { r[%d] = v}", n, k, v, cnt)
			}
			cnt++
		case "uint16":
			s += fmt.Sprintf("// %s ...\n", k)
			s += fmt.Sprintf("func (r %s) %s () %s { return binary.LittleEndian.Uint16(r[%d:])}\n", n, k, v, cnt)
			s += fmt.Sprintf("// Set%s ...\n", k)
			s += fmt.Sprintf("func (r %s) Set%s (v %s) { binary.LittleEndian.PutUint16(r[%d:], v)}", n, k, v, cnt)
			cnt += 2
		case "uint64":
			s += fmt.Sprintf("// %s ...\n", k)
			s += fmt.Sprintf("func (r %s) %s () %s { return binary.LittleEndian.Uint64(r[%d:])}\n", n, k, v, cnt)
			s += fmt.Sprintf("// Set%s ...\n", k)
			s += fmt.Sprintf("func (r %s) Set%s (v %s) { binary.LittleEndian.PutUint64(r[%d:], v)}", n, k, v, cnt)
			cnt += 8
		case "[]byte":
			s += fmt.Sprintf("// %s ...\n", k)
			s += fmt.Sprintf("func (r %s) %s () %s { return r[%d:]}\n", n, k, v, cnt)
			s += fmt.Sprintf("// Set%s ...\n", k)
			s += fmt.Sprintf("func (r %s) Set%s (v %s) { copy(r[%d:], v)}", n, k, v, cnt)
		case "[6]byte":
			s += fmt.Sprintf("// %s ...\n", k)
			s += fmt.Sprintf(`func (r %s) %s () %s {
				 b:=[6]byte{}
				 copy(b[:], r[%d:])
				 return b
				 }
				 `, n, k, v, cnt)
			s += fmt.Sprintf("// Set%s ...\n", k)
			s += fmt.Sprintf(`func (r %s) Set%s (v %s) { copy(r[%d:%d+6], v[:]) }`, n, k, v, cnt, cnt)
			cnt += 6
		case "[12]byte":
			s += fmt.Sprintf("// %s ...\n", k)
			s += fmt.Sprintf(`func (r %s) %s () %s {
				 b:=[12]byte{}
				 copy(b[:], r[%d:])
				 return b
				 }
				 `, n, k, v, cnt)
			s += fmt.Sprintf("// Set%s ...\n", k)
			s += fmt.Sprintf(`func (r %s) Set%s (v %s) { copy(r[%d:%d+12], v[:]) }`, n, k, v, cnt, cnt)
			cnt += 12
		default:
			s += fmt.Sprintf("XXX: %s, %s, %s", n, k, v)
		}
		return s
	},
	"getter": func(n, c, k, v string) string {
		var s string
		switch v {
		case "uint8":
			s = fmt.Sprintf("func (r %s) %s () %s { return r[%d]}\n", n, k, v, cnt)
			cnt++
		case "uint16":
			s = fmt.Sprintf("func (r %s) %s () %s { return binary.LittleEndian.Uint16(r[%d:])}\n", n, k, v, cnt)
			cnt += 2
		case "uint64":
			s = fmt.Sprintf("func (r %s) %s () %s { return binary.LittleEndian.Uint64(r[%d:])}\n", n, k, v, cnt)
			cnt += 8
		case "[]byte":
			s = fmt.Sprintf("func (r %s) %s () %s { return r[%d:]}\n", n, k, v, cnt)
		case "[6]byte":
			s = fmt.Sprintf(`func (r %s) %s () %s {
				 b:=[6]byte{}
				 copy(b[:], r[%d:])
				 return b
				 }
				 `, n, k, v, cnt)
			cnt += 6
		case "[12]byte":
			s = fmt.Sprintf(`func (r %s) %s () %s {
				 b:=[12]byte{}
				 copy(b[:], r[%d:])
				 return b
				 }
				 `, n, k, v, cnt)
			cnt += 12
		default:
			s += fmt.Sprintf("XXX: %s, %s, %s", n, k, v)
		}
		return s
	},
}

func input(s string) []byte {
	fi, err := os.Open(s)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(fi)
	if err != nil {
		panic(err)
	}
	return b
}

func output(s string) io.WriteCloser {
	w, err := os.Create(s)
	if err != nil {
		panic(err)
	}
	return w
}

type field map[string]string

type cmd struct {
	Name   string   // Command Name
	Spec   string   // Specification
	OGF    string   // OoCode Group Field
	OCF    string   // OpCode Command Firld
	Len    int      // Parameter Total Length
	Param  []field  // Command Parameters
	Return []field  // Return Parameters
	Events []string // Relevant events
}

type commands struct {
	LinkControl []cmd
	LinkPolicy  []cmd
	HostControl []cmd
	InfoParam   []cmd
	StatusParam []cmd
	LEControl   []cmd
}

func genCmd(b []byte, w io.Writer, t *template.Template) {
	var cmds commands
	if err := json.Unmarshal(b, &cmds); err != nil {
		log.Printf("failed to read spec.json, err: %s", err)
	}

	gen := func(t *template.Template, w io.Writer, cmds []cmd) {
		for _, c := range cmds {
			if err := t.Execute(w, c); err != nil {
				log.Fatalf("execution: %s", err)
			}
		}
	}

	gen(t, w, cmds.LinkControl)
	gen(t, w, cmds.LinkPolicy)
	gen(t, w, cmds.HostControl)
	gen(t, w, cmds.InfoParam)
	gen(t, w, cmds.StatusParam)
	gen(t, w, cmds.LEControl)
}

type evt struct {
	Name                string
	Spec                string
	Code                string
	SubCode             string
	Param               []field
	DefaultUnmarshaller bool
}

type events struct {
	Events []evt
}

func genEvt(b []byte, w io.Writer, t *template.Template) {
	var evts events
	if err := json.Unmarshal(b, &evts); err != nil {
		log.Printf("failed to read spec.json, err: %s", err)
	}
	for _, e := range evts.Events {
		if err := t.Execute(w, e); err != nil {
			log.Fatalf("execution: %s", err)
		}
	}
}

// Signal Packet format
type signal struct {
	Name   string
	Spec   string
	Code   string
	Fields []field
	Type   string
}

type signals struct {
	Signals []signal
}

func genSignal(b []byte, w io.Writer, t *template.Template) {
	var signals signals
	if err := json.Unmarshal(b, &signals); err != nil {
		log.Printf("failed to read spec.json, err: %s", err)
	}
	for _, p := range signals.Signals {
		if err := t.Execute(w, p); err != nil {
			log.Fatalf("execution: %s", err)
		}
	}
}

type att struct {
	Name  string
	Spec  string
	Code  string
	Param []field
	Type  string
}

type atts struct {
	Atts []att
}

func genAtt(b []byte, w io.Writer, t *template.Template) {
	var atts atts
	if err := json.Unmarshal(b, &atts); err != nil {
		log.Printf("failed to read spec.json, err: %s", err)
	}
	for _, p := range atts.Atts {
		if err := t.Execute(w, p); err != nil {
			log.Fatalf("execution: %s", err)
		}
	}
}

func main() {
	flag.Parse()

	b := input(*tmpl + ".json")
	w := output(*out)

	switch *tmpl {
	case "cmd":
		fmt.Fprintf(w, "package %s\n", *tmpl)
		t, err := template.New(*tmpl).Funcs(funcMap).Parse(string(input("cmd.tmpl")))
		if err != nil {
			log.Fatalf("parsing: %s", err)
		}
		genCmd(b, w, t)
	case "evt":
		fmt.Fprintf(w, "package %s\n", *tmpl)
		t, err := template.New(*tmpl).Funcs(funcMap).Parse(string(input("evt.tmpl")))
		if err != nil {
			log.Fatalf("parsing: %s", err)
		}
		genEvt(b, w, t)
	case "signal":
		fmt.Fprintf(w, "package l2cap\n")
		t, err := template.New(*tmpl).Funcs(funcMap).Parse(string(input("signal.tmpl")))
		if err != nil {
			log.Fatalf("parsing: %s", err)
		}
		genSignal(b, w, t)
	case "att":
		fmt.Fprintf(w, "package att\n")
		t, err := template.New(*tmpl).Funcs(funcMap).Parse(string(input("att.tmpl")))
		if err != nil {
			log.Fatalf("parsing: %s", err)
		}
		genAtt(b, w, t)
	}
}
