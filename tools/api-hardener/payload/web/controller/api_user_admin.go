//go:build toolsignore
// +build toolsignore

package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/web/service"

	"github.com/gin-gonic/gin"
)

// APIUserAdminController exposes API user management for panel admins (session-protected).
type APIUserAdminController struct {
	BaseController
	apiUserService service.APIUserService
	settingService service.SettingService
}

// NewAPIUserAdminController registers routes for managing API users and settings.
func NewAPIUserAdminController(g *gin.RouterGroup) *APIUserAdminController {
	a := &APIUserAdminController{}
	a.initRouter(g)
	return a
}

type createAPIUserForm struct {
	Name string `json:"name" form:"name"`
	Rate int    `json:"rate" form:"rate"`
}

type updateRateForm struct {
	Rate int `json:"rate" form:"rate"`
}

type updateAPISettingForm struct {
	APITokenOnly        bool `json:"apiTokenOnly" form:"apiTokenOnly"`
	APIDefaultRateLimit int  `json:"apiDefaultRateLimit" form:"apiDefaultRateLimit"`
}

func (a *APIUserAdminController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/api-users")

	g.GET("/list", a.list)
	g.POST("/create", a.create)
	g.POST("/enable/:id", a.enable)
	g.POST("/disable/:id", a.disable)
	g.POST("/delete/:id", a.delete)
	g.POST("/rotate/:id", a.rotate)
	g.POST("/rate/:id", a.rate)

	g.GET("/settings", a.getSettings)
	g.POST("/settings", a.updateSettings)
}

func (a *APIUserAdminController) list(c *gin.Context) {
	users, err := a.apiUserService.ListUsers()
	jsonObj(c, users, err)
}

func (a *APIUserAdminController) create(c *gin.Context) {
	form := &createAPIUserForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	user, token, err := a.apiUserService.CreateUser(form.Name, form.Rate)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     I18nWeb(c, "pages.settings.api.tokenGenerated"),
		"obj": gin.H{
			"user":  user,
			"token": token,
		},
	})
}

func (a *APIUserAdminController) enable(c *gin.Context) {
	id := mustID(c.Param("id"))
	err := a.apiUserService.SetEnabled(id, true)
	jsonMsg(c, I18nWeb(c, "pages.settings.api.userEnabled"), err)
}

func (a *APIUserAdminController) disable(c *gin.Context) {
	id := mustID(c.Param("id"))
	err := a.apiUserService.SetEnabled(id, false)
	jsonMsg(c, I18nWeb(c, "pages.settings.api.userDisabled"), err)
}

func (a *APIUserAdminController) delete(c *gin.Context) {
	id := mustID(c.Param("id"))
	err := a.apiUserService.DeleteUser(id)
	jsonMsg(c, I18nWeb(c, "pages.settings.api.userDeleted"), err)
}

func (a *APIUserAdminController) rotate(c *gin.Context) {
	id := mustID(c.Param("id"))
	token, err := a.apiUserService.RotateToken(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.api.tokenRotateFailed"), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     I18nWeb(c, "pages.settings.api.tokenRotated"),
		"obj": gin.H{
			"token": token,
		},
	})
}

func (a *APIUserAdminController) rate(c *gin.Context) {
	id := mustID(c.Param("id"))
	form := &updateRateForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.api.rateUpdateFailed"), err)
		return
	}
	err := a.apiUserService.UpdateRateLimit(id, form.Rate)
	jsonMsg(c, I18nWeb(c, "pages.settings.api.rateUpdated"), err)
}

func (a *APIUserAdminController) getSettings(c *gin.Context) {
	apiTokenOnly, _ := a.settingService.GetAPITokenOnly()
	defaultRate, _ := a.settingService.GetAPIDefaultRateLimit()
	jsonObj(c, updateAPISettingForm{
		APITokenOnly:        apiTokenOnly,
		APIDefaultRateLimit: defaultRate,
	}, nil)
}

func (a *APIUserAdminController) updateSettings(c *gin.Context) {
	form := &updateAPISettingForm{}
	if err := c.ShouldBind(form); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.api.settingsUpdateFailed"), err)
		return
	}
	if err := a.settingService.SetAPITokenOnly(form.APITokenOnly); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.api.settingsUpdateFailed"), err)
		return
	}
	err := a.settingService.SetAPIDefaultRateLimit(form.APIDefaultRateLimit)
	jsonMsg(c, I18nWeb(c, "pages.settings.api.settingsUpdated"), err)
}

func mustID(raw string) int {
	id, _ := strconv.Atoi(strings.TrimSpace(raw))
	return id
}
