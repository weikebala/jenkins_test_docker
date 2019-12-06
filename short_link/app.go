package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"gopkg.in/validator.v2"
	"log"
	"net/http"
	"strconv"
)

type App struct {
	Router       *mux.Router
	Middleware   *Middleware
	Config       *Env
	ConfigHandle *IniParser
}

type shortenReq struct {
	Url string `json:"url" validate:"nonzero"`
	//ExpirationInMinutes int64  `json:"expiration_in_minutes" validate:"min=0"`
}

type shortLinkResp struct {
	ShortLink string `json:"short_link"`
}

func (a *App) Initialize() {
	//log flag 的含义
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	//配置文件读取器
	a.ConfigHandle = GetInstance()
	//路由器
	a.Router = mux.NewRouter()
	//redis存储器
	a.Config = a.getEnv()
	//日志，异常处理中间件处理
	a.Middleware = &Middleware{}
	//路由配置
	a.initializeRoutes()
}

func (a *App) getEnv() *Env {

	//redisAddr := os.Getenv("RedisAddr")
	redisAddr := a.ConfigHandle.GetString("redis", "RedisAddr")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	redisPwd := a.ConfigHandle.GetString("redis", "redisPwd")
	if redisPwd == "" {
		redisPwd = ""
	}
	redisDb := a.ConfigHandle.GetString("redis", "RedisDb")
	if redisDb == "" {
		redisDb = "0"
	}
	db, err := strconv.Atoi(redisDb)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("connect to redis (addr:%s password: %s db: %d", redisAddr, redisPwd, db)
	cli := NewRedisCli(redisAddr, redisPwd, db)
	return &Env{S: cli}
}

func (a *App) Run() {
	addr := a.ConfigHandle.GetString("server", "port")
	log.Fatal(http.ListenAndServe(addr, a.Router))
}

func (a *App) initializeRoutes() {
	//a.Router.HandleFunc("/api/shorten", a.createShortLink).Methods("POST")
	//a.Router.HandleFunc("/api/info", a.getShortLinkInfo).Methods("GET")
	//a.Router.HandleFunc("/{shortLink:[a-zA-z0-9]{1,11}", a.redirect).Methods("GET")
	//m := alice.New(a.Middleware.LoggingHandler, a.Middleware.RecoverHandler)
	m := alice.New(a.Middleware.LoggingHandler, a.Middleware.RecoverHandler)
	a.Router.Handle("/api/shorten", m.ThenFunc(a.createShortLink)).Methods("POST")
	a.Router.Handle("/api/info", m.ThenFunc(a.getShortLinkInfo)).Methods("GET")
	a.Router.Handle("/{shortLink:[a-zA-z0-9]{1,20}}", m.ThenFunc(a.redirect)).Methods("GET")
}

// 生成短地址
func (a *App) createShortLink(w http.ResponseWriter, r *http.Request) {
	var req shortenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseWithError(w, NewBadReqErr(fmt.Errorf("%s", "happen err when parsing json from body")), nil)
		return
	}
	if err := validator.Validate(req); err != nil {
		responseWithError(w, NewBadReqErr(fmt.Errorf("validate parameters failed : %+v", req)), nil)
		return
	}
	defer r.Body.Close()
	shorten, err := a.Config.S.Shorten(req.Url, 2592000*2)
	if err != nil {
		responseWithError(w, err, nil)
	} else {
		responseWithJson(w, http.StatusCreated, shortLinkResp{ShortLink: a.ConfigHandle.GetString("shortlink", "domain") + shorten})
	}
}

// 短地址解析
func (a *App) getShortLinkInfo(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	sl := values.Get("shortLink")
	info, err := a.Config.S.ShortLinkInfo(sl)
	if err != nil {
		responseWithError(w, err, nil)
	} else {
		responseWithJson(w, http.StatusOK, info)
	}
}

//访问，重定向 302
func (a *App) redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url, err := a.Config.S.UnShorten(vars["shortLink"])
	if err != nil {
		responseWithError(w, err, nil)
	} else {
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func responseWithError(w http.ResponseWriter, err error, payload interface{}) {
	switch e := err.(type) {
	case MiError:
		log.Printf("http %d - %s", e.Status(), e)
		resp, _ := json.Marshal(Response{Code: e.Status(), Message: e.Error(), Content: payload})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	default:
		responseWithJson(w, http.StatusInternalServerError, err.Error())
	}
}

func responseWithJson(w http.ResponseWriter, status int, payload interface{}) {
	resp, _ := json.Marshal(Response{Code: status, Message: http.StatusText(status), Content: payload})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Content interface{} `json:"content"`
}
