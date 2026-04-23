package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

func SearchLogs(ctx consolectx.Context, logSvc *service.LogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.SearchLogsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := logSvc.SearchLogs(c.Request.Context(), &req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func AnalyzeErrorLogs(ctx consolectx.Context, logSvc *service.LogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.AnalyzeErrorLogsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusOK, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := logSvc.AnalyzeErrorLogs(c.Request.Context(), &req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
