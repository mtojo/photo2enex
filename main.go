package main

import (
	"crypto/md5"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

const headerTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE en-export SYSTEM "http://xml.evernote.com/pub/evernote-export.dtd">
<en-export export-date="{{.exportDate}}" application="Evernote/Windows" version="4.x">
`

const noteTemplate = `
<note><title>{{.filename}}</title><content><![CDATA[<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE en-note SYSTEM "http://xml.evernote.com/pub/enml2.dtd">
<en-note><en-media hash="{{.hash}}" type="{{.mime}}"/>
</en-note>
]]></content><created>{{.created}}</created><resource><data encoding="base64">
{{.data}}
</data><mime>{{.mime}}</mime><resource-attributes><source-url>{{.sourceUrl}}</source-url><file-name>{{.filename}}</file-name></resource-attributes></resource></note>
`

const footerTemplate = `</en-export>
`

type Photo struct {
	path string
	mime string
	tm   time.Time
	lat  float64
	long float64
}

var photos []Photo

func getFileTime(fname string) time.Time {
  var y, d, h, i, s int
  var m time.Month

  if _, err := fmt.Sscanf(fname, "%d-%d-%d %d.%d.%d", &y, &m, &d, &h, &i, &s); err == nil {
    return time.Date(y, m, d, h, i, s, 0, time.Local).UTC()
  }

  fi, _ := os.Stat(fname)
  return fi.ModTime()
}

func readFileInfo(fname string) {
	file, err := os.Open(fname)

	if err != nil {
		panic(err)
	}

	m := mime.TypeByExtension(filepath.Ext(fname))
	if m == "" {
		m = "application/octet-stream"
	}

	if x, err := exif.Decode(file); err == nil {
		tm, noDateTimePresent := x.DateTime()
		lat, long, _ := x.LatLong()
    if noDateTimePresent != nil {
      tm = getFileTime(fname)
    }
		photos = append(photos, Photo{fname, m, tm, lat, long})
	} else {
    tm := getFileTime(fname)
		photos = append(photos, Photo{fname, m, tm, 0, 0})
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [-o] [file ...]\n\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	outputFilename := flag.String("o", "Photos.enex", "output filename")
	flag.Parse()

	if _, err := os.Stat(*outputFilename); !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "output file is already exists")
		os.Exit(1)
	}

	args := flag.Args()
	for i := 0; i < len(args); i++ {
		fname := args[i]

		err := filepath.Walk(fname,
			func(fname string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() && filepath.Base(fname)[0] != '.' {
					readFileInfo(fname)
				}

				return nil
			})

		if err != nil {
			panic(err)
		}
	}

	if len(photos) == 0 {
		fmt.Fprintln(os.Stderr, "no input file specified")
		os.Exit(1)
	}

	out, err := os.Create(*outputFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create output file")
		os.Exit(1)
	}
	defer out.Close()

	header := template.New("header")
	template.Must(header.Parse(headerTemplate))
	header.Execute(out, map[string]string{
		"exportDate": time.Now().Format(time.RFC3339),
	})

	for i := 0; i < len(photos); i++ {
		data, _ := ioutil.ReadFile(photos[i].path)

		hash := md5.New()
		hash.Write(data)

		note := template.New("note")
		template.Must(note.Parse(noteTemplate))
		note.Execute(out, map[string]string{
			"filename":  filepath.Base(photos[i].path),
			"hash":      fmt.Sprintf("%x", hash.Sum(nil)),
			"mime":      photos[i].mime,
			"created":   photos[i].tm.Format("20060102T150405Z"),
			"data":      base64.StdEncoding.EncodeToString(data),
			"sourceUrl": fmt.Sprintf("file://%s", photos[i].path),
			"lat":       strconv.FormatFloat(photos[i].lat, 'f', 4, 64),
			"long":      strconv.FormatFloat(photos[i].long, 'f', 4, 64),
		})
	}

	footer := template.New("footer")
	template.Must(footer.Parse(footerTemplate))
	footer.Execute(out, map[string]string{})

	fmt.Printf("%d file(s) proceeded\n", len(photos))
}
