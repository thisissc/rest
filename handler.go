package rest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Handler interface {
	Prepare() error
	Handle() error
	Finish() error
	SetApp(Application)
	SetRequest(*http.Request)
	SetWriter(http.ResponseWriter)
	RenderJson(int, interface{}) error
	RenderMsg(int, int, string) error
	RenderError(error) error
	IsAuth() bool
}

type BaseHandler struct {
	App Application
	R   *http.Request
	W   http.ResponseWriter
}

func (h *BaseHandler) SetApp(app Application) {
	h.App = app
}

func (h *BaseHandler) SetRequest(r *http.Request) {
	h.R = r
}

func (h *BaseHandler) SetWriter(w http.ResponseWriter) {
	h.W = w
}

func (h *BaseHandler) FormValue(key string) string {
	return h.R.FormValue(key)
}

func (h *BaseHandler) RouteVarValue(key string) string {
	vars := mux.Vars(h.R)
	if vars != nil {
		return vars[key]
	}
	return ""
}

func (h *BaseHandler) RenderJson(code int, msg interface{}) error {
	h.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	h.W.WriteHeader(code)
	result, _ := json.Marshal(msg)
	fmt.Fprint(h.W, string(result))
	return nil
}

func (h *BaseHandler) RenderMsg(code int, result int, msg string) error {
	return h.RenderJson(code, map[string]interface{}{
		"result":  result,
		"message": msg,
	})
}

func (h *BaseHandler) RenderCsv(code int, src io.Reader) error {
	h.W.Header().Set("Content-Type", "text/csv; charset=utf-8")
	h.W.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%d.csv", time.Now().Unix()))
	h.W.WriteHeader(code)
	io.Copy(h.W, src)
	return nil
}

func (h *BaseHandler) SetPageOffset(offset string) {
	if len(offset) > 0 {
		h.W.Header().Set(PageOffsetSetHeaderName, offset)
	}
}

func (h *BaseHandler) GetUserToken() (string, bool) {
	token := h.R.Header.Get("Authorization") // Authorization: Bearer <token>
	if len(token) > 7 {
		return token[7:], true
	} else {
		return "", false
	}
}

func (h *BaseHandler) RenderError(err error) error {
	re, ok := err.(*RespError)
	if ok {
		return h.RenderMsg(re.StatusCode, re.Result, re.Message)
	} else {
		return h.RenderMsg(400, -99999, err.Error())
	}
}

func (h *BaseHandler) Prepare() error {
	return nil
}

func (h *BaseHandler) Handle() error {
	return NewRespError(500, 500, "Not implemented")
}

func (h *BaseHandler) Finish() error {
	return nil
}

func (h *BaseHandler) ParseBody(i interface{}) error {
	decoder := json.NewDecoder(h.R.Body)
	defer h.R.Body.Close()

	if err := decoder.Decode(i); err != nil {
		return NewRespError(400, 400, "Invaild json body")
	}

	return nil
}

func (h *BaseHandler) ParseXmlBody(i interface{}) error {
	defer h.R.Body.Close()
	b, err := ioutil.ReadAll(h.R.Body)
	if err != nil {
		return NewRespError(400, 400, "Read request body error")
	}
	err = xml.Unmarshal(b, i)
	if err != nil {
		return NewRespError(400, 400, "Invalid XML body")
	}
	return nil
}

func (h *BaseHandler) ClientIP() string {
	var ip string

	ips := h.R.Header.Get("X-Forwarded-For")
	items := strings.Split(ips, ",")
	if len(items) > 0 {
		ip = items[len(items)-1]
	}

	if len(ip) == 0 {
		items = strings.Split(h.R.RemoteAddr, ":")
		if len(items) > 0 {
			ip = items[0]
		}
	}

	return ip
}

func (h *BaseHandler) IsAuth() bool {
	return true
}

type BaseAuthHandler struct {
	BaseHandler

	Token  string
	Uid    string
	RoleId string
}

func (h *BaseAuthHandler) IsAuth() bool {
	if h.App.AppConfig.Debug {
		h.Uid = h.App.AppConfig.DebugUid
		h.RoleId = h.App.AppConfig.DebugRoleId
		return true
	}

	token := GetAuthToken(h.R)
	if len(token) == 0 {
		return false
	}

	userid, roleid, ok := ParseToken(token)
	if ok {
		h.Token = token
		h.Uid = userid
		h.RoleId = roleid
	}
	return ok
}
