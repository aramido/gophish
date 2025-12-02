package models

import (
	"errors"
	"regexp"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Scenario is a struct representing a scenario
type Scenario struct {
	Id           int64         `json:"id"`
	UserId       int64         `json:"user_id"`
	Name         string        `json:"name" sql:"not null"`
	Description  string        `json:"description`
	CreatedDate  time.Time     `json:"created_date"`
	ModifiedDate time.Time     `json:"modified_date"`
	Templates    []Template    `json:"templates" gorm:"many2many:scenario_templates;"`
	PageId       int64         `json:"-"`
	Page         Page          `json:"page"`
	URL          string        `json:"url"`
	Item         Item          `json:"-" gorm:"ForeignKey:item_type_id"`
	Teams        []TeamSummary `json:"teams" gorm:"-"`
}

type CampaignScenarios struct {
	CampaignId int64 `json:"-"`
	ScenarioId int64 `json:"-"`
}

type ScenarioTemplates struct {
	ScenarioId int64 `json:"-"`
	TemplateId int64 `json:"-"`
}

// ErrCampaignNameNotSpecified indicates there was no template given by the user
var ErrScenarioNameNotSpecified = errors.New("Scenario name not specified")

// ErrURLNotValid indicates there was no valid url given by the user
var ErrURLNotValid = errors.New("URL not valid")

// ErrURLNotValid indicates there was no valid url given by the user
var ErrScenarioAttached = errors.New("Scenario is used by at least one running campaign")

// getBaseURL returns the Campaign's configured URL.
// This is used to implement the TemplateContext interface.
func (s *Scenario) getBaseURL() string {
	return s.URL
}

// Validate checks to make sure there are no invalid fields in a submitted campaign
func (s *Scenario) Validate() error {
	temp, _ := regexp.MatchString(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,12}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`, s.URL)
	switch {
	case s.Name == "":
		return ErrScenarioNameNotSpecified
	case len(s.Templates) < 1:
		return ErrTemplateNotSpecified
	case s.Page.Id == 0:
		return ErrPageNotSpecified
	case s.URL == "":
		return ErrURLNotSpecified
	case temp == false:
		return ErrURLNotValid
	}
	return nil
}
func (s *Scenario) getDetails() error {
	templates := []Template{}
	err := db.Model(s).Association("Templates").Find(&templates).Error
	if err != nil {
		return err
	}
	s.Templates = templates
	err = db.Table("pages").Where("id=?", s.PageId).Find(&s.Page).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		s.Page = Page{Name: "[Deleted]"}
		log.Warnf("%s: page not found for campaign", err)
	}
	return nil
}

// GetScenarios returns the scenarios owned by the given user and accessible by the user's teams.
func GetScenarios(uid int64) ([]Scenario, error) {

	// Fetch scenarios accessible by the user's teams
	scenarios := []Scenario{}

	err := db.Preload("Item", "item_type = ?", "scenarios").Preload("Item.Teams.Users.Teams.Role").Find(&scenarios).Error
	if err != nil {
		log.Error(err)
		return scenarios, err
	}

	ss := []Scenario{}

	for _, scenario := range scenarios {
		if scenario.UserId == uid || checkForValue(uid, scenario.Item.Teams) {
			ts := []TeamSummary{}
			for _, team := range scenario.Item.Teams {
				ts = append(ts, convertTeamSummary(team))
			}
			scenario.Teams = ts
			ss = append(ss, scenario)
		}
	}

	// Fetch details for each scenario
	for i := range ss {
		err = ss[i].getDetails()
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	return ss, nil
}

// GetScenario returns the scenario, if it exists, specified by the given id and user_id.
func GetScenario(id int64, uid int64) (Scenario, error) {
	s := Scenario{}
	err := db.Preload("Item", "item_type = ?", "scenarios").Preload("Item.Teams.Users").Preload("Page").Preload("Templates").Find(&s, "id=?", id).Error
	if err != nil {
		log.Error(err)
		return s, err
	}
	// Check if User can view the scenario
	if !(s.UserId == uid || checkForValue(uid, s.Item.Teams)) {
		return s, ErrScenarioNotFound
	}
	// Create TeamSummarys to attach them to the scenario
	ts := []TeamSummary{}
	for _, team := range s.Item.Teams {
		ts = append(ts, convertTeamSummary(team))
	}
	s.Teams = ts

	// Fetch details for scenario
	err = s.getDetails()
	if err != nil {
		log.Error(err)
		return s, err
	}

	return s, nil
}

// PostScenario inserts a scenario and all associated records into the database.
func PostScenario(sc *Scenario, uid int64) error {
	// Validate the scenario
	err := sc.Validate()
	if err != nil {
		return err
	}
	// Fill in the details
	sc.UserId = uid
	sc.ModifiedDate = time.Now().UTC()
	sc.CreatedDate = time.Now().UTC()

	// Check to make sure the template exists
	templates := []Template{}
	for _, template := range sc.Templates {
		t, err := GetTemplate(template.Id, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"template": template.Id,
			}).Error("Template does not exist")
			return ErrTemplateNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		templates = append(templates, t)
	}
	sc.Templates = append([]Template{}, templates...)
	// Check to make sure the page exists
	p, err := GetPage(sc.Page.Id, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"page": sc.Page.Name,
		}).Error("Page does not exist")
		return ErrPageNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	sc.Page = p
	sc.PageId = p.Id

	tx := db.Begin()
	// Create the scenario
	err = tx.Create(&sc).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Create(&Item{ItemType: "scenarios", ItemTypeID: sc.Id}).Error
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
	return nil
}

// PutScenario edits an existing scenario in the database.
// Per the PUT Method RFC, it presumes all data for a scenario is provided.
func PutScenario(s *Scenario) error {
	err := s.Validate()
	if err != nil {
		return err
	}

	// Check to make sure the template exists
	scenarioTemplates := []Template{}
	for _, template := range s.Templates {
		t, err := GetTemplate(template.Id, s.UserId)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"template": template.Id,
			}).Error("Template does not exist")
			return ErrTemplateNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		scenarioTemplates = append(scenarioTemplates, t)
	}
	s.Templates = scenarioTemplates
	// Check to make sure the page exists
	p, err := GetPage(s.Page.Id, s.UserId)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"page": s.Page.Name,
		}).Error("Page does not exist")
		return ErrPageNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	s.Page = p
	s.PageId = p.Id

	// Fetch group's existing targets from database.
	ts, err := GetScenarioTemplates(s.Id)
	if err != nil {
		log.WithFields(logrus.Fields{
			"scenario_id": s.Id,
		}).Error("Error getting templates from scenario")
		return err
	}
	// Preload the caches
	cacheNew := make(map[string]int64, len(s.Templates))
	for _, t := range s.Templates {
		cacheNew[t.Name] = t.Id
	}

	cacheExisting := make(map[string]int64, len(ts))
	for _, t := range ts {
		cacheExisting[t.Name] = t.Id
	}

	tx := db.Begin()

	// Check existing templates, removing any that are no longer in the scenario.
	for _, t := range ts {
		if _, ok := cacheNew[t.Name]; ok {
			continue
		}

		// If the template does not exist in the scenario any longer, we remove it
		err := tx.Where("scenario_id=? and template_id=?", s.Id, t.Id).Delete(&ScenarioTemplates{}).Error
		if err != nil {
			tx.Rollback()
			log.WithFields(logrus.Fields{
				"name": t.Name,
			}).Error("Error deleting template for scenario")
		}
	}

	// Add any template that are not in the database yet.
	for _, nt := range s.Templates {
		if _, ok := cacheExisting[nt.Name]; ok {
			continue
		}

		// Otherwise, add template
		err := tx.Save(&ScenarioTemplates{ScenarioId: s.Id, TemplateId: nt.Id}).Error
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return err
		}
	}

	err = tx.Save(s).Error
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// DeleteScenario deletes the specified scenario
func DeleteScenario(id int64) error {
	log.WithFields(logrus.Fields{
		"scenario_id": id,
	}).Info("Deleting scenario")

	tx := db.Begin()

	item := Item{}
	err := tx.Where("item_type = ? AND item_type_id = ?", "scenarios", id).Delete(&item).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Where("item_id = ?", item.Id).Delete(&ItemTeams{}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	// Delete the scenario
	err = tx.Delete(&Scenario{Id: id}).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return err
}

// GetScenarioTemplates performs a many-to-many select to get all the Templates for a Scenario
func GetScenarioTemplates(sid int64) ([]Template, error) {
	t := []Template{}
	err := db.Table("templates").Select("templates.id, templates.user_id, templates.name, templates.subject, templates.text, templates.html, templates.modified_date, templates.envelope_sender").Joins("left join scenario_templates st ON templates.id = st.template_id").Where("st.scenario_id=?", sid).Scan(&t).Error
	return t, err
}

// GetTemplateScenariosCount performs a many-to-many select to get all the Scenarios for a Template
func GetTemplateScenariosCount(tid int64) int64 {
	var count int64
	db.Model(&ScenarioTemplates{}).Where("template_id = ?", tid).Count(&count)
	return count
}

// GetPageScenariosCount performs a one-to-many select to get all the Scenarios for a Page
func GetPageScenariosCount(pid int64) int64 {
	var count int64
	db.Model(&Scenario{}).Where("page_id = ?", pid).Count(&count)
	return count
}

// GetScenarioCampaignsCount performs a many-to-many select to get all the Campaigns for a Scenario
func GetScenarioCampaignsCount(sid int64) int64 {
	var count int64
	db.Model(&CampaignScenarios{}).Where("scenario_id = ?", sid).Count(&count)
	return count
}
