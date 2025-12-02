package models

import (
	"gopkg.in/check.v1"
)

func (s *ModelsSuite) TestPostTeam(c *check.C) {
	u := User{Username: "Test user", Role: Role{Slug: "user"}}
	err := PutUser(&u)
	c.Assert(err, check.Equals, nil)

	t := TeamSummary{Name: "Test Team", Description: "Test Team"}
	t.Users = []UserSummary{{Id: u.Id, Role: Role{Slug: "viewer"}}}
	err = PostTeam(&t)
	c.Assert(err, check.Equals, nil)
	c.Assert(t.Name, check.Equals, "Test Team")
	c.Assert(t.Users[0].Role.Slug, check.Equals, "viewer")
}

func (s *ModelsSuite) TestPostTeamNoUsers(c *check.C) {
	t := TeamSummary{Name: "No User Team", Description: "Test Team"}
	t.Users = []UserSummary{}
	err := PostTeam(&t)
	c.Assert(err, check.Equals, ErrNoUsersSpecified)
}

func (s *ModelsSuite) TestGetTeamUsers(c *check.C) {
	u := User{Username: "Test user", Role: Role{Slug: "user"}}
	err := PutUser(&u)
	c.Assert(err, check.Equals, nil)

	t := TeamSummary{Id: 1, Name: "Test Team", Description: "Test Team"}
	t.Users = []UserSummary{{Id: u.Id, Role: Role{Slug: "viewer"}}}
	err = PostTeam(&t)
	c.Assert(err, check.Equals, nil)

	users, err := GetTeamUsers(t.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(users[0].Id, check.Equals, u.Id)
}

func (s *ModelsSuite) TestInsertTeamUsers(c *check.C) {
	// Create the first User
	u := User{Username: "Test user", Role: Role{Slug: "user"}, ApiKey: "123456"}
	err := PutUser(&u)
	c.Assert(err, check.Equals, nil)

	// Create the second User
	second_user := User{Username: "Test user2", Role: Role{Slug: "user"}, ApiKey: "654321"}
	err = PutUser(&second_user)
	c.Assert(err, check.Equals, nil)

	// Create the team
	t := TeamSummary{Id: 1, Name: "Test Team", Description: "Test Team"}
	t.Users = []UserSummary{{Id: u.Id, Role: Role{Slug: "viewer"}}}
	err = PostTeam(&t)
	c.Assert(err, check.Equals, nil)

	// Add the second user to the team
	t.Users = []UserSummary{{Id: u.Id, Role: Role{Slug: "viewer"}}, {Id: second_user.Id, Role: Role{Slug: "contributor"}}}
	err = PutTeam(&t)
	c.Assert(err, check.Equals, nil)

	users, err := GetTeamUsers(t.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(users[1].Id, check.Equals, second_user.Id)
}

func (s *ModelsSuite) TestUpdateTeam(c *check.C) {
	// Create the test User
	u := User{Username: "Test user", Role: Role{Slug: "user"}, ApiKey: "123456"}
	err := PutUser(&u)
	c.Assert(err, check.Equals, nil)

	// Create the team
	t := TeamSummary{Id: 1, Name: "Test Team", Description: "Test Team"}
	t.Users = []UserSummary{{Id: u.Id, Role: Role{Slug: "viewer"}}}
	err = PostTeam(&t)
	c.Assert(err, check.Equals, nil)

	// Update the team name
	t.Name = "Changed Team Name"
	err = PutTeam(&t)
	c.Assert(err, check.Equals, nil)

	// Check if updated
	team, err := GetTeam(t.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team.Name, check.Equals, t.Name)

	// Update the team description
	t.Description = "Changed Team Description"
	err = PutTeam(&t)
	c.Assert(err, check.Equals, nil)

	// Check if updated
	team, err = GetTeam(t.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team.Description, check.Equals, t.Description)
}

func (s *ModelsSuite) TestRelateItemsAndTeam(c *check.C) {
	// Create the needed items
	// Create the Group
	group := Group{Name: "Test Group"}
	group.Targets = []Target{
		{BaseRecipient: BaseRecipient{Email: "test1@example.com", FirstName: "First", LastName: "Example"}},
		{BaseRecipient: BaseRecipient{Email: "test2@example.com", FirstName: "Second", LastName: "Example"}},
	}
	group.UserId = 1
	c.Assert(PostGroup(&group), check.Equals, nil)

	// Create the template
	template := Template{Name: "Test Template"}
	template.Subject = "{{.RId}} - Subject"
	template.Text = "{{.RId}} - Text"
	template.HTML = "{{.RId}} - HTML"
	template.UserId = 1
	c.Assert(PostTemplate(&template), check.Equals, nil)

	// Create the landing page
	page := Page{Name: "Test Page"}
	page.HTML = "<html>Test</html>"
	page.UserId = 1
	c.Assert(PostPage(&page), check.Equals, nil)

	// Create the sending profile
	smtp := SMTP{Name: "Test Page"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	c.Assert(PostSMTP(&smtp), check.Equals, nil)

	// Create the scenario
	scenario := Scenario{UserId: 1, Name: "Test", Description: "Test"}
	scenario.Templates = append([]Template{}, template)
	scenario.Page = page
	scenario.URL = "http://localhost.localdomain"
	c.Assert(PostScenario(&scenario, 1), check.Equals, nil)

	// Create the Campaign
	campaign := Campaign{Name: "Test campaign"}
	campaign.UserId = 1
	campaign.Scenarios = append([]Scenario{}, scenario)
	campaign.SMTP = smtp
	campaign.Groups = []Group{group}

	c.Assert(PostCampaign(&campaign, campaign.UserId), check.Equals, nil)

	// Create the user that gets the item shared
	u := User{Username: "Test user", Role: Role{Slug: "user"}}
	err := PutUser(&u)
	c.Assert(err, check.Equals, nil)

	// Create the team
	t := TeamSummary{Id: 1, Name: "Test Team", Description: "Test Team"}
	t.Users = []UserSummary{{Id: u.Id, Role: Role{Slug: "viewer"}}}
	err = PutTeam(&t)
	c.Assert(err, check.Equals, nil)

	// Get the Team to assure creation
	test, err := GetTeam(t.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(test.Name, check.Equals, "Test Team")
	c.Assert(test.Users[0].Id, check.Equals, u.Id)

	// Get the Page data as owner of the item
	page, err = GetPage(page.Id, 1)
	c.Assert(err, check.Equals, nil)
	// Use the data to relate the item to the team
	err = RelateItemAndTeam("pages", page.Item.Id, []Team{{Id: t.Id}}, 1)
	c.Assert(err, check.Equals, nil)
	// Check with the GetItemTeams Function if the item got related correctly
	team, err := GetItemTeams(page.Item.Id, "pages", u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team[0].Id, check.Equals, t.Id)

	// Get the Page data as the test user.
	check_page, err := GetPage(page.Id, u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(check_page.Teams[0].Name, check.Equals, t.Name)

	// Get the template data as owner of the item
	template, err = GetTemplate(template.Id, 1)
	c.Assert(err, check.Equals, nil)
	// Use the data to relate the item to the team
	err = RelateItemAndTeam("templates", template.Item.Id, []Team{{Id: t.Id}}, 1)
	c.Assert(err, check.Equals, nil)
	// Check with the GetItemTeams Function if the item got related correctly
	team, err = GetItemTeams(template.Item.Id, "templates", u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team[0].Id, check.Equals, t.Id)

	// Get the template data as the test user.
	check_template, err := GetTemplate(template.Id, u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(check_template.Teams[0].Name, check.Equals, t.Name)

	// Get the scenario data as owner of the item
	scenario, err = GetScenario(scenario.Id, 1)
	c.Assert(err, check.Equals, nil)
	// Use the data to relate the item to the team
	err = RelateItemAndTeam("scenarios", scenario.Item.Id, []Team{{Id: t.Id}}, 1)
	c.Assert(err, check.Equals, nil)
	// Check with the GetItemTeams Function if the item got related correctly
	team, err = GetItemTeams(scenario.Item.Id, "scenarios", u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team[0].Id, check.Equals, t.Id)

	// Get the scenario data as the test user.
	check_scenario, err := GetScenario(scenario.Id, u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(check_scenario.Teams[0].Name, check.Equals, t.Name)

	// Get the smtp profile data as owner of the item
	smtp, err = GetSMTP(smtp.Id, 1)
	c.Assert(err, check.Equals, nil)
	// Use the data to relate the item to the team
	err = RelateItemAndTeam("smtp", smtp.Item.Id, []Team{{Id: t.Id}}, 1)
	c.Assert(err, check.Equals, nil)
	// Check with the GetItemTeams Function if the item got related correctly
	team, err = GetItemTeams(smtp.Item.Id, "smtp", u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team[0].Id, check.Equals, t.Id)

	// Get the smtp profile data as the test user.
	check_smtp, err := GetSMTP(smtp.Id, u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(check_smtp.Teams[0].Name, check.Equals, t.Name)

	// Get the group data as owner of the item
	group, err = GetGroup(group.Id, 1)
	c.Assert(err, check.Equals, nil)
	// Use the data to relate the item to the team
	err = RelateItemAndTeam("groups", group.Item.Id, []Team{{Id: t.Id}}, 1)
	c.Assert(err, check.Equals, nil)
	// Check with the GetItemTeams Function if the item got related correctly
	team, err = GetItemTeams(group.Item.Id, "groups", u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team[0].Id, check.Equals, t.Id)

	// Get the group data as the test user.
	check_group, err := GetGroup(group.Id, u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(check_group.Teams[0].Name, check.Equals, t.Name)

	// Get the campaign data as owner of the item
	campaign, err = GetCampaign(campaign.Id, 1)
	c.Assert(err, check.Equals, nil)
	// Use the data to relate the item to the team
	err = RelateItemAndTeam("campaigns", campaign.Item.Id, []Team{{Id: t.Id}}, 1)
	c.Assert(err, check.Equals, nil)
	// Check with the GetItemTeams Function if the item got related correctly
	team, err = GetItemTeams(campaign.Item.Id, "campaigns", u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(team[0].Id, check.Equals, t.Id)

	// Get the campaign data as the test user.
	check_campaign, err := GetCampaign(campaign.Id, u.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(check_campaign.Teams[0].Name, check.Equals, t.Name)
}
