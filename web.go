package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Web struct {
	WebConf
	// gin.Engine
	svr *App
}

type CR struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// middleware: check token
func (self *Web) authHandler(c *gin.Context) {
	if c.FullPath() == "/health" {
		return
	}

	token := c.GetHeader("Authorization")
	if token != self.Token {
		c.JSON(http.StatusUnauthorized, CR{
			Code:    1,
			Message: "token not match",
		})
		logrus.Infof("[web.go::authHandler] unexpect token: %v", token)
		c.Abort()
		return
	}
}

// GET /health
// Health for Profile
func (self *Web) health(c *gin.Context) {
	c.JSON(http.StatusOK, CR{
		Code: 0, Message: "OK",
	})
}

// GET /servers
// return servers && server config section
func (self *Web) getServers(c *gin.Context) {
	app := self.svr
	sm := app.serversMap
	c.JSON(http.StatusOK, sm)
}

// GET /server/{name}
// get full config of a server
func (self *Web) getServer(c *gin.Context) {
	name := c.Param("name")
	app := self.svr
	sm := app.serversMap
	n, exist := sm[name]
	if !exist {
		c.JSON(http.StatusNotFound, CR{
			Code:    1,
			Message: "not found",
		})
		return
	}
	c.JSON(http.StatusOK, n)
}

// middleware: check readonly
func (self *Web) readOnly(c *gin.Context) {
	if self.Readonly {
		c.JSON(http.StatusForbidden, CR{
			Code: 1, Message: "webconf set readonly",
		})
		c.Abort()
		return
	}
}

// PUT /server/{name}
// add a new server
func (self *Web) addServer(c *gin.Context) {
	name := c.Param("name")

	app := self.svr
	sm := app.serversMap
	//TODO: lock
	_, exist := sm[name]
	if exist {
		c.JSON(http.StatusBadRequest, CR{
			Code:    1,
			Message: "server already exist",
		})
		return
	}
}

// DELETE /server/{name}
// add a new server
func (self *Web) delServer(c *gin.Context) {
	name := c.Param("name")
	app := self.svr
	sm := app.serversMap
	n, exist := sm[name]
	if !exist {
		c.JSON(http.StatusNotFound, CR{
			Code:    1,
			Message: "not found",
		})
		return
	}
	n.Update(nil)
	c.JSON(http.StatusOK, n)
}

// POST /server/{name}
// set a server
func (self *Web) setServer(c *gin.Context) {
	name := c.Param("name")
	app := self.svr
	sm := app.serversMap
	n, exist := sm[name]
	if !exist {
		c.JSON(http.StatusNotFound, CR{
			Code:    1,
			Message: "not found",
		})
		return
	}
	n.Update(nil)
	c.JSON(http.StatusOK, n)
}

// POST /server/{name}
// set a server
func (self *Web) Run() {
	r := gin.Default()
	r.Use(self.authHandler)

	r.GET("/health", self.health)
	r.GET("/servers", self.getServers)
	r.GET("/server/:name", self.getServer)
	r.PUT("/server/:name", self.readOnly, self.addServer)
	r.POST("/server/:name", self.readOnly, self.setServer)
	r.DELETE("/server/:name", self.readOnly, self.delServer)

	r.Run(self.Addr)
}
