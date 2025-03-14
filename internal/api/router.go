package api

import (
	"net/http"

	"github.com/balu-dk/go-cpms/internal/api/handlers"
	"github.com/balu-dk/go-cpms/internal/api/middleware"
	"github.com/balu-dk/go-cpms/internal/service"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// API handles the API server
type API struct {
	router  chi.Router
	handler *handlers.Handler
}

// NewAPI creates a new API server
func NewAPI(cpms *service.CPMS) *API {
	router := chi.NewRouter()
	handler := handlers.NewHandler(cpms)

	// Setup middleware
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.ContentType)

	// CORS configuration
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Setup routes
	router.Route("/api/v1", func(r chi.Router) {
		// Charge Point routes
		r.Route("/chargepoints", func(r chi.Router) {
			r.Get("/", handler.GetChargePoints)
			r.Get("/{id}", handler.GetChargePoint)
			r.Get("/{id}/connectors", handler.GetConnectors)

			// OCPP commands
			r.Post("/{id}/reset", handler.Reset)
			r.Post("/{id}/availability", handler.ChangeAvailability)
			r.Post("/{id}/unlock", handler.UnlockConnector)
			r.Post("/{id}/starttransaction", handler.RemoteStartTransaction)
			r.Post("/{id}/stoptransaction", handler.RemoteStopTransaction)
			r.Post("/{id}/heartbeat", handler.TriggerHeartbeat)
			r.Post("/{id}/diagnostics", handler.GetDiagnostics)
			r.Post("/{id}/firmware", handler.UpdateFirmware)
			r.Post("/{id}/clearcache", handler.ClearCache)
			r.Post("/{id}/configuration", handler.GetConfiguration)
			r.Put("/{id}/configuration", handler.ChangeConfiguration)
		})

		// Transaction routes
		r.Route("/transactions", func(r chi.Router) {
			r.Get("/{id}", handler.GetTransaction)
		})
	})

	return &API{
		router:  router,
		handler: handler,
	}
}

// ServeHTTP satisfies the http.Handler interface
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}
