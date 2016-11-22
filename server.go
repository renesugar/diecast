package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var log = logging.MustGetLogger(`diecast`)

const DEFAULT_SERVE_ADDRESS = `127.0.0.1`
const DEFAULT_SERVE_PORT = 28419
const DEFAULT_ROUTE_PREFIX = `/`

type Server struct {
	Address          string
	Port             int
	Bindings         []Binding
	DefaultTemplate  string
	RootPath         string
	RoutePrefix      string
	TemplatePatterns []string
	mounts           []Mount
	router           *httprouter.Router
	server           *negroni.Negroni
	fs               http.FileSystem
	fsIsSet          bool
	fileServer       http.Handler
}

func NewServer(root string) *Server {
	return &Server{
		Address:          DEFAULT_SERVE_ADDRESS,
		Port:             DEFAULT_SERVE_PORT,
		RoutePrefix:      DEFAULT_ROUTE_PREFIX,
		RootPath:         root,
		Bindings:         make([]Binding, 0),
		TemplatePatterns: make([]string, 0),
		mounts:           make([]Mount, 0),
	}
}

func (self *Server) SetMounts(mounts []Mount) {
	self.mounts = mounts
}

func (self *Server) Mounts() []Mount {
	return self.mounts
}

func (self *Server) SetFileSystem(fs http.FileSystem) {
	self.fs = fs
	self.fsIsSet = true
	self.fileServer = http.FileServer(self.fs)
}

func (self *Server) Initialize() error {
	// always make sure the root path is absolute
	if v, err := filepath.Abs(self.RootPath); err == nil {
		cwd, err := os.Getwd()

		if v == `./` && err == nil {
			self.RootPath = cwd
		} else {
			self.RootPath = v
		}
	} else {
		return err
	}

	self.RoutePrefix = strings.TrimSuffix(self.RoutePrefix, `/`)

	// if we haven't explicitly set a filesystem, create it
	if !self.fsIsSet {
		self.SetFileSystem(http.Dir(self.RootPath))
	}

	if err := self.setupMounts(); err != nil {
		return err
	}

	if err := self.setupServer(); err != nil {
		return err
	}

	return nil
}

func (self *Server) Serve() {
	self.server.Run(fmt.Sprintf("%s:%d", self.Address, self.Port))
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	self.server.ServeHTTP(w, req)
}

func (self *Server) ShouldApplyTemplate(requestPath string) bool {
	return false
}

func (self *Server) handleFileRequest(w http.ResponseWriter, req *http.Request) {
	// normalize filename from request path
	requestPath := req.URL.Path

	// if we're looking at a directory, assume we want "index.html"
	if strings.HasSuffix(requestPath, `/`) {
		requestPath = path.Join(requestPath, `index.html`)
	}

	// remove the Route Prefix, as that's a structural part of the path but does not
	// represent where the files are (used for embedding diecast in other services
	// to avoid name collisions)
	//
	requestPath = strings.TrimPrefix(requestPath, self.RoutePrefix)

	log.Debugf("Requesting file %q", requestPath)

	// find a mount that has this file
	for _, mount := range self.mounts {
		// attempt to open the file entry
		if file, err := mount.OpenFile(requestPath); err == nil {
			// try to respond with the opened file
			if handled := self.respondToFile(requestPath, file, w, req); handled {
				log.Debugf("  File %q was handled by mount %s", requestPath, mount.MountPoint)
				return
			}
		} else {
			log.Debugf("  Skipping %q: failed to open file: %v", requestPath, err)
		}
	}

	// if we got here, try to serve the file from the filesystem
	if file, err := os.Open(path.Join(self.RootPath, requestPath)); err == nil {
		if handled := self.respondToFile(requestPath, file, w, req); handled {
			log.Debugf("  File %q was handled by filesystem", requestPath)
			return
		}
	}

	// if we got *here*, then File Not Found
	http.Error(w, fmt.Sprintf("File %q was not found.", requestPath), http.StatusNotFound)
}

func (self *Server) respondToFile(requestPath string, file *os.File, w http.ResponseWriter, req *http.Request) bool {
	if stat, err := file.Stat(); err == nil {
		if !stat.IsDir() {
			log.Debugf("File %q -> %q", requestPath, file.Name())

			// we got a real actual file here, figure out if we're templating it or not
			if self.ShouldApplyTemplate(requestPath) {
				http.Error(w, `Not Implemented`, http.StatusNotImplemented)
			} else {
				mimeType := `application/octet-stream`

				if v := mime.TypeByExtension(path.Ext(file.Name())); v != `` {
					mimeType = v
				}

				w.Header().Set(`Content-Type`, mimeType)
				io.Copy(w, file)
			}

			return true
		} else {
			// we know this is a directory, but the request didn't have a trailing slash
			// redirect
			if !strings.HasSuffix(req.URL.Path, `/`) {
				http.Redirect(w, req, fmt.Sprintf("%s/", req.URL.Path), http.StatusMovedPermanently)
				return true
			}
		}
	} else {
		log.Debugf("  Skipping %q: failed to stat file: %v", requestPath, err)
	}

	return false
}

func (self *Server) verifyRequestPathIsValid(validatePath string) error {
	if v, err := filepath.Abs(validatePath); err == nil {
		validatePath = v
	} else {
		return err
	}

	prefixInBounds := false
	validPrefixes := []string{
		self.RootPath,
	}

	for _, mount := range self.mounts {
		validPrefixes = append(validPrefixes, mount.Path)
	}

	for _, prefix := range validPrefixes {
		if v, err := filepath.Abs(prefix); err == nil {
			prefix = v
		} else {
			log.Warningf("Unable to get absolute path from %q: %v", prefix, err)
			continue
		}

		log.Debugf("Trying %q against: %s", validatePath, prefix)

		if strings.HasPrefix(validatePath, prefix) {
			prefixInBounds = true
			break
		}
	}

	if !prefixInBounds {
		return fmt.Errorf("Path %q is not a valid request path", validatePath)
	}

	return nil
}

func (self *Server) setupMounts() error {
	// initialize all mounts
	for _, mount := range self.mounts {
		if err := mount.Initialize(); err != nil {
			return err
		}
	}

	return nil
}

func (self *Server) setupServer() error {
	self.server = negroni.New()

	// setup panic recovery handler
	self.server.Use(negroni.NewRecovery())

	// setup internal/metadata routes
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("%s/_diecast", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc(fmt.Sprintf("%s/_bindings", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self.Bindings); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// all other routes proxy to this http.Handler
	mux.HandleFunc(fmt.Sprintf("%s/", self.RoutePrefix), self.handleFileRequest)

	self.server.UseHandler(mux)

	return nil
}
