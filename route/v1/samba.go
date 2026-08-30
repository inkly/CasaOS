/*
 * @Author: LinkLeong link@icewhale.com
 * @Date: 2022-07-26 11:08:48
 * @LastEditors: LinkLeong
 * @LastEditTime: 2022-08-17 18:25:42
 * @FilePath: /CasaOS/route/v1/samba.go
 * @Description:
 * @Website: https://www.casaos.io
 * Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package v1

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/pkg/samba"
	"github.com/IceWhaleTech/CasaOS/pkg/utils/common_err"
	"github.com/IceWhaleTech/CasaOS/pkg/utils/file"
	"github.com/IceWhaleTech/CasaOS/service"
	model2 "github.com/IceWhaleTech/CasaOS/service/model"
)

// service

func GetSambaStatus(ctx echo.Context) error {
	if status, err := systemctl.IsServiceRunning("smbd"); err != nil || !status {
		return ctx.JSON(http.StatusInternalServerError, model.Result{
			Success: common_err.SERVICE_NOT_RUNNING,
			Message: common_err.GetMsg(common_err.SERVICE_NOT_RUNNING),
		})
	}

	needInit := true
	if file.Exists("/etc/samba/smb.conf") {
		str := file.ReadLine(1, "/etc/samba/smb.conf")
		if strings.Contains(str, "# Copyright (c) 2021-2022 CasaOS Inc. All rights reserved.") {
			needInit = false
		}
	}
	data := make(map[string]string, 1)
	data["need_init"] = fmt.Sprintf("%v", needInit)
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

func GetSambaSharesList(ctx echo.Context) error {
	shares := service.MyService.Shares().GetSharesList()
	shareList := []model.Shares{}
	for _, v := range shares {
		shareList = append(shareList, model.Shares{
			Anonymous: v.Anonymous,
			Path:      v.Path,
			ID:        v.ID,
			Username:  v.Username,
		})
	}
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: shareList})
}

// restrictShareToUser hands the shared directory to the account allowed to mount
// it.
//
// Without this the directory stays root-owned, while the Samba session runs as
// that account: the share would authenticate correctly and then refuse every
// write, which reads as a broken feature rather than a permissions problem.
func restrictShareToUser(path, username string) error {
	account, err := user.Lookup(username)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(account.Gid)
	if err != nil {
		return err
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return err
	}

	return os.Chmod(path, 0o770)
}

func PostSambaSharesCreate(ctx echo.Context) error {
	shares := []model.Shares{}
	if err := ctx.Bind(&shares); err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
	}

	for _, v := range shares {
		if v.Path == "" {
			return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INSUFFICIENT_PERMISSIONS, Message: common_err.GetMsg(common_err.INSUFFICIENT_PERMISSIONS)})
		}
		if !file.Exists(v.Path) {
			return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DIR_NOT_EXISTS, Message: common_err.GetMsg(common_err.DIR_NOT_EXISTS)})
		}
		if len(service.MyService.Shares().GetSharesByPath(v.Path)) > 0 {
			return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.SHARE_ALREADY_EXISTS, Message: common_err.GetMsg(common_err.SHARE_ALREADY_EXISTS)})
		}
		// Sections in smb.casa.conf are named after the directory, and smbd keeps
		// only the first of two sections sharing a name. This used to query the
		// path column with a bare basename, so it never matched and the second
		// share silently vanished from the generated file.
		if len(service.MyService.Shares().GetSharesByName(filepath.Base(v.Path))) > 0 {
			return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.SHARE_NAME_ALREADY_EXISTS, Message: common_err.GetMsg(common_err.SHARE_NAME_ALREADY_EXISTS)})
		}

		// A share the client asked to protect needs an account to protect it with,
		// and that account must already exist. Creating one here would mean
		// inventing a password nobody was ever told.
		if !v.Anonymous {
			if err := service.ValidateSambaUsername(v.Username); err != nil {
				return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: err.Error()})
			}
			if _, err := user.Lookup(v.Username); err != nil {
				return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
			}
		}
	}

	for _, v := range shares {
		shareDBModel := model2.SharesDBModel{
			Anonymous: v.Anonymous,
			Path:      v.Path,
			Name:      filepath.Base(v.Path),
		}

		if v.Anonymous {
			// Unchanged from before authenticated shares existed: a guest share has
			// no owner to speak of, so the directory stays world-writable.
			os.Chmod(v.Path, 0o777)
		} else {
			shareDBModel.Username = v.Username

			if err := restrictShareToUser(v.Path, v.Username); err != nil {
				return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			}
		}

		if err := service.MyService.Shares().CreateShare(shareDBModel); err != nil {
			return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: shares})
}

func DeleteSambaShares(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INSUFFICIENT_PERMISSIONS, Message: common_err.GetMsg(common_err.INSUFFICIENT_PERMISSIONS)})
	}
	if err := service.MyService.Shares().DeleteShare(id); err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: id})
}

// client
func GetSambaConnectionsList(ctx echo.Context) error {
	connections := service.MyService.Connections().GetConnectionsList()
	connectionList := []model.Connections{}
	for _, v := range connections {
		connectionList = append(connectionList, model.Connections{
			ID:         v.ID,
			Username:   v.Username,
			Port:       v.Port,
			Host:       v.Host,
			MountPoint: v.MountPoint,
		})
	}
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: connectionList})
}

func PostSambaConnectionsCreate(ctx echo.Context) error {
	connection := model.Connections{}
	ctx.Bind(&connection)
	if connection.Port == "" {
		connection.Port = "445"
	}
	if connection.Username == "" || connection.Host == "" {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CHARACTER_LIMIT, Message: common_err.GetMsg(common_err.CHARACTER_LIMIT)})
	}

	// if ok, _ := regexp.MatchString(`^[\w@#*.]{4,30}$`, connection.Password); !ok {
	// 	return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CHARACTER_LIMIT, Message: common_err.GetMsg(common_err.CHARACTER_LIMIT)})
	// 	return
	// }
	// if ok, _ := regexp.MatchString(`^[\w@#*.]{4,30}$`, connection.Username); !ok {
	// 	return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	// 	return
	// }
	// if !ip_helper.IsIPv4(connection.Host) && !ip_helper.IsIPv6(connection.Host) {
	// 	return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	// 	return
	// }
	// if ok, _ := regexp.MatchString("^[0-9]{1,6}$", connection.Port); !ok {
	// 	return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	// 	return
	// }

	connection.Host = strings.Split(connection.Host, "/")[0]
	// check is exists
	connections := service.MyService.Connections().GetConnectionByHost(connection.Host)
	if len(connections) > 0 {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.Record_ALREADY_EXIST, Message: common_err.GetMsg(common_err.Record_ALREADY_EXIST), Data: common_err.GetMsg(common_err.Record_ALREADY_EXIST)})
	}
	// check connect is ok
	directories, err := samba.GetSambaSharesList(connection.Host, connection.Port, connection.Username, connection.Password)
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	}

	connectionDBModel := model2.ConnectionsDBModel{}
	connectionDBModel.Username = connection.Username
	connectionDBModel.Password = connection.Password
	connectionDBModel.Host = connection.Host
	connectionDBModel.Port = connection.Port
	connectionDBModel.Directories = strings.Join(directories, ",")
	baseHostPath := "/mnt/" + connection.Host
	connectionDBModel.MountPoint = baseHostPath
	connection.MountPoint = baseHostPath
	file.IsNotExistMkDir(baseHostPath)
	for _, v := range directories {
		mountPoint := baseHostPath + "/" + v
		file.IsNotExistMkDir(mountPoint)
		service.MyService.Connections().MountSmaba(connectionDBModel.Username, connectionDBModel.Host, v, connectionDBModel.Port, mountPoint, connectionDBModel.Password)
	}

	service.MyService.Connections().CreateConnection(&connectionDBModel)

	connection.ID = connectionDBModel.ID
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: connection})
}

func DeleteSambaConnections(ctx echo.Context) error {
	id := ctx.Param("id")
	connection := service.MyService.Connections().GetConnectionByID(id)
	if connection.Username == "" {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.Record_NOT_EXIST, Message: common_err.GetMsg(common_err.Record_NOT_EXIST)})
	}
	mountPointList, err := samba.GetSambaSharesList(connection.Host, connection.Port, connection.Username, connection.Password)
	// mountPointList, err := service.MyService.System().GetDirPath(connection.MountPoint)
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	}
	baseHostPath := "/mnt/" + connection.Host
	for _, v := range mountPointList {
		if service.IsMounted(baseHostPath + "/" + v) {
			err := service.MyService.Connections().UnmountSmaba(baseHostPath + "/" + v)
			if err != nil {
				logger.Error("unmount smaba error", zap.Error(err), zap.Any("path", baseHostPath+"/"+v))
				return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
			}
		}
	}
	dir, _ := ioutil.ReadDir(connection.MountPoint)
	if len(dir) == 0 {
		os.RemoveAll(connection.MountPoint)
	}
	service.MyService.Connections().DeleteConnection(id)
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: id})
}
