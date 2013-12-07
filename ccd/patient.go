package ccd

import (
	"fmt"
	"github.com/jteeuwen/go-pkg-xmlx"
	"time"
)

var (
	PatientParser = Parser{
		Type:     PARSE_DOC,
		Priority: 0,
		Func:     parsePatient,
	}
)

type Name struct {
	Last     string
	First    string
	Middle   string
	Suffix   string
	Prefix   string // title
	Type     string // L = legal name, PN = patient name (not sure)
	NickName string
}

func (n Name) IsZero() bool {
	return n == (Name{})
}

type Address struct {
	Line1   string
	Line2   string
	City    string
	County  string
	State   string
	Zip     string
	Country string
	Type    string // H or HP = home, TMP = temporary, WP = work/office
}

func (a Address) IsZero() bool {
	return a == (Address{})
}

type Patient struct {
	Name          Name
	Dob           time.Time
	Address       Address
	Gender        string
	MaritalStatus string
	RaceCode      string
	EthnicityCode string
}

func (p Patient) IsZero() bool {
	return p == (Patient{})
}

// parses patient information from the CCD and returns
// a Patient struct
func parsePatient(root *xmlx.Node, ccd *CCD) []error {
	anode := Nget(root, "ClinicalDocument", "recordTarget", "patientRole", "addr")
	// address isn't always present
	if anode != nil {
		ccd.Patient.Address.Type = anode.As("*", "use")
		lines := anode.SelectNodes("*", "streetAddressLine")
		if len(lines) > 0 {
			ccd.Patient.Address.Line1 = lines[0].GetValue()
		}
		if len(lines) > 1 {
			ccd.Patient.Address.Line2 = lines[1].GetValue()
		}
		ccd.Patient.Address.City = anode.S("*", "city")
		ccd.Patient.Address.County = anode.S("*", "county")
		ccd.Patient.Address.State = anode.S("*", "state")
		ccd.Patient.Address.Zip = anode.S("*", "postalCode")
		ccd.Patient.Address.Country = anode.S("*", "country")
	}

	pnode := Nget(root, "ClinicalDocument", "recordTarget", "patientRole", "patient")
	if pnode == nil {
		return []error{
			fmt.Errorf("Could not find the node in CCD: ClinicalDocument/recordTarget/patientRole/patient"),
		}
	}

	for n, nameNode := range pnode.SelectNodes("*", "name") {
		given := nameNode.SelectNodes("*", "given")
		// This is a NickName if it's the second <name><given> tag block or the
		// given tag has the qualifier CM.
		if n == 1 || (len(given) > 0 && given[0].As("*", "qualifier") == "CM") {
			ccd.Patient.Name.NickName = given[0].GetValue()
			continue
		}

		ccd.Patient.Name.Type = nameNode.As("*", "use")
		if len(given) > 0 {
			ccd.Patient.Name.First = given[0].GetValue()
		}
		if len(given) > 1 {
			ccd.Patient.Name.Middle = given[1].GetValue()
		}
		ccd.Patient.Name.Last = nameNode.S("*", "family")
		ccd.Patient.Name.Prefix = nameNode.S("*", "prefix")
		suffixes := nameNode.SelectNodes("*", "suffix")
		for n, suffix := range suffixes {
			// if it's the second suffix, or it has the qualifier TITLE
			if n == 1 || (len(ccd.Patient.Name.Prefix) == 0 && suffix.As("*", "qualifier") == "TITLE") {
				ccd.Patient.Name.Prefix = suffix.GetValue()
			} else {
				ccd.Patient.Name.Suffix = suffix.GetValue()
			}
		}
	}

	birthNode := pnode.SelectNode("*", "birthTime")
	if birthNode != nil {
		ccd.Patient.Dob, _ = ParseHL7Time(birthNode.As("*", "value"))
	}

	genderNode := pnode.SelectNode("*", "administrativeGenderCode")
	if genderNode != nil && genderNode.As("*", "codeSystem") == "2.16.840.1.113883.5.1" {
		switch genderNode.As("*", "code") {
		case "M":
			ccd.Patient.Gender = "Male"
		case "F":
			ccd.Patient.Gender = "Female"
		case "UN":
			ccd.Patient.Gender = "Undifferentiated"
		default:
			ccd.Patient.Gender = "Unknown"
		}
	}

	maritalNode := pnode.SelectNode("*", "maritalStatusCode")
	if maritalNode != nil && maritalNode.As("*", "codeSystem") == "2.16.840.1.113883.5.2" {
		ccd.Patient.MaritalStatus = maritalNode.As("*", "code")
	}

	raceNode := pnode.SelectNode("*", "raceCode")
	if raceNode != nil && raceNode.As("*", "codeSystem") == "2.16.840.1.113883.6.238" {
		ccd.Patient.RaceCode = raceNode.As("*", "code")
	}

	ethnicNode := pnode.SelectNode("*", "ethnicGroupCode")
	if ethnicNode != nil && ethnicNode.As("*", "codeSystem") == "2.16.840.1.113883.6.238" {
		ccd.Patient.EthnicityCode = ethnicNode.As("*", "code")
	}

	return nil
}
