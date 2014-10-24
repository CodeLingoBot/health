package hl7

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/iNamik/go_lexer"
	"github.com/iNamik/go_parser"
)

var unmarshalTests = []struct {
	file string
	out  []Segment
}{
	{"simple.hl7", simple_hl7_output},
}

func TestUnmarshal(t *testing.T) {
	for i, tt := range unmarshalTests {
		fp, err := os.Open("testdata/" + tt.file)
		if err != nil {
			t.Fatalf("#%d. received error: %s", i, err)
		}
		defer fp.Close()

		data, err := ioutil.ReadAll(fp)
		if err != nil {
			t.Fatalf("#%d. received error: %s", i, err)
		}

		out, err := Unmarshal(data)
		if err != nil {
			t.Fatalf("#%d. received error: %s", i, err)
		}

		if !reflect.DeepEqual(out, tt.out) {
			t.Fatalf("#%d: mismatch\nhave: %s\nwant: %s", i, getValidGo(out), getValidGo(tt.out))
		}
	}
}

func TestUnmarshalNoError(t *testing.T) {
	filenames, err := filepath.Glob("testdata/*.hl7")
	if err != nil {
		t.Fatalf("received error: %s", err)
	}
	for _, filename := range filenames {
		fp, err := os.Open(filename)
		if err != nil {
			t.Fatalf("%s: received error: %s", filename, err)
		}
		defer fp.Close()

		data, err := ioutil.ReadAll(fp)
		if err != nil {
			t.Fatalf("%s: received error: %s", filename, err)
		}

		out, err := Unmarshal(data)
		if err != nil {
			t.Fatalf("%s: received error: %s", filename, err)
		}

		if out == nil || len(out) == 0 {
			t.Fatalf("%s: expected segments, got none", filename)
		}
	}
}

func TestUnmarshalBadData(t *testing.T) {

	baddata := []byte("")
	_, err := Unmarshal(baddata)
	if err != nil {

		t.Fatal("unmarshalling an empty hl7 returned something")
	}

	//This causes a panic
	badheader := []byte("BAD|^~\\&|stuff|things")
	_, err = Unmarshal(badheader)
	if err == nil {
		t.Fatal("unmarshal did not error when given a bad header")
	}

	//this also panics
	invalidheader := []byte("MSH|^~\\&|\rbadseg|field")
	_, err = Unmarshal(invalidheader)
	if err == nil {
		t.Fatal("Did not error on a bad segment")
	}

}

func TestMultiple(t *testing.T) {
	data := []byte("MSH|^~\\&|||1^2^3^4^^^s1&s2&s3&&~r1~r2~r3~r4~~\rPV1|1^2^3\rPV2|1^2^3\r\r\r\n\r")
	expected := []Segment{
		Segment{
			Field("MSH"),
			Field("^~\\&"),
			Field(nil),
			Field(nil),
			Repeated{
				Component{
					Field("1"),
					Field("2"),
					Field("3"),
					Field("4"),
					Field(nil),
					Field(nil),
					SubComponent{
						Field("s1"),
						Field("s2"),
						Field("s3"),
						Field(nil),
						Field(nil),
					},
				},
				Field("r1"),
				Field("r2"),
				Field("r3"),
				Field("r4"),
				Field(nil),
				Field(nil),
			},
		},
		Segment{
			Field("PV1"),
			Component{
				Field("1"),
				Field("2"),
				Field("3"),
			},
		},
		Segment{
			Field("PV2"),
			Component{
				Field("1"),
				Field("2"),
				Field("3"),
			},
		},
	}

	out, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("received error: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("mismatch\nhave: %s\nwant: %s", getValidGo(out), getValidGo(expected))
	}
}

func BenchmarkParser(b *testing.B) {
	fp, err := os.Open("testdata/simple.hl7")
	if err != nil {
		b.Fatalf("received error: %s", err)
	}
	defer fp.Close()

	data, err := ioutil.ReadAll(fp)
	if err != nil {
		b.Fatalf("received error: %s", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		lexState := newLexerState()
		l := lexer.New(lexState.lexHeader, reader, 3)
		parseState := newParserState(lexState)
		p := parser.New(parseState.parse, l, 3)
		p.Next()
	}
}

func getValidGo(segments []Segment) string {
	b := &bytes.Buffer{}
	printValidGo(b, segments)
	return b.String()
}

func printValidGo(w io.Writer, segments []Segment) {
	indent := "  "
	fmt.Fprintln(w, "[]Segment{")
	for _, segment := range segments {
		printValidGo_(w, indent, segment)
	}
	fmt.Fprintln(w, "}")
}

func printValidGo_(w io.Writer, indent string, data Data) {
	switch t := data.(type) {
	case Segment:
		fmt.Fprintln(w, indent+"Segment{")
		for _, v := range t {
			printValidGo_(w, indent+"  ", v)
		}
		fmt.Fprintln(w, indent+"},")
	case Component:
		fmt.Fprintln(w, indent+"Component{")
		for _, v := range t {
			printValidGo_(w, indent+"  ", v)
		}
		fmt.Fprintln(w, indent+"},")
	case SubComponent:
		fmt.Fprintln(w, indent+"SubComponent{")
		for _, v := range t {
			printValidGo_(w, indent+"  ", v)
		}
		fmt.Fprintln(w, indent+"},")
	case Repeated:
		fmt.Fprintln(w, indent+"Repeated{")
		for _, v := range t {
			printValidGo_(w, indent+"  ", v)
		}
		fmt.Fprintln(w, indent+"},")
	case Field:
		v := "nil"
		if t != nil {
			v = `"` + t.String() + `"`
		}
		fmt.Fprintf(w, indent+"Field(%s),\n", v)
	}
}

var (
	simple_hl7_output = []Segment{
		Segment{
			Field("MSH"),
			Field(`^~\&`),
			Field("field"),
			Field(`\|~^&HEY`),
			Component{
				Field("component1"),
				Field("component2"),
			},
			Component{
				SubComponent{
					Field("subcomponent1a"),
					Field("subcomponent2a"),
				},
				SubComponent{
					Field("subcomponent1b"),
					Field("subcomponent2b"),
				},
			},
			Repeated{
				Component{
					Field("component1a"),
					Field("component2a"),
				},
				Component{
					Field("component1b"),
					Field("component2b"),
				},
			},
		},
	}
)
