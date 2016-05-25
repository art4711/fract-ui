package gim

import (
	"fmt"
	"reflect"
	"strings"
)

/*
 * Instead of gtk we should solve this with more generic interfaces.
 *
 * The labels themselves should be interface { SetText(string) }
 * The grid should be replaced by something that allocates two labels.
 */

type Label interface {
	SetText(string)
}

type LabelPopulator interface {
	AddKV(string, int, int) (Label, Label)
}

type datalabel struct {
	name     string
	fmt      string
	keyLabel Label
	valLabel Label
}

type DataLabels struct {
	labels []datalabel
}

func (dl *DataLabels) Populate(src interface{}, populator LabelPopulator) {

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
		kl, vl := populator.AddKV(ln, 10, 10)
		dl.labels = append(dl.labels, datalabel{fmt: tags[0], name: ft.Name, keyLabel: kl, valLabel: vl})
	}
}

func (dl DataLabels) Update(obj interface{}) {
	v := reflect.ValueOf(obj)

	for _, l := range dl.labels {
		if l.valLabel != nil {
			l.valLabel.SetText(fmt.Sprintf(l.fmt, v.FieldByName(l.name).Interface()))
		}
	}
}
