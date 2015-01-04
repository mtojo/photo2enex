package main

import (
	"crypto/md5"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

const HEADER_TEMPLATE = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE en-export SYSTEM "http://xml.evernote.com/pub/evernote-export.dtd">
<en-export export-date="{{.exportDate}}" application="Evernote/Windows" version="4.x">
`

const NOTE_TEMPLATE = `
<note><title>{{.filename}}</title><content><![CDATA[<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE en-note SYSTEM "http://xml.evernote.com/pub/enml2.dtd">
<en-note><en-media hash="{{.hash}}" type="{{.mime}}"/>
</en-note>
]]></content><created>{{.created}}</created><resource><data encoding="base64">
{{.data}}
</data><mime>{{.mime}}</mime><resource-attributes><source-url>{{.sourceUrl}}</source-url><file-name>{{.filename}}</file-name></resource-attributes></resource></note>
`

const FOOTER_TEMPLATE = `</en-export>
`

type Photo struct {
	path string
	mime string
	tm   time.Time
	lat  float64
	long float64
}

var photos []Photo

func readFileInfo(fname string) {
	file, err := os.Open(fname)

	if err != nil {
		panic(err)
	}

	m := mime.TypeByExtension(path.Ext(fname))
	if m == "" {
		m = "application/octet-stream"
	}

	if x, err := exif.Decode(file); err == nil {
		tm, _ := x.DateTime()
		lat, long, _ := x.LatLong()
		photos = append(photos, Photo{fname, m, tm, lat, long})
	} else {
		fi, _ := os.Stat(fname)
		photos = append(photos, Photo{fname, m, fi.ModTime(), 0, 0})
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [-o] [file ...]\n\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	output_filename := flag.String("o", "Photos.enex", "output filename")
	flag.Parse()

	if _, err := os.Stat(*output_filename); !os.IsNotExist(err) {
		fmt.Println("output file is already exists")
		return
	}

	args := flag.Args()
	for i := 0; i < len(args); i++ {
		fname := args[i]

		err := filepath.Walk(fname,
			func(fname string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !info.IsDir() && path.Base(fname)[0] != '.' {
					readFileInfo(fname)
				}

				return nil
			})

		if err != nil {
			panic(err)
		}
	}

  if len(photos) == 0 {
    fmt.Println("no input file specified")
    return
  }

  out, err := os.Create(*output_filename)
  if err != nil {
    fmt.Println("failed to create output file")
    return
  }

  header := template.New("header")
  template.Must(header.Parse(HEADER_TEMPLATE))
  header.Execute(out, map[string]string{
    "exportDate": time.Now().Format(time.RFC3339),
  })

  for i := 0; i < len(photos); i++ {
    data, _ := ioutil.ReadFile(photos[i].path)

    hash := md5.New()
    hash.Write(data)

    note := template.New("note")
    template.Must(note.Parse(NOTE_TEMPLATE))
    note.Execute(out, map[string]string{
      "filename":  path.Base(photos[i].path),
      "hash":      fmt.Sprintf("%x", hash.Sum(nil)),
      "mime":      photos[i].mime,
      "created":   photos[i].tm.Format(time.RFC3339),
      "data":      base64.StdEncoding.EncodeToString(data),
      "sourceUrl": fmt.Sprintf("file://%s", photos[i].path),
      "lat":       strconv.FormatFloat(photos[i].lat, 'f', 4, 64),
      "long":      strconv.FormatFloat(photos[i].long, 'f', 4, 64),
    })
  }

  footer := template.New("footer")
  template.Must(footer.Parse(FOOTER_TEMPLATE))
  footer.Execute(out, map[string]string{})

  fmt.Printf("%d file(s) proceeded\n", len(photos))
}
