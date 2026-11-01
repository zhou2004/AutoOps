package controller

import (
	cmdbmodel "dodevops-api/api/cmdb/model"
	databasemodel "dodevops-api/api/database/model"
	databaseservice "dodevops-api/api/database/service"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/middleware"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func (ctrl *DatabaseOperationController) RegisterMySQL(group *gin.RouterGroup) {
	p := middleware.NewCmdbAssetPermissionMiddleware()
	r := group.Group("/databases/:id/mysql")
	read := p.RequireAssetActionForID("database", "db_query", "id")
	write := p.RequireAssetActionForID("database", "db_execute", "id")
	r.GET("/capabilities", read, ctrl.MySQLCapabilities)
	r.GET("/table", read, ctrl.MySQLTable)
	r.GET("/data", read, ctrl.MySQLData)
	r.GET("/export", read, ctrl.MySQLExport)
	r.POST("/rows", write, ctrl.MySQLRow)
	r.POST("/import", write, ctrl.MySQLImport)
	r.POST("/schema", write, ctrl.MySQLSchema)
	r.GET("/variables", read, ctrl.MySQLVariables)
	r.PUT("/variables", p.AdminOnly(), ctrl.MySQLSetVariable)
}
func mysqlDataRequest(c *gin.Context) service.MySQLDataRequest {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "50"))
	return service.MySQLDataRequest{Database: c.Query("database"), Table: c.Query("table"), Page: page, Size: size, Sort: c.Query("sort"), Desc: c.Query("desc") == "true", Column: c.Query("column"), Operator: c.Query("operator"), Value: c.Query("value")}
}
func (ctrl *DatabaseOperationController) MySQLTable(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	ctrl.audited(c, id, "table_structure", c.Query("database")+"/"+c.Query("table"), func() (any, error) {
		return ctrl.service.MySQLTable(c.Request.Context(), id, c.Query("database"), c.Query("table"))
	})
}
func (ctrl *DatabaseOperationController) MySQLData(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	r := mysqlDataRequest(c)
	ctrl.audited(c, id, "table_data", r.Database+"/"+r.Table, func() (any, error) { return ctrl.service.MySQLData(c.Request.Context(), id, r, false) })
}
func (ctrl *DatabaseOperationController) MySQLRow(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	var r service.MySQLRowRequest
	if c.ShouldBindJSON(&r) != nil {
		result.Failed(c, 400, "行操作参数无效或超过 1 MiB")
		return
	}
	ctrl.audited(c, id, "row_"+r.Action, r.Database+"/"+r.Table, func() (any, error) {
		n, e := ctrl.service.MySQLMutateRow(c.Request.Context(), id, r)
		return gin.H{"affectedRows": n}, e
	})
}
func (ctrl *DatabaseOperationController) MySQLExport(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	r := mysqlDataRequest(c)
	format := c.DefaultQuery("format", "json")
	ctrl.audited(c, id, "table_export", r.Database+"/"+r.Table, func() (any, error) {
		if format != "csv" && format != "json" {
			return nil, fmt.Errorf("仅支持 CSV/JSON 导出")
		}
		data, e := ctrl.service.MySQLData(c.Request.Context(), id, r, true)
		if e != nil {
			return nil, e
		}
		content, e := service.EncodeMySQLExport(data, format)
		return gin.H{"content": string(content), "format": format, "rows": len(data.Rows)}, e
	})
}
func (ctrl *DatabaseOperationController) MySQLImport(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, (8<<20)+(64<<10))
	if c.Request.ParseMultipartForm(1<<20) != nil {
		result.Failed(c, 400, "导入文件格式无效或超过 8 MiB")
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.Failed(c, 400, "请选择导入文件")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, (8<<20)+1))
	if err != nil || len(content) > 8<<20 {
		result.Failed(c, 400, "导入文件超过 8 MiB")
		return
	}
	database, table := c.PostForm("database"), c.PostForm("table")
	ctrl.audited(c, id, "table_import", database+"/"+table, func() (any, error) {
		n, e := ctrl.service.MySQLImport(c.Request.Context(), id, database, table, strings.ToLower(c.PostForm("format")), c.PostForm("confirmName"), content)
		return gin.H{"affectedRows": n}, e
	})
}
func (ctrl *DatabaseOperationController) MySQLSchema(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	var r service.MySQLDefinitionRequest
	if c.ShouldBindJSON(&r) != nil {
		result.Failed(c, 400, "结构操作参数无效")
		return
	}
	ctrl.audited(c, id, "schema_"+r.Action, r.Database+"/"+r.Table, func() (any, error) { return true, ctrl.service.MySQLDefinition(c.Request.Context(), id, r) })
}
func (ctrl *DatabaseOperationController) MySQLVariables(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	ctrl.audited(c, id, "read_variables", "global", func() (any, error) { return ctrl.service.MySQLVariables(c.Request.Context(), id) })
}
func (ctrl *DatabaseOperationController) MySQLSetVariable(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	var r struct {
		Name        string `json:"name"`
		Value       string `json:"value"`
		ConfirmName string `json:"confirmName"`
	}
	if c.ShouldBindJSON(&r) != nil {
		result.Failed(c, 400, "配置参数无效")
		return
	}
	ctrl.audited(c, id, "set_variable", r.Name, func() (any, error) {
		var source cmdbmodel.CmdbSQL
		if err := ctrl.db.First(&source, id).Error; err != nil {
			return nil, err
		}
		if r.ConfirmName != source.Name {
			return nil, errors.New("请输入完整数据源名称确认配置变更")
		}
		if source.ClusterID > 0 {
			user, err := jwt.GetAdminId(c)
			if err != nil {
				return nil, err
			}
			domain := databaseservice.New(ctrl.db)
			if err = domain.Authorize(user, source.ClusterID, "db_cluster_config"); err != nil {
				return nil, err
			}
			conn, err := ctrl.service.OpenSQL(c.Request.Context(), id, "")
			if err != nil {
				return nil, err
			}
			var identity string
			err = conn.QueryRowContext(c.Request.Context(), "SELECT @@server_uuid").Scan(&identity)
			conn.Close()
			if err != nil {
				return nil, errors.New("无法确认连接的真实实例")
			}
			var member databasemodel.Member
			if ctrl.db.Where("cluster_id = ? AND native_id = ? AND kind = ?", source.ClusterID, identity, "data").First(&member).Error != nil {
				return nil, errors.New("实际实例未登记到当前集群，请先刷新拓扑")
			}
			cluster, err := domain.Refresh(c.Request.Context(), source.ClusterID)
			if err != nil {
				return nil, err
			}
			plan, err := domain.Plan(c.Request.Context(), cluster.ID, user, "configure", databaseservice.Parameters{MemberID: member.ID, RuntimeOnly: true, Changes: map[string]string{r.Name: r.Value}})
			if err != nil {
				return nil, err
			}
			if !plan.Executable {
				return nil, errors.New("集群配置预检未通过，请查看管理级别、权限及实例健康")
			}
			op, err := domain.Submit(c.Request.Context(), cluster.ID, user, plan.ID, fmt.Sprintf("workbench-config-%d-%d", id, time.Now().UnixNano()), cluster.Name)
			if err != nil {
				return nil, err
			}
			return gin.H{"operationId": op.ID}, nil
		}
		return true, ctrl.service.MySQLSetVariable(c.Request.Context(), id, r.Name, r.Value, r.ConfirmName)
	})
}

func (ctrl *DatabaseOperationController) MySQLCapabilities(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	user, e := jwt.GetAdminId(c)
	if e != nil {
		result.Failed(c, 401, "请先登录")
		return
	}
	permission := dao.NewCmdbAssetPermissionDao(ctrl.db)
	write, e := permission.CanUserAccess(user, "database", id, "db_execute")
	if e != nil {
		result.Failed(c, 500, "权限读取失败")
		return
	}
	audit, e := permission.CanUserAccess(user, "database", id, "audit")
	if e != nil {
		result.Failed(c, 500, "权限读取失败")
		return
	}
	result.Success(c, gin.H{"write": write, "audit": audit})
}
