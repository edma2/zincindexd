package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"flag"

	"9fans.net/go/acme"
	"9fans.net/go/plan9"
	"9fans.net/go/plumb"

	"github.com/edma2/classy/index"
	"github.com/edma2/classy/zinc"
	"github.com/edma2/classy/zinc/fsevents"
)

func showChildren(idx *index.Index, name string) error {
	var w *acme.Win = nil
	var err error
	title := "/c/" + name + "/"
	infos, err := acme.Windows()
	if err != nil {
		return err
	}
	for _, info := range infos {
		if info.Name == title {
			w, err = acme.Open(info.ID, nil)
			if err != nil {
				return err
			}
		}
	}
	if w == nil {
		w, err = newWin(title)
		if err != nil {
			return err
		}
	}
	w.Addr(",")
	w.Write("data", nil)
	idx.Walk(name, func(name string) {
		if !strings.ContainsRune(name, '$') {
			w.Fprintf("body", "%s\n", name)
		}
	})
	w.Fprintf("addr", "#0")
	w.Ctl("dot=addr")
	w.Ctl("show")
	w.Ctl("clean")
	return nil
}

func newWin(title string) (*acme.Win, error) {
	win, err := acme.New()
	if err != nil {
		return nil, err
	}
	win.Name(title)
	return win, nil
}

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

func plumbFile(m *plumb.Message, w io.Writer, name, path string) error {
	m.Src = "classy"
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
	log.Printf("Sending to plumber: %s\n", m)
	return m.Send(w)
}

func serve(idx *index.Index) error {
	fid, err := plumb.Open("editclass", plan9.OREAD)
	if err != nil {
		return err
	}
	defer fid.Close()
	r := bufio.NewReader(fid)
	w, err := plumb.Open("send", plan9.OWRITE)
	if err != nil {
		return err
	}
	defer w.Close()
	for {
		m := plumb.Message{}
		err := m.Recv(r)
		if err != nil {
			return err
		}
		log.Printf("Received from plumber: %s\n", m)
		name := string(m.Data)
		if strings.HasPrefix(m.Dir, "/c/") {
			name = strings.TrimPrefix(m.Dir, "/c/") + "." + name
		}
		var get *index.GetResult
		for _, c := range candidatesOf(name) {
			if get = idx.Get(c); get != nil {
				break
			}
		}
		if get == nil {
			log.Printf("Found no results for: %s\n", name)
			continue
		}
		if get.Path != "" {
			if err := plumbFile(&m, w, name, get.Path); err != nil {
				log.Printf("%s: %s\n", get.Path, err)
			}
		} else if get.Children != nil {
			if err := showChildren(idx, name); err != nil {
				log.Printf("error opening win: %s\n", err)
			}
		} else {
			log.Printf("Result was empty: %s\n", name)
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
	for _, path := range paths {
		idx.Watch(zinc.Watch(fsevents.Watch(path)))
	}
	return serve(idx)
}

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
}
