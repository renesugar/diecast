package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func doTestServerRequest(s *Server, method string, path string, tester func(*httptest.ResponseRecorder)) {
	req := httptest.NewRequest(method,
		fmt.Sprintf("http://%s:%d%s", DEFAULT_SERVE_ADDRESS, DEFAULT_SERVE_PORT, path), nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code >= 400 {
		log.Errorf("Response %d: %s", w.Code, w.Body.String())
	}

	tester(w)
}

func TestStaticServer(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./examples/hello`)
	mounts := getTestMounts(assert)

	server.SetMounts(mounts)
	assert.Nil(server.Initialize())
	assert.Equal(len(mounts), len(server.Mounts()))

	doTestServerRequest(server, `GET`, `/_diecast`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			data := make(map[string]interface{})
			err := json.Unmarshal(w.Body.Bytes(), &data)

			assert.Nil(err)
			assert.True(len(data) > 0)
		})

	doTestServerRequest(server, `GET`, `/_bindings`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			data := make(map[string]interface{})
			err := json.Unmarshal(w.Body.Bytes(), &data)

			assert.Nil(err)
			assert.Nil(data)
		})

	doTestServerRequest(server, `GET`, `/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(string(w.Body.Bytes()), `Hello`)
		})

	doTestServerRequest(server, `GET`, `/css/bootstrap.min.css`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Contains(string(data[:]), `Bootstrap`)
		})

	doTestServerRequest(server, `GET`, `/js/jquery.min.js`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Contains(string(data[:]), `jQuery`)
		})
}

func TestStaticServerWithRoutePrefix(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./examples/hello`)
	server.RoutePrefix = `/ui`

	mounts := getTestMounts(assert)

	server.SetMounts(mounts)
	assert.Nil(server.Initialize())
	assert.Equal(len(mounts), len(server.Mounts()))

	// paths without RoutePrefix should fail
	doTestServerRequest(server, `GET`, `/_diecast`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/_bindings`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/css/bootstrap.min.css`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	// paths with RoutePrefix should succeed
	doTestServerRequest(server, `GET`, `/ui/_diecast`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			data := make(map[string]interface{})
			err := json.Unmarshal(w.Body.Bytes(), &data)

			assert.Nil(err)
			assert.True(len(data) > 0)
		})

	doTestServerRequest(server, `GET`, `/ui/_bindings`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			data := make(map[string]interface{})
			err := json.Unmarshal(w.Body.Bytes(), &data)

			assert.Nil(err)
			assert.Nil(data)
		})

	doTestServerRequest(server, `GET`, `/ui/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(string(w.Body.Bytes()), `Hello`)
		})

	doTestServerRequest(server, `GET`, `/ui/js/jquery.min.js`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Contains(string(data[:]), `jQuery`)
		})

	doTestServerRequest(server, `GET`, `/ui/css/bootstrap.min.css`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Contains(string(data[:]), `Bootstrap`)
		})
}

func TestStaticServerTemplateSomethingInMount(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./examples/hello`)
	mounts := getTestMounts(assert)

	server.SetMounts(mounts)
	server.TemplatePatterns = append(server.TemplatePatterns, `*.txt`)

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "GET\n")
		})

	doTestServerRequest(server, `POST`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "POST\n")
		})
}

func TestStaticServerTemplateSomethingInMountWithRoutePrefix(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./examples/hello`)
	server.RoutePrefix = `/ui`
	mounts := getTestMounts(assert)

	server.SetMounts(mounts)
	server.TemplatePatterns = append(server.TemplatePatterns, `*.txt`)

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `POST`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/ui/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "GET\n")
		})

	doTestServerRequest(server, `POST`, `/ui/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "POST\n")
		})
}

func TestFilesInRootSubdirectories(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./examples/test_root1`)
	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/subdir1/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(string(w.Body.Bytes()), `Hello`)
		})

	doTestServerRequest(server, `GET`, `/subdir1/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(string(w.Body.Bytes()), `Hello`)
		})
}

func TestFilesInMountSubdirectories(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./examples/hello`)
	mounts := getTestMounts(assert)

	server.SetMounts(mounts)
	server.TemplatePatterns = append(server.TemplatePatterns, `*.txt`)

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/test/subdir1`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(301, w.Code)
		})

	doTestServerRequest(server, `GET`, `/test/subdir1/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/test/subdir1/test.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "<h1>GET</h1>\n")
		})

	doTestServerRequest(server, `GET`, `/test/subdir2`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(301, w.Code)
		})

	doTestServerRequest(server, `GET`, `/test/subdir2/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "INDEX GET\n")
		})

	doTestServerRequest(server, `PUT`, `/test/subdir2/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "INDEX PUT\n")
		})

	doTestServerRequest(server, `GET`, `/test/subdir2/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "INDEX GET\n")
		})

	doTestServerRequest(server, `PUT`, `/test/subdir2/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			data := w.Body.Bytes()
			assert.Equal(string(data[:]), "INDEX PUT\n")
		})
}