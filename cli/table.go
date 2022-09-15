package cli

import (
	"io"
	"os"

	gotable "github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
)

type Table interface {
	SetHeader(v ...interface{}) Table
	AddRow(v ...interface{}) Table
	SetStyle(Style) Table
	SetOutput(w io.Writer)
	Render()
}

type Style = gotable.Style

type table struct {
	tw gotable.Writer
}

func NewTable() Table {
	t := &table{
		tw: gotable.NewWriter(),
	}
	style := gotable.StyleDefault
	style.Format.Header = text.FormatDefault
	t.tw.SetStyle(style)
	t.tw.SetOutputMirror(os.Stdout)
	return t
}

func (t *table) SetStyle(style Style) Table {
	t.tw.SetStyle(style)
	return t
}

func (t *table) SetOutput(w io.Writer) {
	t.tw.SetOutputMirror(w)
}

func (t *table) SetHeader(v ...interface{}) Table {
	t.tw.AppendHeader(cellsToRow(v...))
	return t
}

func (t *table) AddRow(cells ...interface{}) Table {
	t.tw.AppendRows([]gotable.Row{cellsToRow(cells...)})
	return t
}

func cellsToRow(list ...interface{}) gotable.Row {
	return gotable.Row(list)
}

func (t *table) Render() {
	t.tw.Render()
}
