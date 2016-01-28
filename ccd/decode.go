package ccd

import (
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html/charset"

	"github.com/jteeuwen/go-pkg-xmlx"
)

var (
	DefaultParsers = []Parser{
		PatientParser, AllergiesParser, ImmunizationsParser,
		MedicationsParser, ProblemsParser,
		ResultsParser, VitalSignsParser, SocialHistoryParser, EncountersParser,
	}
)

type Code struct {
	CodeSystemName string
	Type           string
	CodeSystem     string
	Code           string
	DisplayName    string
}

func (c *Code) decode(n *xmlx.Node) {
	if n == nil {
		return
	}
	c.CodeSystem = n.As("*", "codeSystem")
	// code.CodeSystemName, _ = codeSystemToName(code.CodeSystem)
	c.Code = n.As("*", "code")
	c.DisplayName = n.As("*", "displayName")
	c.Type = n.As("*", "type")
}

type CCD struct {
	Patient       Patient
	Immunizations []Immunization
	Medications   []Medication
	Problems      []Problem
	Results       []Result
	VitalSigns    []VitalSign
	Allergies     []Allergy
	SocialHistory []SocialHistory
	Encounters    []Encounter

	// Right now doc_parsers will only have one map entry "*"
	doc_parsers     map[string]Parsers
	section_parsers map[string]Parsers
}

// New CCD object with no parsers. Use NewDefaultCCD()
// or add your own parsers with AddParsers() if you want
// to actually parse anything.
func NewCCD() *CCD {
	c := &CCD{}
	c.doc_parsers = make(map[string]Parsers)
	c.section_parsers = make(map[string]Parsers)
	return c
}

// New CCD object with all the default parsers
func NewDefaultCCD() *CCD {
	c := NewCCD()
	c.AddParsers(DefaultParsers...)
	return c
}

func (c *CCD) AddParsers(parsers ...Parser) {
	for _, p := range parsers {
		if p.Organization == "" {
			p.Organization = "*"
		}

		p.Organization = strings.ToLower(p.Organization)

		if p.Type == PARSE_DOC {
			if p.Values == nil {
				p.Values = []string{"*"}
			}

			for _, v := range p.Values {
				c.doc_parsers[v] = insertSortParser(p, c.doc_parsers[v])
			}
		} else if p.Type == PARSE_SECTION {
			if p.Values == nil {
				panic("ccd: Section parser cannot have an empty Value.")
			}

			for _, v := range p.Values {
				c.section_parsers[v] = insertSortParser(p, c.section_parsers[v])
			}
		} else {
			panic("ccd: Unknown parser type.")
		}
	}
}

func (c *CCD) ParseFile(filename string) error {
	doc := xmlx.New()
	err := doc.LoadFile(filename, charset.NewReaderLabel)
	if err != nil {
		return err
	}

	return c.ParseDoc(doc)
}

func (c *CCD) ParseStream(r io.Reader) error {
	doc := xmlx.New()
	err := doc.LoadStream(r, charset.NewReaderLabel)
	if err != nil {
		return err
	}

	return c.ParseDoc(doc)
}

func (c *CCD) Parse(data []byte) error {
	doc := xmlx.New()
	err := doc.LoadBytes(data, charset.NewReaderLabel)
	if err != nil {
		return err
	}

	return c.ParseDoc(doc)
}

type ParseType int

const (
	PARSE_DOC ParseType = iota
	PARSE_SECTION
)

type ParseFunc func(root *xmlx.Node, ccd *CCD) []error

type Parser struct {
	Type         ParseType
	Values       []string
	Organization string
	Priority     int
	Func         ParseFunc
}

type Parsers []Parser

// Parses a CCD into a CCD struct.
func (c *CCD) ParseDoc(doc *xmlx.Document) error {
	var errs_ []error
	// var errs []error

	// Reset any data retrieved from another parse
	c.Patient = Patient{}
	c.Immunizations = nil
	c.Medications = nil
	c.Problems = nil
	c.Results = nil
	c.VitalSigns = nil
	c.Allergies = nil
	c.SocialHistory = nil

	if Nget(doc.Root, "ClinicalDocument") == nil {
		return errors.New("invalid CCD")
	}

	nRecordTarget := Nget(doc.Root, "recordTarget")

	org := Nget(nRecordTarget, "providerOrganization", "name")
	orgName := "*"
	if org != nil {
		orgName = strings.ToLower(org.S("*", "name"))
	}

	for _, p := range c.doc_parsers["*"] {
		if orgName == "*" || p.Organization == "*" || orgName == p.Organization {
			errs_ = p.Func(doc.Root, c)
			//errs = append(errs, errs_...)
		}
	}

	componentNode := Nget(doc.Root, "component", "structuredBody")
	if componentNode != nil {
		componentNodes := componentNode.SelectNodes("*", "component")
		for _, componentNode := range componentNodes {
			sectionNode := componentNode.SelectNode("*", "section")

			tidNodes := sectionNode.SelectNodes("*", "templateId")
			for _, tidNode := range tidNodes {
				tid := tidNode.As("*", "root")
				if parsers, ok := c.section_parsers[tid]; ok {
					for _, p := range parsers {
						if orgName == "*" || p.Organization == "*" || orgName == p.Organization {
							errs_ = p.Func(sectionNode, c)
							//errs = append(errs, errs_...)
						}
					}
				}
			}
		}
	}

	// disabling errors for now. may return "warnings" or something
	_ = errs_

	return nil
}
