package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"9fans.net/go/acme"
	"9fans.net/go/plan9"
	"9fans.net/go/plumb"

	"github.com/edma2/zincindexd/index"
)

func leafOf(name string) string {
	if i := strings.LastIndexByte(name, '.'); i != -1 && i+1 <= len(name) {
		return name[i+1:]
	}
	return ""
}

func candidatesOf(name string) []string {
	candidates := []string{}
	elems := strings.Split(name, ".")
	for i, _ := range elems {
		candidates = append(candidates, strings.Join(elems[0:i+1], "."))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
	return candidates
}

func plumbFile(m *plumb.Message, send io.Writer, name, path string) {
	m.Src = "zincindexd"
	m.Dst = ""
	m.Data = []byte(path)
	var attr *plumb.Attribute
	for attr = m.Attr; attr != nil; attr = attr.Next {
		if attr.Name == "addr" {
			break
		}
	}
	if attr == nil {
		if leafName := leafOf(name); leafName != "" {
			addr := fmt.Sprintf("/(trait|class|object|interface)[ 	]*%s/", leafName)
			m.Attr = &plumb.Attribute{Name: "addr", Value: addr, Next: m.Attr}
		}
	}
	if err := m.Send(send); err != nil {
		log.Printf("send error: %s\n", err)
	}
}

func newWin(title string) (*acme.Win, error) {
	win, err := acme.New()
	if err != nil {
		return nil, err
	}
	win.Name(title)
	return win, nil
}

func openWin(name string, childNames []string) {
	w, err := newWin("/zinc/" + name)
	if err != nil {
		log.Printf("acme win: %s\n", err)
	}
	for _, name := range childNames {
		if !strings.ContainsRune(name, '$') {
			w.Fprintf("body", "%s\n", name)
		}
	}
	w.Ctl("clean")
	w.Addr("#0")
	w.Ctl("show")
}

func serve(idx *index.Index) error {
	recv, err := plumb.Open("zincindexd", plan9.OREAD)
	if err != nil {
		return err
	}
	defer recv.Close()
	r := bufio.NewReader(recv)
	send, err := plumb.Open("send", plan9.OWRITE)
	if err != nil {
		return err
	}
	defer send.Close()
	for {
		m := plumb.Message{}
		err := m.Recv(r)
		if err != nil {
			log.Printf("recv error: %s\n", err)
		}
		name := string(m.Data)
		var get *index.GetResult
		for _, c := range candidatesOf(name) {
			if get = idx.Get(c); get != nil {
				break
			}
		}
		if get == nil {
			log.Println("couldn't find " + name)
			continue
		}
		if get.Path != "" {
			plumbFile(&m, send, name, get.Path)
		}
		if get.Children != nil {
			openWin(name, get.Children)
		}
	}
	return nil
}

func Main() error {
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		return nil
	}
	for _, path := range paths {
		log.Println("Watching " + path)
	}
	idx := index.NewIndex()
	defer idx.Stop()
	for _, path := range paths {
		if err := idx.Watch(path); err != nil {
			return err
		}
	}
	return serve(idx)
}

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
}
