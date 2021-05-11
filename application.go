package rest

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

const (
	PageOffsetSetHeaderName = "Page-Offset"

	DefaultServerTimeout = 15 * time.Second
)

type Application struct {
	Router     *mux.Router
	ConfigFile string
	AppConfig  AppConfig
	ServerPort int
}

func NewApplication() *Application {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var port int
	var config string
	flag.IntVar(&port, "port", 3001, "Port on listening")
	flag.StringVar(&config, "config", "config.toml", "Path of configuration")
	flag.Parse()

	app := Application{
		Router:     mux.NewRouter(),
		ConfigFile: config,
		ServerPort: port,
	}
	return &app
}

func (app *Application) AddRoute(path string, handler Handler, method string) {
	className := reflect.Indirect(reflect.ValueOf(handler)).Type()
	f := func(w http.ResponseWriter, r *http.Request) {
		h, ok := reflect.New(className).Interface().(Handler)
		if ok {
			h.SetApp(*app)
			h.SetRequest(r)
			h.SetWriter(w)

			err := h.Prepare()
			if err != nil {
				h.RenderError(err)
			} else if h.IsAuth() {
				err = h.Handle()
				if err != nil {
					h.RenderError(err)
				}
			} else {
				err := NewRespError(401, 401, "Not login")
				h.RenderError(err)
			}

			h.Finish()
		} else {
			log.Printf("%s is not implemented interface Handler\n", className)
		}
	}
	app.Router.HandleFunc(path, f).Methods(method)
}

func (app *Application) Run() {
	app.AppConfig = globalAppConfig
	app.Router.HandleFunc("/ping", PingHandler).Methods("GET")

	corsDefault := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool { // https://github.com/rs/cors/pull/57
			origin = strings.ToLower(origin)
			allowedOriginList := strings.Split(app.AppConfig.AllowedOrigin, ",")
			allowed := false
			for _, suffix := range allowedOriginList {
				if len(suffix) > 0 {
					allowed = strings.HasSuffix(origin, suffix)
					if allowed {
						break
					}
				}
			}
			if !allowed {
				allowed, _ = regexp.MatchString(`^http://(localhost|127.0.0.1)(:[0-9]+)?$`, origin)
			}
			return allowed
		},
		AllowedHeaders:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "DELETE", "PUT", "PATCH", "OPTION"},
		ExposedHeaders:     []string{PageOffsetSetHeaderName},
		AllowCredentials:   true,
		OptionsPassthrough: false,
	})

	writeTimeout := DefaultServerTimeout
	if app.AppConfig.WriteTimeout > 0 {
		writeTimeout = time.Duration(app.AppConfig.WriteTimeout) * time.Second
	}

	readTimeout := DefaultServerTimeout
	if app.AppConfig.ReadTimeout > 0 {
		readTimeout = time.Duration(app.AppConfig.ReadTimeout) * time.Second
	}

	srv := &http.Server{
		Handler:      corsDefault.Handler(app.Router),
		Addr:         fmt.Sprintf(":%d", app.ServerPort),
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
	}

	log.Printf("LISTEN %d\n", app.ServerPort)
	log.Fatal(srv.ListenAndServe())
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
}
