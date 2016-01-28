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

//Problem represents an Observation Problem  (templateId: 2.16.840.1.113883.10.20.22.4.4)
type Problem struct {
	Name string
	Date time.Time
	// Duration time.Duration
	Status      string
	ProblemType string
	Code        Code
}

func parseProblems(node *xmlx.Node, ccd *CCD) []error {
	entryNodes := node.SelectNodes("*", "entry")
	for _, entryNode := range entryNodes {
		observationNode := Nget(entryNode, "act", "entryRelationship", "observation")
		problem := decodeProblem(observationNode)

		ccd.Problems = append(ccd.Problems, problem)
	}

	return nil
}

func decodeProblem(node *xmlx.Node) Problem {
	problem := Problem{}

	valueNode := Nget(node, "value")
	if valueNode == nil {
		//the spec says there must be a value, but better to be safe than to panic.
		return Problem{}
	}
	problem.Name = valueNode.As("*", "displayName")

	//The spec seems to imply we should look at translation when value's nullFlavor is OTH
	//in my testing, it appears we should always check for it.

	//This current implementation decodes the value node to Code, then replaces it with its translation if it exists.
	//This does not seem right, but seems to lead to the best results in our samples.
	problem.Code.decode(valueNode)
	if translation := Nget(valueNode, "translation"); translation != nil {
		problem.Code.decode(translation)
		if problem.Name == "" {
			problem.Name = problem.Code.DisplayName
		}

	}

	//get the problem type from the highest level code node
	if topCode := Nget(node, "code"); topCode != nil {
		name := topCode.As("*", "displayName")
		if name == "" {
			name = topCode.As("*", "code")
		}
		problem.ProblemType = name
	}

	effectiveTimeNode := Nget(node, "effectiveTime")
	t := decodeTime(effectiveTimeNode)
	problem.Date = t.Value

	// observationNode2 := Nget(observationNode, "entryRelationship", "observation")
	// if observationNode2 != nil {
	//   problem.Status = Nget(observationNode2, "value").As("*", "displayName")
	// }

	erNodes := node.SelectNodes("*", "entryRelationship")
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
	return problem
}
