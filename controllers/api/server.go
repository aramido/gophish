package api

import (
	"net/http"

	mid "github.com/gophish/gophish/middleware"
	"github.com/gophish/gophish/middleware/ratelimit"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/worker"
	"github.com/gorilla/mux"
)

// ServerOption is an option to apply to the API server.
type ServerOption func(*Server)

// Server represents the routes and functionality of the Gophish API.
// It's not a server in the traditional sense, in that it isn't started and
// stopped. Rather, it's meant to be used as an http.Handler in the
// AdminServer.
type Server struct {
	handler http.Handler
	worker  worker.Worker
	limiter *ratelimit.PostLimiter
}

// NewServer returns a new instance of the API handler with the provided
// options applied.
func NewServer(options ...ServerOption) *Server {
	defaultWorker, _ := worker.New()
	defaultLimiter := ratelimit.NewPostLimiter()
	as := &Server{
		worker:  defaultWorker,
		limiter: defaultLimiter,
	}
	for _, opt := range options {
		opt(as)
	}
	as.registerRoutes()
	return as
}

// WithWorker is an option that sets the background worker.
func WithWorker(w worker.Worker) ServerOption {
	return func(as *Server) {
		as.worker = w
	}
}

func WithLimiter(limiter *ratelimit.PostLimiter) ServerOption {
	return func(as *Server) {
		as.limiter = limiter
	}
}

func (as *Server) registerRoutes() {
	root := mux.NewRouter()
	root = root.StrictSlash(true)
	router := root.PathPrefix("/api/").Subrouter()
	router.Use(mid.RequireAPIKey)
	router.Use(mid.EnforceViewOnly)
	router.HandleFunc("/imap/", as.IMAPServer)
	router.HandleFunc("/imap/validate", as.IMAPServerValidate)
	router.HandleFunc("/reset", as.Reset)
	router.HandleFunc("/campaigns/", mid.Use(as.Campaigns, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/campaigns/summary", mid.Use(as.CampaignsSummary, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/campaigns/{id:[0-9]+}", mid.Use(as.Campaign, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/campaigns/{id:[0-9]+}/results", mid.Use(as.CampaignResults, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/campaigns/{id:[0-9]+}/results/{rid:[A-Za-z0-9]+}/schedule", mid.Use(as.CampaignResultSchedule, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/campaigns/{id:[0-9]+}/summary", mid.Use(as.CampaignSummary, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/campaigns/{id:[0-9]+}/complete", mid.Use(as.CampaignComplete, mid.EnforceTeamViewOnly("campaigns")))
	router.HandleFunc("/users/{id:[0-9]+}/teams", as.GetUserTeams)
	router.HandleFunc("/{item}/{id:[0-9]+}/teams", mid.Use(as.ItemTeams, mid.CanShareItem()))
	router.HandleFunc("/teams", mid.Use(as.Teams, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/teams/{id:[0-9]+}", mid.Use(as.Team, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/groups/", mid.Use(as.Groups, mid.EnforceTeamViewOnly("groups")))
	router.HandleFunc("/groups/summary", mid.Use(as.GroupsSummary, mid.EnforceTeamViewOnly("groups")))
	router.HandleFunc("/groups/{id:[0-9]+}", mid.Use(as.Group, mid.EnforceTeamViewOnly("groups")))
	router.HandleFunc("/groups/{id:[0-9]+}/summary", mid.Use(as.GroupSummary, mid.EnforceTeamViewOnly("groups")))
	router.HandleFunc("/templates/", mid.Use(as.Templates, mid.EnforceTeamViewOnly("templates")))
	router.HandleFunc("/templates/{id:[0-9]+}", mid.Use(as.Template, mid.EnforceTeamViewOnly("templates")))
	router.HandleFunc("/pages/", mid.Use(as.Pages, mid.EnforceTeamViewOnly("pages")))
	router.HandleFunc("/pages/{id:[0-9]+}", mid.Use(as.Page, mid.EnforceTeamViewOnly("pages")))
	router.HandleFunc("/scenarios/", mid.Use(as.Scenarios, mid.EnforceTeamViewOnly("scenarios")))
	router.HandleFunc("/scenarios/{id:[0-9]+}", mid.Use(as.Scenario, mid.EnforceTeamViewOnly("scenarios")))
	router.HandleFunc("/smtp/", mid.Use(as.SendingProfiles, mid.EnforceTeamViewOnly("smtp")))
	router.HandleFunc("/smtp/{id:[0-9]+}", mid.Use(as.SendingProfile, mid.EnforceTeamViewOnly("smtp")))
	router.HandleFunc("/users/", mid.Use(as.Users, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/users/{id:[0-9]+}", mid.Use(as.User))
	router.HandleFunc("/user", mid.Use(as.GetCurrent))
	router.HandleFunc("/util/send_test_email", as.SendTestEmail)
	router.HandleFunc("/import/group", as.ImportGroup)
	router.HandleFunc("/import/email", as.ImportEmail)
	router.HandleFunc("/import/site", as.ImportSite)
	router.HandleFunc("/webhooks/", mid.Use(as.Webhooks, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/{id:[0-9]+}/validate", mid.Use(as.ValidateWebhook, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/{id:[0-9]+}", mid.Use(as.Webhook, mid.RequirePermission(models.PermissionModifySystem)))
	as.handler = router
}

func (as *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.handler.ServeHTTP(w, r)
}
