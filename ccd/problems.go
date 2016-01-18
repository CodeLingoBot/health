package ccd

import (
	"time"

	"github.com/jteeuwen/go-pkg-xmlx"
)

var (
	ProblemsTid = []string{"2.16.840.1.113883.10.20.1.11", "2.16.840.1.113883.10.20.22.2.5.1"}

	ProblemsParser = Parser{
		Type:     PARSE_SECTION,
		Values:   ProblemsTid,
		Priority: 0,
		Func:     parseProblems,
	}
)

type Problem struct {
	Name string
	Date time.Time
	// Duration time.Duration
	Status string
	Code   Code
}

func parseProblems(node *xmlx.Node, ccd *CCD) []error {
	entryNodes := node.SelectNodes("*", "entry")
	for _, entryNode := range entryNodes {
		problem := Problem{}

		observationNode := Nget(entryNode, "act", "entryRelationship", "observation")
		valueNode := Nget(observationNode, "value")
		if valueNode == nil {
			continue
		}
		problem.Name = valueNode.As("*", "displayName")

		translation := Nget(valueNode, "translation")
		if translation != nil {
			problem.Code.decode(translation)
			//if the value node didnt have a displayName, try the translation node.
			if problem.Name == "" && problem.Code.DisplayName != "" {
				problem.Name = problem.Code.DisplayName
			}
		}

		effectiveTimeNode := Nget(observationNode, "effectiveTime")
		t := decodeTime(effectiveTimeNode)
		problem.Date = t.Value

		// observationNode2 := Nget(observationNode, "entryRelationship", "observation")
		// if observationNode2 != nil {
		//   problem.Status = Nget(observationNode2, "value").As("*", "displayName")
		// }

		erNodes := observationNode.SelectNodes("*", "entryRelationship")
		for _, erNode := range erNodes {
			oNode := Nget(erNode, "observation")
			codeNode := Nget(oNode, "code")
			valueNode := Nget(oNode, "value")

			if codeNode == nil {
				continue
			}

			if codeNode.As("*", "code") == "33999-4" { // Status
				problem.Status = valueNode.As("*", "displayName")
			}
		}

		ccd.Problems = append(ccd.Problems, problem)
	}

	return nil
}
