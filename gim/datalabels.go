package gim

import (
	"github.com/gotk3/gotk3/gtk"
	"reflect"
	"log"
	"strings"
	"fmt"
)

type datalabel struct {
	name string
	fmt string
	keyLabel *gtk.Label
	valLabel *gtk.Label
}

type DataLabels struct {
	labels []datalabel
}

func (dl *DataLabels)populate(src interface{}, gr *gtk.Grid) {
	l := func(s string) *gtk.Label {
		label, err := gtk.LabelNew(s)
		if err != nil {
			log.Fatal(err)
		}
		label.SetWidthChars(10)
		return label
	}

	srcv := reflect.ValueOf(src)
	srct := srcv.Type()

	for i := 0; i < srct.NumField(); i++ {
		ft := srct.Field(i)
		tags := strings.SplitN(ft.Tag.Get("dl"), ",", 2)
		if tags[0] == "" {
			continue
		}

		ln := ft.Name
		if len(tags) == 2 {
			ln = tags[1]
		}
		dl.labels = append(dl.labels, datalabel{ fmt: tags[0], name: ft.Name, keyLabel: l(ln), valLabel: l("") })
	}

	for i := range dl.labels {
		gr.Attach(dl.labels[i].keyLabel, 0, i, 1, 1)
		gr.Attach(dl.labels[i].valLabel, 1, i, 1, 1)		
	}
}

func (dl DataLabels)update(obj interface{}) {
	v := reflect.ValueOf(obj)
	
	for _, l := range dl.labels {
		if l.valLabel != nil {
			l.valLabel.SetText(fmt.Sprintf(l.fmt, v.FieldByName(l.name).Interface()))
		}
	}
}

