package service

import (
	"dodevops-api/api/webservice/model"
	"strings"
	"testing"
)

func TestSiteErrorLogRenderingAndInjectionRejection(t *testing.T) {
	site := model.SiteSpec{Path: "/etc/nginx/conf.d/site.conf", Domains: []string{"example.test"}, Ports: []int{8080}, ErrorLog: "/var/log/nginx/site-error.log", Locations: []model.Location{{Path: "/", Kind: "static", Target: "/srv/www"}}}
	content, err := RenderSite(site)
	if err != nil || !strings.Contains(content, `error_log "/var/log/nginx/site-error.log";`) {
		t.Fatalf("missing error log: %v %s", err, content)
	}
	for _, path := range []string{"relative/error.log", "/tmp/error.log; include /tmp/evil;", "/tmp/error\nlog"} {
		site.ErrorLog = path
		if _, err = RenderSite(site); err == nil {
			t.Fatalf("accepted invalid log path %q", path)
		}
	}
}
