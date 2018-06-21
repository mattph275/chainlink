package web

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr"
	"github.com/smartcontractkit/chainlink/logger"
	"github.com/smartcontractkit/chainlink/services"
	"github.com/smartcontractkit/chainlink/store"
)

// Router listens and responds to requests to the node for valid paths.
func Router(app *services.ChainlinkApplication) (server *gin.Engine, gui *gin.Engine) {
	server = serverEngine(app)
	gui = guiEngine(app)
	return server, gui
}

// Serve API requests
func serverEngine(app *services.ChainlinkApplication) *gin.Engine {
	engine := gin.New()
	config := app.Store.Config
	basicAuth := gin.BasicAuth(gin.Accounts{config.BasicAuthUsername: config.BasicAuthPassword})
	cors := uiCorsHandler(config)
	engine.Use(
		loggerFunc(),
		gin.Recovery(),
		cors,
		basicAuth,
	)

	v1 := engine.Group("/v1")
	{
		ac := AssignmentsController{app}
		v1.POST("/assignments", ac.Create)
		v1.GET("/assignments/:ID", ac.Show)

		sc := SnapshotsController{app}
		v1.POST("/assignments/:AID/snapshots", sc.CreateSnapshot)
		v1.GET("/snapshots/:ID", sc.ShowSnapshot)
	}

	v2 := engine.Group("/v2")
	{
		ab := AccountBalanceController{app}
		v2.GET("/account_balance", ab.Show)

		j := JobSpecsController{app}
		v2.GET("/specs", j.Index)
		v2.POST("/specs", j.Create)
		v2.GET("/specs/:SpecID", j.Show)

		jr := JobRunsController{app}
		v2.GET("/specs/:SpecID/runs", jr.Index)
		v2.POST("/specs/:SpecID/runs", jr.Create)
		v2.PATCH("/runs/:RunID", jr.Update)
		v2.GET("/runs/:RunID", jr.Show)

		tt := BridgeTypesController{app}
		v2.GET("/bridge_types", tt.Index)
		v2.POST("/bridge_types", tt.Create)
		v2.GET("/bridge_types/:BridgeName", tt.Show)
		v2.DELETE("/bridge_types/:BridgeName", tt.Destroy)

		backup := BackupController{app}
		v2.GET("/backup", backup.Show)

		cc := ConfigController{app}
		v2.GET("/config", cc.Show)
	}

	return engine
}

// Serve static assets for the GUI
func guiEngine(app *services.ChainlinkApplication) *gin.Engine {
	engine := gin.New()
	config := app.Store.Config
	basicAuth := gin.BasicAuth(gin.Accounts{config.BasicAuthUsername: config.BasicAuthPassword})
	engine.Use(
		loggerFunc(),
		gin.Recovery(),
		basicAuth,
	)

	box := guiBox()
	boxList := box.List()

	engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if filepath.Ext(path) == "" {
			matchedPath := matchPath(boxList, path, "/index.html")
			if matchedPath != "" {
				writeData(c, "text/html", box, matchedPath)
			}
		} else if filepath.Ext(path) == ".json" {
			matchedPath := matchPath(boxList, path, "")
			if matchedPath != "" {
				writeData(c, "appliction/json", box, matchedPath)
			}
		}
	})
	engine.StaticFS("/", box)

	return engine
}

func guiBox() packr.Box {
	if flag.Lookup("test.v") == nil {
		return packr.NewBox("../gui/dist/")
	}

	return packr.NewBox("../internal/fixtures/gui/dist/")
}

func matchPath(boxList []string, path string, suffix string) (matchedPath string) {
	pathSegments := strings.Split(path, "/")
	pathSegments = pathSegments[1:]

	if len(pathSegments) > 0 {
		i := 0
		for i < len(pathSegments) && matchedPath == "" {
			wildcardPathSegments := []string{}
			wildcardPathSegments = append(wildcardPathSegments, pathSegments...)
			wildcardPathSegments[i] = ":[a-zA-Z]+"
			wildcardPath := strings.Join(wildcardPathSegments, "/")
			wildcardIndex := fmt.Sprintf("^%s%s$", wildcardPath, suffix)

			j := 0
			for j < len(boxList) && matchedPath == "" {
				boxPath := boxList[j]
				match, _ := regexp.MatchString(wildcardIndex, boxPath)
				if match {
					matchedPath = boxPath
				}
				j++
			}
			i++
		}

	}

	return matchedPath
}

func writeData(c *gin.Context, contentType string, box packr.Box, serveFile string) {
	index, err := box.Open(serveFile)
	if err != nil {
		logger.Warn(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(index)
	c.Data(http.StatusOK, contentType, buf.Bytes())
}

// Inspired by https://github.com/gin-gonic/gin/issues/961
func loggerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		buf, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			logger.Warn("Web request log error: ", err.Error())
			c.Next()
			return
		}
		rdr := bytes.NewBuffer(buf)
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

		start := time.Now()
		c.Next()
		end := time.Now()

		logger.Infow("Web request",
			"method", c.Request.Method,
			"status", c.Writer.Status(),
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"body", readBody(rdr),
			"clientIP", c.ClientIP(),
			"errors", c.Errors.String(),
			"servedAt", end.Format("2006/01/02 - 15:04:05"),
			"latency", fmt.Sprintf("%v", end.Sub(start)),
		)
	}
}

// Add CORS headers so UI can make api requests
func uiCorsHandler(config store.Config) gin.HandlerFunc {
	webpackDevServer := "http://localhost:3000"
	gui := "http://localhost:" + config.GuiPort
	c := cors.Config{
		AllowOrigins:     []string{webpackDevServer, gui},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	return cors.New(c)
}

func readBody(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	s := buf.String()
	return s
}
