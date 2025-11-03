package web

import (
	"fmt"
	"go-barcode-server/server"
	"html/template"
	"net/http"
	"strconv"
)

type WebHandler struct {
	server *server.Server
	tmpl   *template.Template
}

type TemplateData struct {
	Title       string
	Clients     []map[string]interface{}
	Logs        []server.LogEntry
	COMPort     *server.COMPort
	ClientCount int
}

func NewWebHandler(srv *server.Server) *WebHandler {
	handler := &WebHandler{server: srv}
	handler.loadTemplates()
	return handler
}

func (wh *WebHandler) loadTemplates() {
	templates, err := template.ParseFiles(
		"./web/templates/common.html",
		"./web/templates/dashboard.html",
		"./web/templates/logs.html",
		"./web/templates/test.html",
	)
	if err != nil {
		panic("Failed to parse templates: " + err.Error())
	}
	wh.tmpl = templates

	// Debug: list defined templates
	fmt.Println("Loaded templates:")
	for _, t := range wh.tmpl.Templates() {
		fmt.Println("  -", t.Name())
	}
}

func (wh *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		wh.dashboardHandler(w, r)
	case "/logs":
		wh.logsHandler(w, r)
	case "/reconnect-com":
		wh.reconnectCOMPortHandler(w, r)
	case "/static/style.css":
		wh.staticHandler(w, r)
	case "/test":
		wh.testHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (wh *WebHandler) testHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title: "Test Page",
	}

	if err := wh.tmpl.ExecuteTemplate(w, "test.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (wh *WebHandler) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:       "Barcode Server Dashboard",
		Clients:     wh.server.GetClients(),
		COMPort:     wh.server.GetCOMPort(),
		ClientCount: wh.server.GetClientCount(),
	}

	if err := wh.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (wh *WebHandler) logsHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title: "Server Logs",
		Logs:  wh.server.GetLogger().GetAllEntries(),
	}

	wh.tmpl.ExecuteTemplate(w, "logs.html", data)
}

func (wh *WebHandler) reconnectCOMPortHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portName := r.FormValue("port")
	baudRateStr := r.FormValue("baudrate")

	baudRate, err := strconv.ParseUint(baudRateStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid baud rate", http.StatusBadRequest)
		return
	}

	if err := wh.server.ReconnectCOMPort(portName, uint(baudRate)); err != nil {
		wh.server.GetLogger().Error("Failed to reconnect to COM port %s: %v", portName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (wh *WebHandler) staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/static/style.css")
}
