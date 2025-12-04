package models

import (
	"errors"
	"math"
	"math/rand"
	"net/url"
	"sort"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/webhook"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Campaign is a struct representing a created campaign
type Campaign struct {
	Id            int64         `json:"id"`
	UserId        int64         `json:"user_id"`
	Name          string        `json:"name" sql:"not null"`
	CreatedDate   time.Time     `json:"created_date"`
	LaunchDate    time.Time     `json:"launch_date"`
	SendByDate    time.Time     `json:"send_by_date"`
	CompletedDate time.Time     `json:"completed_date"`
	Scenarios     []Scenario    `json:"scenarios" gorm:"many2many:campaign_scenarios;"`
	Status        string        `json:"status"`
	Results       []Result      `json:"results,omitempty"`
	Groups        []Group       `json:"groups,omitempty" gorm:"-"`
	Events        []Event       `json:"timeline,omitempty"`
	SMTPId        int64         `json:"-"`
	SMTP          SMTP          `json:"smtp"`
	Item          Item          `json:"-" gorm:"ForeignKey:item_type_id"`
	Teams         []TeamSummary `json:"teams" gorm:"-"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Location  	  string		`json:"location"`
}

// CampaignResults is a struct representing the results from a campaign
type CampaignResults struct {
	Id      int64    `json:"id"`
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Results []Result `json:"results,omitempty"`
	Events  []Event  `json:"timeline,omitempty"`
}

// CampaignSummaries is a struct representing the overview of campaigns
type CampaignSummaries struct {
	Total     int64             `json:"total"`
	Campaigns []CampaignSummary `json:"campaigns"`
}

// CampaignSummary is a struct representing the overview of a single camaign
type CampaignSummary struct {
	Id            int64         `json:"id"`
	UserId        int64         `json:"user_id"`
	CreatedDate   time.Time     `json:"created_date"`
	LaunchDate    time.Time     `json:"launch_date"`
	SendByDate    time.Time     `json:"send_by_date"`
	CompletedDate time.Time     `json:"completed_date"`
	Status        string        `json:"status"`
	Name          string        `json:"name"`
	Stats         CampaignStats `json:"stats"`
	Teams         []TeamSummary `json:"teams" gorm:"-"`
}

// CampaignStats is a struct representing the statistics for a single campaign
type CampaignStats struct {
	Total         int64 `json:"total"`
	EmailsSent    int64 `json:"sent"`
	OpenedEmail   int64 `json:"opened"`
	ClickedLink   int64 `json:"clicked"`
	SubmittedData int64 `json:"submitted_data"`
	EmailReported int64 `json:"email_reported"`
	Error         int64 `json:"error"`
}

// CampaignTeams is used for a many-to-many relationship between 1..* Campaigns and 1..* Teams
type CampaignTeams struct {
	CampaignId int64 `json:"-"`
	TeamId     int64 `json:"-"`
}

// Added CampaignMailContext to allow Campaign specific Mail Context to be different from Scenario Context
type CampaignMailContext struct {
	Id         int64    `json:"id"`
	TemplateId int64    `json:"-"`
	Template   Template `json:"template"`
	SMTPId     int64    `json:"-"`
	SMTP       SMTP     `json:"smtp" gorm:"-"`
	Status     string   `json:"status"`
	Results    []Result `json:"results,omitempty"`
	Groups     []Group  `json:"groups,omitempty"`
	Events     []Event  `json:"timeline,omitempty"`
	URL        string   `json:"url"`
	UserId     int64    `json:"-"`
}

// Event contains the fields for an event
// that occurs during the campaign
type Event struct {
	Id         int64     `json:"-"`
	CampaignId int64     `json:"campaign_id"`
	Email      string    `json:"email"`
	Time       time.Time `json:"time"`
	Message    string    `json:"message"`
	Details    string    `json:"details"`
}

// EventDetails is a struct that wraps common attributes we want to store
// in an event
type EventDetails struct {
	Payload url.Values        `json:"payload"`
	Browser map[string]string `json:"browser"`
}

// EventError is a struct that wraps an error that occurs when sending an
// email to a recipient
type EventError struct {
	Error string `json:"error"`
}

type Assignment struct {
	Scenario Scenario
	Template Template
}
type Recipient struct {
	Recipient   Target
	Assignments []Assignment
}

// ErrCampaignNameNotSpecified indicates there was no template given by the user
var ErrCampaignNameNotSpecified = errors.New("Campaign name not specified")

// ErrGroupNotSpecified indicates there was no template given by the user
var ErrGroupNotSpecified = errors.New("No groups specified")

// ErrTemplateNotSpecified indicates there was no template given by the user
var ErrTemplateNotSpecified = errors.New("No email template specified")

// ErrPageNotSpecified indicates a landing page was not provided for the campaign
var ErrPageNotSpecified = errors.New("No landing page specified")

// ErrSMTPNotSpecified indicates a sending profile was not provided for the campaign
var ErrSMTPNotSpecified = errors.New("No sending profile specified")

// ErrTemplateNotFound indicates the template specified does not exist in the database
var ErrTemplateNotFound = errors.New("Template not found")

// ErrScenarioNotFound indicates the scenario specified does not exist in the database
var ErrScenarioNotFound = errors.New("Scenario not found")

// ErrGroupNotFound indicates a group specified by the user does not exist in the database
var ErrGroupNotFound = errors.New("Group not found")

// ErrPageNotFound indicates a page specified by the user does not exist in the database
var ErrPageNotFound = errors.New("Page not found")

// ErrSMTPNotFound indicates a sending profile specified by the user does not exist in the database
var ErrSMTPNotFound = errors.New("Sending profile not found")

// ErrInvalidSendByDate indicates that the user specified a send by date that occurs before the
// launch date
var ErrInvalidSendByDate = errors.New("The launch date must be before the \"send emails by\" date")

// ErrNoWorkingDays indicates that there are no working days in the given timeframe
var ErrNoWorkingDays = errors.New("There are no working days in the given timeframe")

// RecipientParameter is the URL parameter that points to the result ID for a recipient.
const RecipientParameter = "rid"

// Validate checks to make sure there are no invalid fields in a submitted campaign
func (c *Campaign) Validate() error {
	switch {
	case c.Name == "":
		return ErrCampaignNameNotSpecified
	case len(c.Groups) == 0:
		return ErrGroupNotSpecified
	case len(c.Scenarios) == 0:
		return ErrScenarioNotFound
	case c.SMTP.Id == 0:
		return ErrSMTPNotSpecified
	case !c.SendByDate.IsZero() && !c.LaunchDate.IsZero() && c.SendByDate.Before(c.LaunchDate):
		return ErrInvalidSendByDate
	}
	return nil
}

// UpdateStatus changes the campaign status appropriately
func (c *Campaign) UpdateStatus(s string) error {
	// This could be made simpler, but I think there's a bug in gorm
	return db.Table("campaigns").Where("id=?", c.Id).Update("status", s).Error
}

// UpdateStatus changes the campaign status appropriately
func (c *CampaignMailContext) UpdateStatus(s string) error {
	// This could be made simpler, but I think there's a bug in gorm
	return db.Table("campaigns").Where("id=?", c.Id).Update("status", s).Error
}

// AddEvent creates a new campaign event in the database
func AddEvent(e *Event, campaignID int64) error {
	e.CampaignId = campaignID
	e.Time = time.Now().UTC()

	whs, err := GetActiveWebhooks()
	if err == nil {
		whEndPoints := []webhook.EndPoint{}
		for _, wh := range whs {
			whEndPoints = append(whEndPoints, webhook.EndPoint{
				URL:    wh.URL,
				Secret: wh.Secret,
			})
		}
		webhook.SendAll(whEndPoints, e)
	} else {
		log.Errorf("error getting active webhooks: %v", err)
	}

	return db.Save(e).Error
}

// getDetails retrieves the related attributes of the campaign
// from the database. If the Events and the Results are not available,
// an error is returned. Otherwise, the attribute name is set to [Deleted],
// indicating the user deleted the attribute (template, smtp, etc.)
func (c *Campaign) getDetails() error {

	err := db.Model(c).Related(&c.Results).Error
	if err != nil {
		log.Warnf("%s: results not found for campaign", err)
		return err
	}
	err = db.Model(c).Related(&c.Events).Error
	if err != nil {
		log.Warnf("%s: events not found for campaign", err)
		return err
	}
	err = db.Preload("Scenarios.Templates").Preload("Scenarios.Templates.Attachments").Preload("Scenarios.Page").First(&c, c.Id).Error
	if err != nil {
		log.Warnf("%s: scenarios not found for campaign", err)
		return err
	}
	err = db.Table("smtp").Where("id=?", c.SMTPId).Find(&c.SMTP).Error
	if err != nil {
		// Check if the SMTP was deleted
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.SMTP = SMTP{Name: "[Deleted]"}
		log.Warnf("%s: sending profile not found for campaign", err)
	}
	err = db.Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}

	return nil
}

// getBaseURL returns the Campaign's configured URL.
// This is used to implement the TemplateContext interface.
func (c *CampaignMailContext) getBaseURL() string {
	return c.URL
}

// getFromAddress returns the Campaign's configured SMTP "From" address.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getFromAddress() string {
	return c.SMTP.FromAddress
}

func (c *CampaignMailContext) getFromAddress() string {
	return c.SMTP.FromAddress
}

func (c *Campaign) assignSendDate(idx int, timeSlots []time.Time) time.Time {
	if c.SendByDate.IsZero() {
		return c.LaunchDate
	}
	// Using the idx of the recipient we can assign the timeSlot
	return timeSlots[idx]
}

// helper function for time location
func (c *Campaign) resolveLoc() *time.Location {
    if c.Location != "" {
        if l, err := time.LoadLocation(c.Location); err == nil {
            return l
        }
    }
    return time.UTC
}

// Generates timeSlots. When timeSlots are generated the startHour and endHour defined at Campaign creation is ignored, but instead only whole days between 9 to 5 are regarded.
func (c *Campaign) generateTimeSlots(totalRecipients int) []time.Time {
	location := c.resolveLoc()
	// For future proofing i added the startDate and endDate. Should they change the parameter c.LaunchDate or c.SendByDate we only need to edit it here.
	startDate := time.Date(c.LaunchDate.Year(), c.LaunchDate.Month(), c.LaunchDate.Day(), 0, 0, 0, 0, location)
	endDate := time.Date(c.SendByDate.Year(), c.SendByDate.Month(), c.SendByDate.Day(), 0, 0, 0, 0, location)
	// Calculate the duration of each day in which we can send Emails
	durationPerDay := c.EndTime.Sub(c.StartTime)

	weekendDays := 0
	var weekdaysList []time.Time
	var timeSlots []time.Time

	// Check which days in the given Timeframe are Workdays
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		weekday := date.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			weekendDays++
		} else {
			weekdaysList = append(weekdaysList, date)
		}
	}

	// If the given Timeframe has no workdays the empty timeSlots slice gets returned.
	if len(weekdaysList) < 1 {
		return timeSlots
	}

	// Calculate the number of recipients per day
	recipientsPerDate := totalRecipients / len(weekdaysList)
	remainingRecipients := totalRecipients % len(weekdaysList)

	// Create a dictionary to store the recipients count for each date
	recipientsCountByDate := make(map[time.Time]int)

	// Assign the even number of recipients to each date
	for _, date := range weekdaysList {
		recipientsCountByDate[date] = recipientsPerDate
	}

	currentWeekday := 0
	// Assign the remaining recipients to the first few dates
	if remainingRecipients >= 1 {
		dateOffset := int(math.Max(1, float64(len(weekdaysList)/remainingRecipients)))

		for i := 0; i < remainingRecipients; i++ {
			recipientsCountByDate[weekdaysList[currentWeekday]] += 1
			currentWeekday += dateOffset
		}
	}

	// Create the timeSlots for all days / all recipients
	for date, count := range recipientsCountByDate {
		// Set the first time for the day
		currentTime := date.Add(time.Duration(c.StartTime.Hour()) * time.Hour)
		endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), c.EndTime.Hour(), 0, 0, 0, location)

		// Calculate the offset between each recipients in seconds
		offset := float64(durationPerDay) / float64(count) // offset as float in h
		timeBetweenRecipients := time.Duration(offset)

		// Create the time slots for this day
		for i := 0; i < count; i++ {
			// Create a jitter so that Emails get send more random
			randomDuration := time.Duration(rand.Int63n(int64(timeBetweenRecipients)))

			// Add jitter to currentTime
			mailTime := currentTime.Add(randomDuration)
			if mailTime.After(endTime) {
				// make mailTime maximal endTime
				mailTime = endTime
			}

			// Iterate to the next time
			currentTime = currentTime.Add(timeBetweenRecipients)
			// Append the timeSlot to the timeSlots list
			timeSlots = append(timeSlots, mailTime.UTC())
		}
	}
	// Sort timeSlots
	sort.Slice(timeSlots, func(i, j int) bool {
		return timeSlots[i].Before(timeSlots[j])
	})
	// Return the timeSlots so that they can be used
	return timeSlots
}

// getCampaignStats returns a CampaignStats object for the campaign with the given campaign ID.
// It also backfills numbers as appropriate with a running total, so that the values are aggregated.
func getCampaignStats(cid int64) (CampaignStats, error) {
	s := CampaignStats{}
	query := db.Table("results").Where("campaign_id = ?", cid)
	err := query.Count(&s.Total).Error
	if err != nil {
		return s, err
	}
	err = query.Where("status=?", EventDataSubmit).Count(&s.SubmittedData).Error
	if err != nil {
		return s, err
	}
	err = query.Where("status=?", EventClicked).Count(&s.ClickedLink).Error
	if err != nil {
		return s, err
	}
	err = query.Where("reported=?", true).Count(&s.EmailReported).Error
	if err != nil {
		return s, err
	}
	// Every submitted data event implies they clicked the link
	s.ClickedLink += s.SubmittedData
	err = query.Where("status=?", EventOpened).Count(&s.OpenedEmail).Error
	if err != nil {
		return s, err
	}
	// Every clicked link event implies they opened the email
	s.OpenedEmail += s.ClickedLink
	err = query.Where("status=?", EventSent).Count(&s.EmailsSent).Error
	if err != nil {
		return s, err
	}
	// Every opened email event implies the email was sent
	s.EmailsSent += s.OpenedEmail
	err = query.Where("status=?", Error).Count(&s.Error).Error
	return s, err
}

// GetCampaigns returns the campaigns owned by the given user and accessible by the user's teams.
func GetCampaigns(uid int64) ([]Campaign, error) {

	// Fetch campaigns accessible by the user's teams
	campaigns := []Campaign{}

	err := db.Preload("Item", "item_type = ?", "campaigns").Preload("Item.Teams.Users.Teams.Role").Preload("Scenarios").Find(&campaigns).Error
	if err != nil {
		log.Error(err)
		return campaigns, err
	}

	cs := []Campaign{}
	// Check if user is allowed to see the campaign. By being the owner or related to it through a team
	for _, campaign := range campaigns {
		if campaign.UserId == uid || checkForValue(uid, campaign.Item.Teams) {
			ts := []TeamSummary{}
			for _, team := range campaign.Item.Teams {
				ts = append(ts, convertTeamSummary(team))
			}
			campaign.Teams = ts
			cs = append(cs, campaign)
		}
	}

	// Fetch details for each campaign
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	return cs, nil
}

func GetCampaignSummaries(uid int64) (CampaignSummaries, error) {
	overview := CampaignSummaries{}

	campaigns, err := GetCampaigns(uid)
	if err != nil {
		log.Error(err)
		return overview, err
	}
	cs := []CampaignSummary{}

	for i := range campaigns {
		c := CampaignSummary{
			Id:            campaigns[i].Id,
			UserId:        campaigns[i].UserId,
			CreatedDate:   campaigns[i].CreatedDate,
			LaunchDate:    campaigns[i].LaunchDate,
			SendByDate:    campaigns[i].SendByDate,
			CompletedDate: campaigns[i].CompletedDate,
			Status:        campaigns[i].Status,
			Name:          campaigns[i].Name,
			Teams:         campaigns[i].Teams,
		}
		cs = append(cs, c)
	}

	for i := range cs {
		s, err := getCampaignStats(cs[i].Id)
		if err != nil {
			log.Error(err)
			return overview, err
		}
		cs[i].Stats = s
	}
	overview.Total = int64(len(cs))
	overview.Campaigns = cs
	return overview, nil
}

// GetCampaignSummary gets the summary object for a campaign specified by the campaign ID
func GetCampaignSummary(id int64, uid int64) (CampaignSummary, error) {
	// Fetch campaigns accessible by the user's teams
	cs := CampaignSummary{}

	campaign, err := GetCampaign(id, uid)
	if err != nil {
		log.Error(err)
		return cs, err
	}

	cs = CampaignSummary{
		Id:            campaign.Id,
		UserId:        campaign.UserId,
		CreatedDate:   campaign.CreatedDate,
		LaunchDate:    campaign.LaunchDate,
		SendByDate:    campaign.SendByDate,
		CompletedDate: campaign.CompletedDate,
		Status:        campaign.Status,
		Name:          campaign.Name,
		Teams:         campaign.Teams,
	}

	s, err := getCampaignStats(cs.Id)
	if err != nil {
		log.Error(err)
		return cs, err
	}
	cs.Stats = s
	return cs, nil
}

func GetCampaignMailContext(id int64, uid int64, temid int64) (CampaignMailContext, error) {
	c := Campaign{}
	cm := CampaignMailContext{}
	// fetch the Mail Context Information
	err := db.Preload("Item", "item_type = ?", "campaigns").
		Preload("Item.Teams.Users.Teams.Role").
		Preload("Scenarios.Templates.Attachments").
		Preload("SMTP").
		Find(&c, "campaigns.id =?", id).
		Error
	if err != nil {
		return cm, err
	}

	// Check if user is allowed to see the campaign. By being the owner or related to it through a team
	if !(c.UserId == uid || checkForValue(uid, c.Item.Teams)) {
		return cm, ErrPageNotFound
	}

	for _, scenario := range c.Scenarios {
		for _, template := range scenario.Templates {
			if template.Id == temid {
				cm.Template = template
			}
		}
		cm.URL = scenario.URL
	}
	cm.SMTP = c.SMTP
	return cm, nil
}

func GetCampaignContext(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := db.Where("id = ?", id).Where("user_id = ?", uid).Find(&c).Error
	if err != nil {
		return c, err
	}
	err = db.Table("smtp").Where("id=?", c.SMTPId).Find(&c.SMTP).Error
	if err != nil {
		return c, err
	}
	err = db.Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return c, err
	}
	return c, nil
}

// GetCampaign returns the campaign, if it exists, specified by the given id and user_id.
func GetCampaign(id int64, uid int64) (Campaign, error) {
	c := Campaign{}

	// Fetch Campaign Information
	err := db.Preload("Item", "item_type = ?", "campaigns").Preload("Item.Teams.Users.Teams.Role").Find(&c, "campaigns.id =?", id).Error
	if err != nil {
		log.Error(err)
		return c, err
	}

	// Check if User can access the information
	if !(c.UserId == uid || checkForValue(uid, c.Item.Teams)) {
		return c, ErrPageNotFound
	}

	// Convert Team into the correct format
	ts := []TeamSummary{}
	for _, team := range c.Item.Teams {
		ts = append(ts, convertTeamSummary(team))
	}
	c.Teams = ts

	// Get Details related to the campaign
	err = c.getDetails()
	if err != nil {
		log.Error(err)
		return c, err
	}

	return c, nil
}

// GetCampaignResults returns just the campaign results for the given campaign
func GetCampaignResults(id int64, uid int64) (CampaignResults, error) {
	cr := CampaignResults{}
	campaign, err := GetCampaign(id, uid)
	if err != nil {
		log.Error(err)
		return cr, err
	}

	cr = CampaignResults{
		Id:      campaign.Id,
		Name:    campaign.Name,
		Events:  campaign.Events,
		Results: campaign.Results,
	}

	return cr, err
}

// GetQueuedCampaigns returns the campaigns that are queued up for this given minute
func GetQueuedCampaigns(t time.Time) ([]Campaign, error) {
	cs := []Campaign{}
	err := db.Where("launch_date <= ?", t).
		Where("status = ?", CampaignQueued).Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	log.Infof("Found %d Campaigns to run\n", len(cs))
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// PostCampaign inserts a campaign and all associated records into the database.
func PostCampaign(c *Campaign, uid int64) error {
	err := c.Validate()
	if err != nil {
		return err
	}
	// Fill in the details
	c.UserId = uid
	c.CreatedDate = time.Now().UTC()
	c.CompletedDate = time.Time{}
	c.Status = CampaignQueued
	location := c.resolveLoc()
	if c.LaunchDate.IsZero() {
		c.LaunchDate = c.CreatedDate
	} else {
		c.LaunchDate = c.LaunchDate.In(location)
	}
	if !c.SendByDate.IsZero() {
		c.SendByDate = c.SendByDate.In(location)
	}
	if c.LaunchDate.Before(c.CreatedDate) || c.LaunchDate.Equal(c.CreatedDate) {
		c.Status = CampaignInProgress
	}
	if c.StartTime.IsZero() {
		c.StartTime = time.Date(c.LaunchDate.Year(), c.LaunchDate.Month(), c.LaunchDate.Day(), 10, 0, 0, 0, location).UTC()
	} else {
		c.StartTime = c.StartTime.In(location)
	}
	if c.EndTime.IsZero() {
		c.EndTime = time.Date(c.LaunchDate.Year(), c.LaunchDate.Month(), c.LaunchDate.Day(), 18, 0, 0, 0, location).UTC()
	} else {
		c.EndTime = c.EndTime.In(location)
	}
	// Check to make sure all the groups already exist

	groups := []Group{}
	// Get all the Groups assigned to the campaign
	for _, group := range c.Groups {
		g, err := GetGroup(group.Id, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"group": group.Id,
			}).Error("Group does not exist")
			return ErrGroupNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		groups = append(groups, g)
	}
	c.Groups = append(c.Groups, groups...)
	// Check to make sure the Scenario exists
	scenarios := []Scenario{}
	for _, scenario := range c.Scenarios {
		s, err := GetScenario(scenario.Id, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"scenario": scenario.Id,
			}).Error("Scenario does not exist")
			return ErrScenarioNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		scenarios = append(scenarios, s)
	}
	c.Scenarios = append([]Scenario{}, scenarios...)
	// Get sending Profile
	s, err := GetSMTP(c.SMTP.Id, uid)
	log.Info(s)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"smtp": c.SMTP.Id,
		}).Error("Sending profile does not exist")
		return ErrSMTPNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}

	c.SMTP = s
	c.SMTPId = s.Id
	// Insert into the DB
	tx := db.Begin()
	err = tx.Save(&c).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Create(&Item{ItemType: "campaigns", ItemTypeID: c.Id}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = AddEvent(&Event{Message: "Campaign Created"}, c.Id)
	if err != nil {
		log.Error(err)
		return err
	}
	// Insert all the results
	resultMap := make(map[string]bool)
	recipientList := []Target{}
	for _, g := range c.Groups {
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
	// Create a list of all (recipient, scenario, template) combinations
	totalTimeSlotsNeeded := 0
	recipients := make(map[Target]*Recipient)
	for _, recipient := range recipientList {
		var assignments []Assignment
		for _, scenario := range c.Scenarios {
			for _, template := range scenario.Templates {
				assignments = append(assignments, Assignment{Scenario: scenario, Template: template})
				totalTimeSlotsNeeded += 1
			}
		}
		recipients[recipient] = &Recipient{
			Recipient:   recipient,
			Assignments: assignments,
		}
	}

	// Generate the timeSlots
	timeSlots := c.generateTimeSlots(totalTimeSlotsNeeded)
	timeSlotsIndex := 0
	// Check to make sure enough timeSlots were generated
	// If timeSlots are smaller than the number of recipients, an error gets thrown
	if (len(timeSlots) < totalTimeSlotsNeeded) && !(c.SendByDate.IsZero() || c.SendByDate.Equal(c.LaunchDate)) {
		log.WithFields(logrus.Fields{
			"timeSlots": len(timeSlots),
		}).Error("There are no working days in the given timeframe")
		return ErrNoWorkingDays
	}
	for i := 0; i < totalTimeSlotsNeeded/len(recipientList); i++ {
		for _, recipient := range recipients {
			// Take a random scenario from the Recipients list of scenarios
			index := rand.Intn(len(recipient.Assignments))
			assignment := recipient.Assignments[index]
			// Remove the scenario for that Recipient
			recipient.Assignments = append(recipient.Assignments[:index], recipient.Assignments[index+1:]...)
			tx = db.Begin()
			// Insert a result for each target in the group
			sendDate := c.assignSendDate(timeSlotsIndex, timeSlots)
			r := &Result{
				BaseRecipient: BaseRecipient{
					Email:     recipient.Recipient.Email,
					Position:  recipient.Recipient.Position,
					FirstName: recipient.Recipient.FirstName,
					LastName:  recipient.Recipient.LastName,
				},
				Status:       StatusScheduled,
				CampaignId:   c.Id,
				UserId:       c.UserId,
				SendDate:     sendDate,
				Reported:     false,
				ModifiedDate: c.CreatedDate,
				ScenarioId:   assignment.Scenario.Id,
				TemplateId:   assignment.Template.Id,
			}
			err = r.GenerateId(tx)
			if err != nil {
				log.Error(err)
				tx.Rollback()
				return err
			}
			processing := false
			if r.SendDate.Before(c.CreatedDate) || r.SendDate.Equal(c.CreatedDate) {
				r.Status = StatusSending
				processing = true
			}
			err = tx.Save(r).Error

			if err != nil {
				log.WithFields(logrus.Fields{
					"email": recipient.Recipient.Email,
				}).Errorf("error creating result: %v", err)
				tx.Rollback()
				return err
			}
			c.Results = append(c.Results, *r)
			log.WithFields(logrus.Fields{
				"email":     r.Email,
				"send_date": sendDate,
			}).Debug("creating maillog")
			m := &MailLog{
				UserId:     c.UserId,
				CampaignId: c.Id,
				RId:        r.RId,
				SendDate:   sendDate,
				Processing: processing,
				ScenarioId: assignment.Scenario.Id,
				TemplateId: assignment.Template.Id,
			}
			err = tx.Save(m).Error
			if err != nil {
				log.WithFields(logrus.Fields{
					"email": recipient.Recipient.Email,
				}).Errorf("error creating maillog entry: %v", err)
				tx.Rollback()
				return err
			}
			timeSlotsIndex++
			tx.Commit()
		}
	}

	return nil
}

// DeleteCampaign deletes the specified campaign
func DeleteCampaign(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Deleting campaign")
	tx := db.Begin()
	// Delete all the campaign results
	err := tx.Where("campaign_id=?", id).Delete(&Result{}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Delete all the campaign events
	err = tx.Where("campaign_id=?", id).Delete(&Event{}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Delete all the campaign mailogs
	err = tx.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Delete all the campaign item
	item := Item{}
	err = tx.Where("item_type = ? AND item_type_id = ?", "campaigns", id).Delete(&item).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Delete all the relations between the campaign_item and teams
	err = tx.Where("item_id = ?", item.Id).Delete(&ItemTeams{}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Delete the campaign
	err = tx.Delete(&Campaign{Id: id}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Commit the changes
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return err
}

// CompleteCampaign effectively "ends" a campaign.
// Any future emails clicked will return a simple "404" page.
func CompleteCampaign(id int64, uid int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Marking campaign as complete")
	c, err := GetCampaign(id, uid)
	if err != nil {
		return err
	}
	// Delete any maillogs still set to be sent out, preventing future emails
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Don't overwrite original completed time
	if c.Status == CampaignComplete {
		return nil
	}
	// Mark the campaign as complete
	c.CompletedDate = time.Now().UTC()
	c.Status = CampaignComplete

	err = db.Model(&Campaign{}).Where("id=?", id).
		Select([]string{"completed_date", "status"}).UpdateColumns(&c).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetSMTPCampaignCount performs a one-to-many select to get all the Campaigns that use this smtp profile
func GetSMTPCampaignCount(sid int64) int64 {
	var count int64
	db.Model(&Campaign{}).Where("smtp_id = ?", sid).Count(&count)
	return count
}
