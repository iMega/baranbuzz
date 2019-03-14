package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
)

func main() {
	var filename = "418.html"
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("failed to read file %s", err)
	}

	buf := bytes.NewReader(b)

	node, err := html.Parse(buf)
	if err != nil {
		log.Fatalf("failed to parse html, %s", err)
	}

	foundNode, err := getParent(node)

	var csv []string

	current := foundNode.NextSibling
	for {
		if node != nil && current.Data == "h2" {
			fmt.Println(node.DataAtom.String())
			break
		}
		node, _ := nextTable(current)

		if node != nil {
			piece := tableToCsv(node)
			csv = append(csv, piece...)
		}

		current = current.NextSibling
	}

	file, err := os.Create(filename + ".csv")
	if err != nil {
		log.Fatalf("failed to create file, %s", err)
	}

	for _, l := range csv {
		file.WriteString(l + "\n")
	}
}

func nextTable(doc *html.Node) (*html.Node, error) {
	var b *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n == nil {
			return
		}

		if n != nil && n.Type == html.ElementNode && n.Data == "table" && len(n.Attr) > 0 {
			if n.Attr[0].Key == "cellpadding" && n.Attr[0].Val == "4" {
				//fmt.Printf("%#v\n\n", n)
				b = n
			}

		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	if b != nil {
		return b, nil
	}

	return nil, errors.New("Missing <table> in the node tree")
}

func getParent(doc *html.Node) (*html.Node, error) {
	var b *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" && n.Attr[0].Key == "name" {
			if n.FirstChild != nil {
				nnn := n.FirstChild.Data
				utf := string(DecodeWindows1251([]byte(nnn)))

				if utf == "Данные ГН " {
					b = n.Parent
				}

			}

		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	if b != nil {
		return b, nil
	}

	return nil, errors.New("Missing <body> in the node tree")
}

func DecodeWindows1251(in []byte) []byte {
	dec := charmap.Windows1251.NewDecoder()
	out, _ := dec.Bytes(in)
	return out
}

type TableData struct {
	Time     time.Time
	KWhDel   string
	KWhRec   string
	KVARhDel string
	KVARhRec string
}

func nextTR(doc *html.Node) (*html.Node, error) {
	var b *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n == nil {
			return
		}

		if n != nil && n.Type == html.ElementNode && n.Data == "tr" {
			b = n
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	if b != nil {
		return b, nil
	}

	return nil, errors.New("Missing <tr> in the node tree")
}

func nextThTd(doc *html.Node) (*html.Node, error) {
	var b *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n == nil {
			return
		}

		if n != nil && n.Type == html.ElementNode && (n.Data == "th" || n.Data == "td") {
			b = n
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	if b != nil {
		return b, nil
	}

	return nil, errors.New("Missing <th,td> in the node tree")
}

func ThTdTagsToTableData(date time.Time, thtd *html.Node) (*TableData, *time.Time, error) {
	var result []string
	current := thtd
	for {

		if current == nil {
			break
		}

		node, _ := nextThTd(current)

		if node != nil {
			result = append(result, string(DecodeWindows1251([]byte(node.FirstChild.Data))))
		}

		current = current.NextSibling
	}

	if result[0] == "Время" {
		return nil, nil, errors.New("title")
	}

	if strings.Contains(result[0], "Дата") {
		sp := strings.Split(result[0], ":")

		date, _ := time.Parse("02.01.2006", strings.TrimSpace(sp[1]))

		return nil, &date, nil
	}

	t, _ := time.Parse("15:04", result[0])

	return &TableData{
		Time:     t.AddDate(date.Year(), int(date.Month()), date.Day()),
		KWhDel:   result[1],
		KWhRec:   result[2],
		KVARhDel: result[3],
		KVARhRec: result[4],
	}, nil, nil
}

func tableToCsv(table *html.Node) []string {
	var (
		result []string
		d      time.Time
	)

	tbody := table.FirstChild.NextSibling

	titleTR := tbody.FirstChild

	current := titleTR
	for {
		if current == nil {
			break
		}
		node, _ := nextTR(current)

		if node != nil {
			dt, date, err := ThTdTagsToTableData(d, node.FirstChild)
			if err != nil {
				current = current.NextSibling
				continue
			}

			if date != nil {
				d = *date
				current = current.NextSibling
				continue
			}

			result = append(result, fmt.Sprintf("%s;%s;%s;%s;%s;%s", dt.Time.Format("02.01.2006"), dt.Time.Format("15:04:00"), dt.KWhDel, dt.KWhRec, dt.KVARhDel, dt.KVARhRec))
		}

		current = current.NextSibling
	}

	return result
}
