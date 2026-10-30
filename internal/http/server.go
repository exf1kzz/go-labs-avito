package transport

import (
	"context"
	"net/http"
	"time"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	createhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/create"
	finishhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/finish"
	gethttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/get"
)

type databasePinger interface {
	Ping(context.Context) error
}

type Server struct {
	api.Unimplemented

	database     databasePinger
	createTrip   *createhttp.Handler
	getTrip      *gethttp.Handler
	finishTrip   *finishhttp.Handler
	queryTimeout time.Duration
}

func NewServer(
	pinger databasePinger,
	createTrip *createhttp.Handler,
	getTrip *gethttp.Handler,
	finishTrip *finishhttp.Handler,
	queryTimeout time.Duration,
) *Server {
	return &Server{
		database:     pinger,
		createTrip:   createTrip,
		getTrip:      getTrip,
		finishTrip:   finishTrip,
		queryTimeout: queryTimeout,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, p api.CreateTripParams) {
	s.createTrip.CreateTrip(w, r, p)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.getTrip.GetTrip(w, r, id)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.finishTrip.FinishTrip(w, r, id)
}

var _ api.ServerInterface = (*Server)(nil)
