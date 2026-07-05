package models

import (
	"fmt"
	"sort"
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

func assertSendDatesWithinRecipientTimeline(
	c *check.C,
	slots []time.Time,
	loc *time.Location,
	timelineStart, sendBy time.Time,
	startHour, endHour int,
) {
	for _, ts := range slots {
		local := ts.In(loc)
		weekday := local.Weekday()
		c.Assert(weekday == time.Saturday || weekday == time.Sunday, check.Equals, false)
		c.Assert(!ts.Before(timelineStart), check.Equals, true)
		c.Assert(!ts.After(sendBy), check.Equals, true)

		windowStart := time.Date(local.Year(), local.Month(), local.Day(), startHour, 0, 0, 0, loc)
		windowEnd := time.Date(local.Year(), local.Month(), local.Day(), endHour, 0, 0, 0, loc)
		c.Assert(!local.Before(windowStart), check.Equals, true)
		c.Assert(!local.After(windowEnd), check.Equals, true)
	}
}

func assignmentKey(scenarioID, templateID int64) string {
	return fmt.Sprintf("%d:%d", scenarioID, templateID)
}

func recipientAssignmentKey(email string, scenarioID, templateID int64) string {
	return fmt.Sprintf("%s:%d:%d", email, scenarioID, templateID)
}

func campaignAssignmentSet(campaign Campaign) map[string]int {
	assignments := make(map[string]int)
	for _, scenario := range campaign.Scenarios {
		for _, template := range scenario.Templates {
			assignments[assignmentKey(scenario.Id, template.Id)]++
		}
	}
	return assignments
}

func campaignAssignmentCount(campaign Campaign) int {
	count := 0
	for _, scenario := range campaign.Scenarios {
		count += len(scenario.Templates)
	}
	return count
}

func resultsByEmail(results []Result) map[string][]Result {
	grouped := make(map[string][]Result)
	for _, result := range results {
		grouped[result.Email] = append(grouped[result.Email], result)
	}
	return grouped
}

func assertRecipientHasEveryAssignment(c *check.C, results []Result, expected map[string]int) {
	c.Assert(len(results), check.Equals, len(expected))
	got := make(map[string]int)
	for _, result := range results {
		got[assignmentKey(result.ScenarioId, result.TemplateId)]++
	}
	c.Assert(got, check.DeepEquals, expected)
}

func assertMailLogsMirrorResults(c *check.C, campaign Campaign, results []Result) {
	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(ms), check.Equals, len(results))

	resultsByRID := make(map[string]Result)
	for _, result := range results {
		resultsByRID[result.RId] = result
	}
	for _, m := range ms {
		result, ok := resultsByRID[m.RId]
		c.Assert(ok, check.Equals, true)
		c.Assert(m.CampaignId, check.Equals, result.CampaignId)
		c.Assert(m.UserId, check.Equals, result.UserId)
		c.Assert(m.ScenarioId, check.Equals, result.ScenarioId)
		c.Assert(m.TemplateId, check.Equals, result.TemplateId)
		c.Assert(m.SendDate.Equal(result.SendDate), check.Equals, true)
	}
}

func defaultSchedulingTargets() []Target {
	return []Target{
		{BaseRecipient: BaseRecipient{Email: "schedule1@example.com", FirstName: "First", LastName: "Recipient"}},
		{BaseRecipient: BaseRecipient{Email: "schedule2@example.com", FirstName: "Second", LastName: "Recipient"}},
	}
}

func (s *ModelsSuite) createSchedulingCampaign(c *check.C, templateCounts []int, targets []Target) Campaign {
	if len(targets) == 0 {
		targets = defaultSchedulingTargets()
	}

	group := Group{Name: "Scheduling Test Group", UserId: 1, Targets: targets}
	c.Assert(PostGroup(&group), check.Equals, nil)

	smtp := SMTP{Name: "Scheduling SMTP", UserId: 1, Host: "example.com", FromAddress: "test@test.com"}
	c.Assert(PostSMTP(&smtp), check.Equals, nil)

	scenarios := make([]Scenario, 0, len(templateCounts))
	for scenarioIndex, templateCount := range templateCounts {
		page := Page{Name: fmt.Sprintf("Scheduling Page %d", scenarioIndex), HTML: "<html>Test</html>", UserId: 1}
		c.Assert(PostPage(&page), check.Equals, nil)

		scenario := Scenario{
			UserId:      1,
			Name:        fmt.Sprintf("Scheduling Scenario %d", scenarioIndex),
			Description: "Scheduling test",
			Page:        page,
			URL:         fmt.Sprintf("http://localhost.localdomain/%d", scenarioIndex),
		}
		for templateIndex := 0; templateIndex < templateCount; templateIndex++ {
			template := Template{
				Name:    fmt.Sprintf("Scheduling Template %d-%d", scenarioIndex, templateIndex),
				Subject: "{{.RId}} - Subject",
				Text:    "{{.RId}} - Text",
				HTML:    "{{.RId}} - HTML",
				UserId:  1,
			}
			c.Assert(PostTemplate(&template), check.Equals, nil)
			scenario.Templates = append(scenario.Templates, template)
		}
		c.Assert(PostScenario(&scenario, 1), check.Equals, nil)
		scenarios = append(scenarios, scenario)
	}

	return Campaign{
		Name:      "Scheduling Test Campaign",
		UserId:    1,
		Scenarios: scenarios,
		SMTP:      smtp,
		Groups:    []Group{group},
	}
}

func setCampaignWorkingWindow(campaign *Campaign, loc *time.Location, launch, sendBy time.Time) {
	campaign.Location = loc.String()
	campaign.LaunchDate = launch
	campaign.SendByDate = sendBy
	campaign.StartTime = time.Date(launch.Year(), launch.Month(), launch.Day(), 9, 0, 0, 0, loc)
	campaign.EndTime = time.Date(launch.Year(), launch.Month(), launch.Day(), 17, 0, 0, 0, loc)
}

func (s *ModelsSuite) TestCampaignSchedulesEveryTemplateForEachRecipient(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{2, 1, 3}, defaultSchedulingTargets())
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 9, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 11, 17, 0, 0, 0, loc)
	setCampaignWorkingWindow(&campaign, loc, launch, sendBy)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	expected := campaignAssignmentSet(stored)
	c.Assert(len(expected), check.Equals, 6)
	grouped := resultsByEmail(stored.Results)
	c.Assert(len(grouped), check.Equals, len(defaultSchedulingTargets()))
	for _, recipient := range defaultSchedulingTargets() {
		assertRecipientHasEveryAssignment(c, grouped[recipient.Email], expected)
	}
	assertMailLogsMirrorResults(c, stored, stored.Results)
}

func (s *ModelsSuite) TestCampaignSchedulesEachRecipientOnChronologicalWorkingTimeline(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{2, 1, 3}, defaultSchedulingTargets())
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 9, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 11, 17, 0, 0, 0, loc)
	setCampaignWorkingWindow(&campaign, loc, launch, sendBy)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	for email, recipientResults := range resultsByEmail(stored.Results) {
		var slots []time.Time
		for _, result := range recipientResults {
			slots = append(slots, result.SendDate)
		}
		sort.Slice(slots, func(i, j int) bool {
			return slots[i].Before(slots[j])
		})
		for i := 1; i < len(slots); i++ {
			c.Assert(!slots[i].Before(slots[i-1]), check.Equals, true, check.Commentf("recipient %s", email))
		}
		assertSendDatesWithinRecipientTimeline(c, slots, loc, launch, sendBy, 9, 17)
	}
}

func (s *ModelsSuite) TestScheduleRecipientDoesNotModifyExistingRecipientSchedules(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{2, 1}, defaultSchedulingTargets())
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 9, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 10, 17, 0, 0, 0, loc)
	setCampaignWorkingWindow(&campaign, loc, launch, sendBy)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	before := make(map[string]time.Time)
	for _, result := range stored.Results {
		before[recipientAssignmentKey(result.Email, result.ScenarioId, result.TemplateId)] = result.SendDate
	}

	newRecipient := Target{BaseRecipient: BaseRecipient{Email: "added@example.com", FirstName: "Added", LastName: "Recipient"}}
	tx := db.Begin()
	newResults, err := stored.scheduleRecipient(tx, newRecipient, launch.Add(2*time.Hour))
	c.Assert(err, check.Equals, nil)
	c.Assert(tx.Commit().Error, check.Equals, nil)
	c.Assert(len(newResults), check.Equals, campaignAssignmentCount(stored))

	after, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	for _, result := range after.Results {
		if result.Email == newRecipient.Email {
			continue
		}
		key := recipientAssignmentKey(result.Email, result.ScenarioId, result.TemplateId)
		original, ok := before[key]
		c.Assert(ok, check.Equals, true)
		c.Assert(result.SendDate.Equal(original), check.Equals, true)
	}
}

func (s *ModelsSuite) TestScheduleRecipientAddedMidCampaignUsesAddedAtTimeline(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{2, 1, 3}, []Target{
		{BaseRecipient: BaseRecipient{Email: "initial@example.com", FirstName: "Initial", LastName: "Recipient"}},
	})
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 9, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 11, 17, 0, 0, 0, loc)
	setCampaignWorkingWindow(&campaign, loc, launch, sendBy)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	addedAt := time.Date(2030, time.January, 8, 12, 30, 0, 0, loc)
	newRecipient := Target{BaseRecipient: BaseRecipient{Email: "midcampaign@example.com", FirstName: "Mid", LastName: "Campaign"}}
	tx := db.Begin()
	newResults, err := stored.scheduleRecipient(tx, newRecipient, addedAt)
	c.Assert(err, check.Equals, nil)
	c.Assert(tx.Commit().Error, check.Equals, nil)

	expected := campaignAssignmentSet(stored)
	assertRecipientHasEveryAssignment(c, newResults, expected)

	var slots []time.Time
	for _, result := range newResults {
		c.Assert(result.Email, check.Equals, newRecipient.Email)
		slots = append(slots, result.SendDate)
	}
	assertSendDatesWithinRecipientTimeline(c, slots, loc, addedAt, sendBy, 9, 17)
}

func (s *ModelsSuite) TestCampaignWithNoSendByDateSendsEveryAssignmentAtLaunch(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{2, 1}, defaultSchedulingTargets())
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 9, 0, 0, 0, loc)
	campaign.Location = loc.String()
	campaign.LaunchDate = launch
	campaign.StartTime = time.Date(launch.Year(), launch.Month(), launch.Day(), 9, 0, 0, 0, loc)
	campaign.EndTime = time.Date(launch.Year(), launch.Month(), launch.Day(), 17, 0, 0, 0, loc)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(stored.Results), check.Equals, campaignAssignmentCount(stored)*len(defaultSchedulingTargets()))
	for _, result := range stored.Results {
		c.Assert(result.SendDate.Equal(stored.LaunchDate), check.Equals, true)
	}
	assertMailLogsMirrorResults(c, stored, stored.Results)
}

func (s *ModelsSuite) TestCampaignReturnsErrorWhenRecipientHasNoWorkingTime(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{1}, []Target{
		{BaseRecipient: BaseRecipient{Email: "nowork@example.com", FirstName: "No", LastName: "Work"}},
	})
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 18, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 7, 20, 0, 0, 0, loc)
	setCampaignWorkingWindow(&campaign, loc, launch, sendBy)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, ErrNoWorkingDays)
}

func (s *ModelsSuite) TestCampaignSchedulesEveryTemplateInSingleScenario(c *check.C) {
	campaign := s.createSchedulingCampaign(c, []int{3}, []Target{
		{BaseRecipient: BaseRecipient{Email: "single@example.com", FirstName: "Single", LastName: "Scenario"}},
	})
	loc := time.UTC
	launch := time.Date(2030, time.January, 7, 9, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 9, 17, 0, 0, 0, loc)
	setCampaignWorkingWindow(&campaign, loc, launch, sendBy)

	err := PostCampaign(&campaign, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	stored, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.Equals, nil)

	expected := campaignAssignmentSet(stored)
	c.Assert(len(expected), check.Equals, 3)
	c.Assert(len(stored.Results), check.Equals, 3)
	assertRecipientHasEveryAssignment(c, stored.Results, expected)
	assertMailLogsMirrorResults(c, stored, stored.Results)
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
	campaign.LaunchDate = time.Date(2030, time.January, 7, 10, 0, 0, 0, time.UTC)
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
	c.Assert(err, check.IsNil)

	launch := time.Date(2030, time.January, 1, 0, 0, 0, 0, loc)
	sendBy := time.Date(2030, time.January, 5, 0, 0, 0, 0, loc)

	campaign := Campaign{
		LaunchDate: launch,
		SendByDate: sendBy,
		StartTime:  time.Date(launch.Year(), launch.Month(), launch.Day(), 9, 0, 0, 0, loc),
		EndTime:    time.Date(sendBy.Year(), sendBy.Month(), sendBy.Day(), 17, 0, 0, 0, loc),
		Location:   "Europe/Berlin",
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
