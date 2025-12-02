package models

import (
	"gopkg.in/check.v1"
	"testing"
)

func setupScenarioDependencies(b *testing.B) {
	// Add a template
	template := Template{Name: "Test Template"}
	template.Subject = "{{.RId}} - Subject"
	template.Text = "{{.RId}} - Text"
	template.HTML = "{{.RId}} - HTML"
	template.UserId = 1
	err := PostTemplate(&template)
	if err != nil {
		b.Fatalf("error posting template: %v", err)
	}

	// Add a landing page
	p := Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	err = PostPage(&p)
	if err != nil {
		b.Fatalf("error posting page: %v", err)
	}
}

func setupScenario(b *testing.B) Scenario {
	setupScenarioDependencies(b)
	scenario := Scenario{Name: "Test scenario"}
	scenario.UserId = 1
	scenario.Templates = append(scenario.Templates, Template{Id: 1})
	scenario.Page = Page{Id: 1}
	scenario.URL = "http://localhost.localdomain"
	PostScenario(&scenario, 1)
	return scenario
}

func (s *ModelsSuite) TestPostScenario(c *check.C) {
	sc := s.createScenarioDependencies(c)
	c.Assert(PostScenario(&sc, 1), check.Equals, nil)
}

func (s *ModelsSuite) TestGetScenarioTemplates(c *check.C) {
	sc := s.createScenarioDependencies(c)
	c.Assert(PostScenario(&sc, 1), check.Equals, nil)

	t, err := GetScenarioTemplates(sc.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(t[0].Name, check.Equals, sc.Templates[0].Name)
}

func (s *ModelsSuite) TestGetScenario(c *check.C) {
	sc := s.createScenarioDependencies(c)
	c.Assert(PostScenario(&sc, 1), check.Equals, nil)

	scenario, err := GetScenario(sc.Id, sc.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(scenario.Name, check.Equals, sc.Name)
}

func (s *ModelsSuite) TestScenarioValidation(c *check.C) {
	t := Template{Name: "Test Template"}
	t.Subject = "{{.RId}} - Subject"
	t.Text = "{{.RId}} - Text"
	t.HTML = "{{.RId}} - HTML"
	t.UserId = 1
	PostTemplate(&t)

	// Add a landing page
	p := Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	PostPage(&p)

	sc := Scenario{}

	// Validate that a name is required
	err := sc.Validate()
	c.Assert(err, check.Equals, ErrScenarioNameNotSpecified)
	sc.Name = "Test Scenario"

	// Validate that a Template is required
	err = sc.Validate()
	c.Assert(err, check.Equals, ErrTemplateNotSpecified)
	sc.Templates = append(sc.Templates, t)

	// Validate that a Page is required
	err = sc.Validate()
	c.Assert(err, check.Equals, ErrPageNotSpecified)
	sc.Page = p

	// Validate that a URL is required
	err = sc.Validate()
	c.Assert(err, check.Equals, ErrURLNotSpecified)
	sc.URL = "http://localhost.localdomain"

	err = sc.Validate()
	c.Assert(err, check.Equals, nil)
}
