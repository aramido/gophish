package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ErrModifyingOnlyAdmin occurs when there is an attempt to modify the only
// user account with the Admin role in such a way that there will be no user
// accounts left in Gophish with that role.
var ErrModifyingOnlyAdmin = errors.New("Cannot remove the only administrator")
var ErrAttached = errors.New("Some of the templates of the users are still attached to scenarios or campaigns")

// User represents the user model for gophish.
type User struct {
	Id                     int64     `json:"id"`
	Username               string    `json:"username" sql:"not null;unique"`
	Hash                   string    `json:"-"`
	ApiKey                 string    `json:"api_key" sql:"not null;unique"`
	Role                   Role      `json:"role" gorm:"association_autoupdate:false;association_autocreate:false"`
	RoleID                 int64     `json:"-"`
	PasswordChangeRequired bool      `json:"password_change_required"`
	AccountLocked          bool      `json:"account_locked"`
	LastLogin              time.Time `json:"last_login"`
	Teams                  []Team    `json:"teams" gorm:"many2many:team_users;"`
	Items                  []Item
}

type UserSummary struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// GetUser returns the user that the given id corresponds to. If no user is found, an
// error is thrown.
func GetUser(id int64) (User, error) {
	u := User{}

	err := db.Preload("Role").Preload("Teams.Role").Preload("Teams.Users").Find(&u, id).Error
	return u, err
}

// GetUsers returns the users registered in Gophish
func GetUsers() ([]User, error) {
	us := []User{}

	err := db.Preload("Role").Preload("Teams.Role").Find(&us).Error

	return us, err
}

// GetUserByAPIKey returns the user that the given API Key corresponds to. If no user is found, an
// error is thrown.
func GetUserByAPIKey(key string) (User, error) {

	u := User{}

	err := db.Preload("Role").Preload("Teams.Role").Find(&u, "api_key = ?", key).Error

	return u, err
}

// GetUserByUsername returns the user that the given username corresponds to. If no user is found, an
// error is thrown.
func GetUserByUsername(username string) (User, error) {
	u := User{}
	err := db.Preload("Role").Preload("Teams.Role").Find(&u, "username = ?", username).Error

	return u, err
}

// PutUser updates the given user
func PutUser(u *User) error {
	// Is needed to prevent the password reset bug
	u.Teams = []Team{}
	err := db.Save(u).Error
	return err
}

type UserId struct {
	UserId int64 `json:"-" gorm:"column:user_id"`
}

// EnsureEnoughAdmins ensures that there is more than one user account in
// Gophish with the Admin role. This function is meant to be called before
// modifying a user account with the Admin role in a non-revokable way.
func EnsureEnoughAdmins() error {
	role, err := GetRoleBySlug(RoleAdmin)
	if err != nil {
		return err
	}
	var adminCount int
	err = db.Model(&User{}).Where("role_id=?", role.ID).Count(&adminCount).Error
	if err != nil {
		return err
	}
	if adminCount == 1 {
		return ErrModifyingOnlyAdmin
	}
	return nil
}

// DeleteUser deletes the given user. To ensure that there is always at least
// one user account with the Admin role, this function will refuse to delete
// the last Admin.
func DeleteUser(id int64) error {
	existing, err := GetUser(id)
	if err != nil {
		return err
	}
	// If the user is an admin, we need to verify that it's not the last one.
	if existing.Role.Slug == RoleAdmin {
		err = EnsureEnoughAdmins()
		if err != nil {
			return err
		}
	}
	campaigns, err := GetCampaigns(id)
	if err != nil {
		return err
	}
	// Delete the campaigns
	log.Infof("Deleting campaigns for user ID %d", id)
	for _, campaign := range campaigns {
		if campaign.UserId == id {
			err = DeleteCampaign(campaign.Id)
			if err != nil {
				return err
			}
		}
	}
	log.Infof("Deleting pages for user ID %d", id)
	// Delete the landing pages
	pages, err := GetPages(id)
	if err != nil {
		return err
	}
	for _, page := range pages {
		if page.UserId == id {
			if GetPageScenariosCount(page.Id) >= 1 {
				return ErrAttached
			}
			err = DeletePage(page.Id, id)
			if err != nil {
				return err
			}
		}
	}
	// Delete the templates
	log.Infof("Deleting templates for user ID %d", id)
	templates, err := GetTemplates(id)
	if err != nil {
		return err
	}
	for _, template := range templates {
		if template.UserId == id {
			if GetTemplateScenariosCount(template.Id) >= 1 {
				return ErrAttached
			}
			err = DeleteTemplate(template.Id, id)
			if err != nil {
				return err
			}
		}
	}
	// Delete the scenario
	log.Infof("Deleting scenarios for user ID %d", id)
	scenarios, err := GetScenarios(id)
	if err != nil {
		return err
	}
	for _, scenario := range scenarios {
		if scenario.UserId == id {
			if GetScenarioCampaignsCount(scenario.Id) >= 1 {
				return ErrAttached
			}
			err = DeleteScenario(scenario.Id)
			if err != nil {
				return err
			}
		}
	}
	// Delete the groups
	log.Infof("Deleting groups for user ID %d", id)
	groups, err := GetGroups(id)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.UserId == id {
			err = DeleteGroup(&group)
			if err != nil {
				return err
			}
		}
	}
	// Delete the sending profiles
	log.Infof("Deleting sending profiles for user ID %d", id)
	profiles, err := GetSMTPs(id)
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if profile.UserId == id {
			if GetSMTPCampaignCount(profile.Id) >= 1 {
				return ErrAttached
			}
			err = DeleteSMTP(profile.Id, id)
			if err != nil {
				return err
			}
		}
	}
	// Finally, delete the user
	err = db.Where("id=?", id).Delete(&User{}).Error
	return err
}

func (u *User) IsOwnerOfItem(iid int64, item string) (bool, error) {

	uid := UserId{}

	err := db.Table(item).
		Where("id = ? ", iid).
		Find(&uid).Error
	if err != nil {
		log.Error(err)
	}

	if uid.UserId == u.Id {
		return true, nil
	}

	return false, err
}

func UserTeams(uid int64) ([]Team, error) {

	user := User{}
	err := db.Preload("Teams").Find(&user, "id = ?", uid).Error
	if err != nil {
		log.Error(err)
	}

	return user.Teams, err
}
