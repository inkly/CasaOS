package v1

import (
	"github.com/ReCasaOS/CasaOS/model"
	"github.com/ReCasaOS/CasaOS/pkg/utils/common_err"
	"github.com/ReCasaOS/CasaOS/service"
	"github.com/labstack/echo/v4"
)

// GetSambaUsersList returns the share accounts CasaOS created. Accounts that
// belong to the operating system are not listed and cannot be managed here.
func GetSambaUsersList(ctx echo.Context) error {
	users, err := service.ListSambaUsers()
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: users})
}

// PostSambaUserCreate adds an account that can mount protected shares. It has no
// home directory and no login shell, so it cannot be used to reach the host.
func PostSambaUserCreate(ctx echo.Context) error {
	request := model.SambaUser{}
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
	}

	if err := service.CreateSambaUser(request.Username, request.Password); err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
	}

	// Deliberately echoes only the name back.
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: request.Username})
}

// PutSambaUserPassword replaces the password of an existing share account.
func PutSambaUserPassword(ctx echo.Context) error {
	username := ctx.Param("username")

	request := model.SambaUser{}
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
	}

	if err := service.SetSambaPassword(username, request.Password); err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: username})
}

// DeleteSambaUser removes a share account, refusing while any share still names
// it. Removing it anyway would leave those shares listing a "valid users" entry
// that no longer resolves, which smbd treats as nobody being allowed in — the
// share would simply stop working with no explanation.
func DeleteSambaUser(ctx echo.Context) error {
	username := ctx.Param("username")

	for _, share := range service.MyService.Shares().GetSharesList() {
		if share.Username == username {
			return ctx.JSON(common_err.CLIENT_ERROR, model.Result{
				Success: common_err.CLIENT_ERROR,
				Message: "this account is still used by the share " + share.Path,
			})
		}
	}

	if err := service.DeleteSambaUser(username); err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: username})
}
