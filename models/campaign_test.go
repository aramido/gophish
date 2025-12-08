package models

import (
	"fmt"
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

func assertSlotsWithinWindow(
	c *check.C,
	slots []time.Time,
	loc *time.Location,
	launch, sendBy time.Time,
	startHour, endHour int,
) {
	startDate := time.Date(launch.Year(), launch.Month(), launch.Day(), 0, 0, 0, 0, loc)
	endDate := time.Date(sendBy.Year(), sendBy.Month(), sendBy.Day(), 23, 59, 59, 0, loc)

	for _, ts := range slots {
		local := ts.In(loc)

		// No weekends.
		weekday := local.Weekday()
		c.Assert(weekday == time.Saturday || weekday == time.Sunday, check.Equals, false)

		// Within the date range [launch, sendBy].
		c.Assert(!local.Before(startDate), check.Equals, true)
		c.Assert(!local.After(endDate), check.Equals, true)

		// Within working hours [startHour, endHour] local time.
		hour := local.Hour()
		c.Assert(hour >= startHour && hour <= endHour, check.Equals, true)
	}
}

func (s *ModelsSuite) TestGenerateSendDate(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// Test that if no launch date is provided, the campaign's creation date
	// is used.
	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(campaign.LaunchDate, check.Equals, campaign.CreatedDate)

	// For comparing the dates, we need to fetch the campaign again. This is
	// to solve an issue where the campaign object right now has time down to
	// the microsecond, while in MySQL it's rounded down to the second.
	campaign, _ = GetCampaign(campaign.Id, campaign.UserId)

	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	for _, m := range ms {
		c.Assert(m.SendDate, check.Equals, campaign.CreatedDate)
	}

	// Test that if no send date is provided, all the emails are sent at the
	// campaign's launch date
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	campaign, _ = GetCampaign(campaign.Id, campaign.UserId)

	ms, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	for _, m := range ms {
		c.Assert(m.SendDate, check.Equals, campaign.LaunchDate)
	}

	// Finally, test that if a send date is provided, the emails are staggered
	// correctly.
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	campaign.SendByDate = campaign.LaunchDate.Add(2 * time.Minute)
	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	campaign, _ = GetCampaign(campaign.Id, campaign.UserId)

	_, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	// ADD CHECK FOR TIME IN THE FUTURE
}

func (s *ModelsSuite) TestGenerateTimeSlots(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// Test that if no launch date is provided, the campaign's creation date
	// is used.
	scenario := s.createScenarioDependencies(c)
	c.Assert(PostScenario(&scenario, 1), check.Equals, nil)
	campaign.Scenarios = append(campaign.Scenarios, scenario)

	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	for _, m := range ms {
		c.Assert(m.SendDate, check.Equals, campaign.LaunchDate)
	}

	// Finally, test that if a send date is provided, the emails are staggered
	// correctly.

	date_str := "2024-01-01T9:00:00.000Z"
	date_obj, err := time.Parse(time.RFC3339, date_str)
	c.Assert(err, check.Equals, nil)

	campaign.LaunchDate = date_obj
	campaign.SendByDate = campaign.LaunchDate.Add(9 * 24 * time.Hour)

	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	resultMap := make(map[string]bool)
	recipientList := []Target{}
	for _, g := range campaign.Groups {
		// Insert a result for each target in the group
		for _, t := range g.Targets {
			//Remove duplicate results - we should only send emails to unique email addresses.
			if _, ok := resultMap[t.Email]; ok {
				continue
			}
			resultMap[t.Email] = true
			recipientList = append(recipientList, t)
		}
	}

	ms, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, len(campaign.Scenarios)*len(recipientList))

	// ---------------------------------------------------------------------------------

	campaign.Id = campaign.Id + 1
	campaign.LaunchDate = date_obj
	campaign.SendByDate = campaign.LaunchDate.Add(3 * 24 * time.Hour)

	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	ms, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, len(campaign.Scenarios)*len(recipientList))

	// Check if Timeslots are in the expected Window
	loc := time.UTC
	var slots []time.Time
	for _, m := range ms {
		slots = append(slots, m.SendDate)
	}
	assertSlotsWithinWindow(c, slots, loc, campaign.LaunchDate, campaign.SendByDate, 9, 17)
}

func (s *ModelsSuite) TestCampaignDateValidation(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// If both are zero, then the campaign should start immediately with no
	// send by date
	err := campaign.Validate()
	c.Assert(err, check.Equals, nil)

	// If the launch date is specified, then the send date is optional
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	err = campaign.Validate()
	c.Assert(err, check.Equals, nil)

	// If the send date is greater than the launch date, then there's no
	//problem
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	campaign.SendByDate = campaign.LaunchDate.Add(1 * time.Minute)
	err = campaign.Validate()
	c.Assert(err, check.Equals, nil)

	// If the send date is less than the launch date, then there's an issue
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	campaign.SendByDate = campaign.LaunchDate.Add(-1 * time.Minute)
	err = campaign.Validate()
	c.Assert(err, check.Equals, ErrInvalidSendByDate)
}

func (s *ModelsSuite) TestCampaignValidation(c *check.C) {
	campaign := s.createCampaignDependencies(c)

	campaign.Name = ""
	err := campaign.Validate()
	c.Assert(err, check.Equals, ErrCampaignNameNotSpecified)

	campaign.Name = "Test"

	groups := campaign.Groups

	campaign.Groups = make([]Group, 0)
	err = campaign.Validate()
	c.Assert(err, check.Equals, ErrGroupNotSpecified)

	campaign.Groups = append(campaign.Groups, groups...)

	scenarios := campaign.Scenarios

	campaign.Scenarios = make([]Scenario, 0)
	err = campaign.Validate()
	c.Assert(err, check.Equals, ErrScenarioNotFound)

	campaign.Scenarios = append(campaign.Scenarios, scenarios...)

	smtp := campaign.SMTP

	campaign.SMTP = SMTP{}
	err = campaign.Validate()
	c.Assert(err, check.Equals, ErrSMTPNotSpecified)

	campaign.SMTP = smtp

	// If the launch date is specified, then the send date is optional
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	err = campaign.Validate()
	c.Assert(err, check.Equals, nil)

	// If the send date is greater than the launch date, then there's no
	//problem
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	campaign.SendByDate = campaign.LaunchDate.Add(1 * time.Minute)
	err = campaign.Validate()
	c.Assert(err, check.Equals, nil)

	// If the send date is less than the launch date, then there's an issue
	campaign = s.createCampaignDependencies(c)
	campaign.LaunchDate = time.Now().UTC()
	campaign.SendByDate = campaign.LaunchDate.Add(-1 * time.Minute)
	err = campaign.Validate()
	c.Assert(err, check.Equals, ErrInvalidSendByDate)
}

func (s *ModelsSuite) TestCampaignStats(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// If both are zero, then the campaign should start immediately with no
	// send by date
	err := campaign.Validate()
	c.Assert(err, check.Equals, nil)

	_, err = getCampaignStats(campaign.Id)
	c.Assert(err, check.Equals, nil)
}

func (s *ModelsSuite) TestCampaignSummaries(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// If both are zero, then the campaign should start immediately with no
	// send by date
	err := campaign.Validate()
	c.Assert(err, check.Equals, nil)

	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	css, err := GetCampaignSummaries(campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(css.Total, check.Equals, int64(1))

	cs, err := GetCampaignSummary(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(cs.Name, check.Equals, campaign.Name)
}

func (s *ModelsSuite) TestCampaignMailContext(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// If both are zero, then the campaign should start immediately with no
	// send by date
	err := campaign.Validate()
	c.Assert(err, check.Equals, nil)

	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	mc, err := GetCampaignMailContext(campaign.Id, campaign.UserId, campaign.Scenarios[0].Templates[0].Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(mc.Template.Id, check.Equals, campaign.Scenarios[0].Templates[0].Id)
}

func (s *ModelsSuite) TestCampaignResults(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	// If both are zero, then the campaign should start immediately with no
	// send by date
	err := campaign.Validate()
	c.Assert(err, check.Equals, nil)

	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	cr, err := GetCampaignResults(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(cr.Id, check.Equals, campaign.Id)
}

func (s *ModelsSuite) TestLaunchCampaignMaillogStatus(c *check.C) {
	// For the first test, ensure that campaigns created with the zero date
	// (and therefore are set to launch immediately) have maillogs that are
	// locked to prevent race conditions.
	campaign := s.createCampaign(c)
	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)

	for _, m := range ms {
		c.Assert(m.Processing, check.Equals, true)
	}

	// Next, verify that campaigns scheduled in the future do not lock the
	// maillogs so that they can be picked up by the background worker.
	campaign = s.createCampaignDependencies(c)
	campaign.Name = "New Campaign"
	campaign.LaunchDate = time.Now().Add(1 * time.Hour)
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.Equals, nil)
	ms, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)

	for _, m := range ms {
		c.Assert(m.Processing, check.Equals, false)
	}
}

func (s *ModelsSuite) TestDeleteCampaignAlsoDeletesMailLogs(c *check.C) {
	campaign := s.createCampaign(c)
	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, len(campaign.Results))

	err = DeleteCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)

	ms, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, 0)
}

func (s *ModelsSuite) TestCompleteCampaignAlsoDeletesMailLogs(c *check.C) {
	campaign := s.createCampaign(c)
	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, len(campaign.Results))

	err = CompleteCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	ms, err = GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, 0)
}

func (s *ModelsSuite) TestCampaignGetResults(c *check.C) {
	campaign := s.createCampaign(c)
	got, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(campaign.Results), check.Equals, len(got.Results))
}

func (s *ModelsSuite) TestCampaignResolveLoc(c *check.C) {
	var campaign Campaign

	// Empty Location -> fallback to UTC
	loc := campaign.resolveLoc()
	c.Assert(loc, check.NotNil)
	c.Assert(loc.String(), check.Equals, "UTC")

	// Invalid Location -> fallback to UTC
	campaign.Location = "Not/AZone"
	loc = campaign.resolveLoc()
	c.Assert(loc, check.NotNil)
	c.Assert(loc.String(), check.Equals, "UTC")

	// Valid IANA timezone -> returned as-is
	campaign.Location = "Europe/Berlin"
	loc = campaign.resolveLoc()
	c.Assert(loc, check.NotNil)
	c.Assert(loc.String(), check.Equals, "Europe/Berlin")
}

func (s *ModelsSuite) TestPostCampaignAppliesTimezoneToDates(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	campaign.Location = "Europe/Berlin"

	loc, err := time.LoadLocation("Europe/Berlin")
	c.Assert(err, check.IsNil)

	// fixed date so the test is stable
	launch := time.Date(2030, time.January, 2, 9, 0, 0, 0, loc)
	sendBy := launch.AddDate(0, 0, 2)

	campaign.LaunchDate = launch
	campaign.SendByDate = sendBy

	err = PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.IsNil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.IsNil)

	c.Assert(stored.Location, check.Equals, "Europe/Berlin")

	// Launch / SendBy should represent the same instants in time
	c.Assert(stored.LaunchDate.Equal(launch), check.Equals, true)
	if !stored.SendByDate.IsZero() {
		c.Assert(stored.SendByDate.Equal(sendBy), check.Equals, true)
	}

	startLocal := stored.StartTime.In(loc)
	endLocal := stored.EndTime.In(loc)

	c.Assert(startLocal.Year(), check.Equals, launch.Year())
	c.Assert(startLocal.Month(), check.Equals, launch.Month())
	c.Assert(startLocal.Day(), check.Equals, launch.Day())

	c.Assert(endLocal.Year(), check.Equals, launch.Year())
	c.Assert(endLocal.Month(), check.Equals, launch.Month())
	c.Assert(endLocal.Day(), check.Equals, launch.Day())
}

func (s *ModelsSuite) TestGenerateTimeSlotsRespectsLocation(c *check.C) {
	loc, err := time.LoadLocation("Europe/Berlin")
	c.Assert(err, check.Equals, nil)

	launch := time.Date(2030, time.January, 1, 0, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 5, 0, 0, 0, 0, loc)

	campaign := Campaign{
		LaunchDate: launch,
		SendByDate: sendBy,
		StartTime: time.Date(launch.Year(), launch.Month(), launch.Day(), 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(sendBy.Year(), sendBy.Month(), sendBy.Day(), 17, 0, 0, 0, time.UTC),
		Location:  "Europe/Berlin",
	}

	totalRecipients := 20
	slots := campaign.generateTimeSlots(totalRecipients)
	
	assertSlotsWithinWindow(c, slots, loc, launch, sendBy, 9, 17)
}

func setupCampaignDependencies(b *testing.B, size int) {
	group := Group{Name: "Test Group"}
	// Create a large group of 5000 members
	for i := 0; i < size; i++ {
		group.Targets = append(group.Targets, Target{BaseRecipient: BaseRecipient{Email: fmt.Sprintf("test%d@example.com", i), FirstName: "User", LastName: fmt.Sprintf("%d", i)}})
	}
	group.UserId = 1
	err := PostGroup(&group)
	if err != nil {
		b.Fatalf("error posting group: %v", err)
	}

	// Add a template
	template := Template{Name: "Test Template"}
	template.Subject = "{{.RId}} - Subject"
	template.Text = "{{.RId}} - Text"
	template.HTML = "{{.RId}} - HTML"
	template.UserId = 1
	err = PostTemplate(&template)
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

	// Add a scenario
	s := Scenario{UserId: 1, Name: "Test Scenario", Description: "Test"}
	s.URL = "http://localhost.localdomain"
	s.UserId = 1
	s.Templates = append(s.Templates, template)
	s.Page = p
	err = PostScenario(&s, s.UserId)
	if err != nil {
		b.Fatalf("error posting scenario: %v", err)
	}

	// Add a sending profile
	smtp := SMTP{Name: "Test Page"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	err = PostSMTP(&smtp)
	if err != nil {
		b.Fatalf("error posting smtp: %v", err)
	}
}

// setupCampaign sets up the campaign dependencies as well as posting the
// actual campaign
func setupCampaign(b *testing.B, size int) Campaign {
	setupCampaignDependencies(b, size)
	campaign := Campaign{Name: "Test campaign"}
	campaign.UserId = 1
	campaign.Scenarios = append(campaign.Scenarios, Scenario{Id: 1})
	campaign.SMTP = SMTP{Id: 1, Name: "Test Page"}
	campaign.Groups = []Group{{Id: 1, Name: "Test Group"}}
	err := PostCampaign(&campaign, 1)
	if err != nil {
		b.Fatalf("error posting campaign: %v", err)
	}
	return campaign
}

func BenchmarkCampaign100(b *testing.B) {
	setupBenchmark(b)
	setupCampaignDependencies(b, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		campaign := Campaign{Name: "Test campaign"}
		campaign.UserId = 1
		campaign.Scenarios = append(campaign.Scenarios, Scenario{Id: 1})
		campaign.SMTP = SMTP{Id: 1}
		campaign.Groups = []Group{{Id: 1}}

		b.StartTimer()
		err := PostCampaign(&campaign, 1)
		if err != nil {
			b.Fatalf("error posting campaign: %v", err)
		}
		b.StopTimer()
		db.Delete(Result{})
		db.Delete(MailLog{})
		db.Delete(Campaign{})
	}
	tearDownBenchmark(b)
}

func BenchmarkCampaign1000(b *testing.B) {
	setupBenchmark(b)
	setupCampaignDependencies(b, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		campaign := Campaign{Name: "Test campaign"}
		campaign.UserId = 1
		campaign.Scenarios = append(campaign.Scenarios, Scenario{Id: 1})
		campaign.SMTP = SMTP{Id: 1}
		campaign.Groups = []Group{{Id: 1}}

		b.StartTimer()
		err := PostCampaign(&campaign, 1)
		if err != nil {
			b.Fatalf("error posting campaign: %v", err)
		}
		b.StopTimer()
		db.Delete(Result{})
		db.Delete(MailLog{})
		db.Delete(Campaign{})
	}
	tearDownBenchmark(b)
}

func BenchmarkCampaign10000(b *testing.B) {
	setupBenchmark(b)
	setupCampaignDependencies(b, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		campaign := Campaign{Name: "Test campaign"}
		campaign.UserId = 1
		campaign.Scenarios = append(campaign.Scenarios, Scenario{Id: 1})
		campaign.SMTP = SMTP{Id: 1}
		campaign.Groups = []Group{{Id: 1}}

		b.StartTimer()
		err := PostCampaign(&campaign, 1)
		if err != nil {
			b.Fatalf("error posting campaign: %v", err)
		}
		b.StopTimer()
		db.Delete(Result{})
		db.Delete(MailLog{})
		db.Delete(Campaign{})
	}
	tearDownBenchmark(b)
}

func BenchmarkGetCampaign100(b *testing.B) {
	setupBenchmark(b)
	campaign := setupCampaign(b, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetCampaign(campaign.Id, campaign.UserId)
		if err != nil {
			b.Fatalf("error getting campaign: %v", err)
		}
	}
	tearDownBenchmark(b)
}

func BenchmarkGetCampaign1000(b *testing.B) {
	setupBenchmark(b)
	campaign := setupCampaign(b, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetCampaign(campaign.Id, campaign.UserId)
		if err != nil {
			b.Fatalf("error getting campaign: %v", err)
		}
	}
	tearDownBenchmark(b)
}

func BenchmarkGetCampaign5000(b *testing.B) {
	setupBenchmark(b)
	campaign := setupCampaign(b, 5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetCampaign(campaign.Id, campaign.UserId)
		if err != nil {
			b.Fatalf("error getting campaign: %v", err)
		}
	}
	tearDownBenchmark(b)
}

func BenchmarkGetCampaign10000(b *testing.B) {
	setupBenchmark(b)
	campaign := setupCampaign(b, 10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetCampaign(campaign.Id, campaign.UserId)
		if err != nil {
			b.Fatalf("error getting campaign: %v", err)
		}
	}
	tearDownBenchmark(b)
}
