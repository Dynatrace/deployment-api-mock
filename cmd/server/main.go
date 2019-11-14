package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//type queryKey struct{ Token, Platform, InstallerType string }
type queryKey struct{ Platform, InstallerType string }

type queryHandler func(w http.ResponseWriter, r *http.Request) error

type deploymentAPI struct {
	mtx   sync.Mutex
	Mocks map[queryKey]queryHandler
}

func (api *deploymentAPI) installerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := queryKey{
		// returns only the token instead of the whole 'Api-Token 123asdf' string
		strings.Split(r.Header.Get("Authorization"), " ")[1],
		vars["platform"],
		vars["installerType"],
	}

	api.mtx.Lock()
	handler, ok := api.Mocks[key]
	api.mtx.Unlock()

	if !ok {
		log.Println("No handler found for", key)
		http.NotFound(w, r)
	} else {
		if err := handler(w, r); err != nil {
			log.Println("Failed to write response:", err)
		}
	}
}

func (api *deploymentAPI) registerHandler(w http.ResponseWriter, r *http.Request) {
	platform := r.FormValue("platform")
	installerType := r.FormValue("installerType")
	//apiToken := r.FormValue("apiToken")

	if platform == "" || installerType == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("One of required arguments are missing: platform, installerType\n"))
		return
	}

	var wait time.Duration
	var err error

	if rawWait := r.FormValue("waitTime"); rawWait != "" {
		if wait, err = time.ParseDuration(rawWait); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error() + "\n"))
			return
		}
	}

	rw, err := makeResponseWriter(platform, installerType, r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error() + "\n"))
		return
	}

	handler := func(response http.ResponseWriter, request *http.Request) error {
		if wait != 0 {
			log.Println("Sleeping", wait, "before sending response.")
			time.Sleep(wait)
		}

		response.Header().Set("Content-Type", "application/octet-stream")
		response.WriteHeader(http.StatusOK)

		return rw(response)
	}

	api.mtx.Lock()
	//api.Mocks[queryKey{apiToken, platform, installerType}] = handler
	api.Mocks[queryKey{platform, installerType}] = handler
	api.mtx.Unlock()

	w.WriteHeader(http.StatusOK)
}

func makeResponseWriter(platform, installerType string, settings url.Values) (func(io.Writer) error, error) {
	exitCode := settings.Get("exitCode")
	if exitCode == "" {
		exitCode = "0"
	}

	switch platform {
	case "unix":
		switch installerType {
		case "default":
			return func(w io.Writer) error {
				_, err := w.Write([]byte(makeUnixInstaller(exitCode)))
				return err
			}, nil
		default:
			return nil, fmt.Errorf("Unknown installer type: %s", installerType)
		}

	case "windows":
		if exitCode != "0" {
			return nil, fmt.Errorf("Windows installer doesn't support non-0 exit codes")
		}

		switch installerType {
		case "default":
			return func(w io.Writer) error {
				in, err := os.Open("winsvc.exe")
				if err != nil {
					return err
				}
				defer in.Close()

				if _, err = io.Copy(w, in); err != nil {
					return err
				}

				return nil
			}, nil
		default:
			return nil, fmt.Errorf("Unknown installer type: %s", installerType)
		}

	default:
		return nil, fmt.Errorf("Unknown platform: %s", platform)
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	api := &deploymentAPI{
		Mocks: make(map[queryKey]queryHandler),
	}

	r := mux.NewRouter()
	r.HandleFunc("/v1/deployment/installer/agent/{platform}/{installerType}/latest", api.installerHandler).Methods("GET")
	r.HandleFunc("/register", api.registerHandler).Methods("POST")

	http.Handle("/", handlers.LoggingHandler(os.Stdout, r))

	log.Println("Running server at port 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error running the server: %v", err)
	}
}
