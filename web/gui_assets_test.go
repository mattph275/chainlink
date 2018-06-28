package web_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink/internal/cltest"
)

func TestGuiAssets_ExactMatch(t *testing.T) {
	t.Parallel()

	app, cleanup := cltest.NewApplication()
	defer cleanup()

	resp := cltest.BasicAuthGet(app.GuiServer.URL + "/exact.html")
	cltest.AssertServerResponse(t, resp, 200)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/not_found.html")
	cltest.AssertServerResponse(t, resp, 404)
}

func TestGuiAssets_WildcardIndexMatch(t *testing.T) {
	t.Parallel()

	app, cleanup := cltest.NewApplication()
	defer cleanup()

	resp := cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/_abc123_")
	cltest.AssertServerResponse(t, resp, 200)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/jjob_specs/_abc123_")
	cltest.AssertServerResponse(t, resp, 404)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/_abc123_/runs")
	cltest.AssertServerResponse(t, resp, 200)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/_abc123_/rruns")
	cltest.AssertServerResponse(t, resp, 404)
}

func TestGuiAssets_WildcardRouteInfoMatch(t *testing.T) {
	t.Parallel()

	app, cleanup := cltest.NewApplication()
	defer cleanup()

	resp := cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/abc123/routeInfo.json")
	cltest.AssertServerResponse(t, resp, 200)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/abc123/rrouteInfo.json")
	cltest.AssertServerResponse(t, resp, 404)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/abc123/runs/routeInfo.json")
	cltest.AssertServerResponse(t, resp, 200)

	resp = cltest.BasicAuthGet(app.GuiServer.URL + "/job_specs/abc123/runs/rrouteInfo.json")
	cltest.AssertServerResponse(t, resp, 404)
}
