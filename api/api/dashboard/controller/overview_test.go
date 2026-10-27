package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	model "dodevops-api/api/dashboard/model1"
	"dodevops-api/api/dashboard/service"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"github.com/gin-gonic/gin"
)

type overviewStub struct {
	service.IDashboardService
	called bool
	days   int
	userID uint
}

func (s *overviewStub) GetOverview(_ context.Context, userID uint, _ []uint, days int) (*model.Overview, error) {
	s.called, s.days, s.userID = true, days, userID
	return &model.Overview{Days: days, GeneratedAt: time.Now(), Errors: map[string]string{}}, nil
}

func TestOverviewAuthenticationAndRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		query  string
		logged bool
		code   int
		days   int
	}{
		{"", false, 401, 0},
		{"?days=0", true, 400, 0},
		{"?days=abc", true, 400, 0},
		{"?days=365", true, 400, 0},
		{"", true, 200, 7},
		{"?days=30", true, 200, 30},
		{"?days=90", true, 200, 90},
	} {
		t.Run(fmt.Sprintf("logged=%t%s", tc.logged, tc.query), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("GET", "/dashboard/overview"+tc.query, nil)
			if tc.logged {
				ctx.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: 17})
			}
			stub := &overviewStub{}
			(&DashboardController{service: stub}).GetOverview(ctx)
			var response struct {
				Code int
				Data model.Overview
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != tc.code || stub.called != (tc.code == 200) {
				t.Fatalf("unexpected response/call: %s called=%v", recorder.Body, stub.called)
			}
			if stub.called && (stub.days != tc.days || stub.userID != 17 || response.Data.Days != tc.days) {
				t.Fatal("controller must pass authenticated user and selected range")
			}
		})
	}
}
