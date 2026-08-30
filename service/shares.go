/*
 * @Author: LinkLeong link@icewhale.org
 * @Date: 2022-07-26 11:21:14
 * @LastEditors: LinkLeong
 * @LastEditTime: 2022-08-18 11:16:25
 * @FilePath: /CasaOS/service/shares.go
 * @Description:
 * @Website: https://www.casaos.io
 * Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package service

import (
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils/command"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS/pkg/config"
	"github.com/IceWhaleTech/CasaOS/pkg/utils/file"
	"github.com/IceWhaleTech/CasaOS/service/model"
	model2 "github.com/IceWhaleTech/CasaOS/service/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SharesService interface {
	GetSharesList() (shares []model2.SharesDBModel)
	GetSharesByPath(path string) (shares []model2.SharesDBModel)
	GetSharesByName(name string) (shares []model2.SharesDBModel)
	CreateShare(share model2.SharesDBModel) error
	DeleteShare(id string) error
	UpdateConfigFile() error
	InitSambaConfig()
	DeleteShareByPath(path string) error
}

type sharesStruct struct {
	db *gorm.DB
}

func (s *sharesStruct) DeleteShareByPath(path string) error {
	s.db.Where("path LIKE ?", path+"%").Delete(&model.SharesDBModel{})

	return s.UpdateConfigFile()
}

func (s *sharesStruct) GetSharesByName(name string) (shares []model2.SharesDBModel) {
	s.db.Where("name = ?", name).Find(&shares)

	return
}

func (s *sharesStruct) GetSharesByPath(path string) (shares []model2.SharesDBModel) {
	s.db.Where("path = ?", path).Find(&shares)
	return
}

func (s *sharesStruct) GetSharesList() (shares []model2.SharesDBModel) {
	s.db.Find(&shares)
	return
}

func (s *sharesStruct) CreateShare(share model2.SharesDBModel) error {
	s.db.Create(&share)
	s.InitSambaConfig()

	return s.UpdateConfigFile()
}

func (s *sharesStruct) DeleteShare(id string) error {
	s.db.Where("id= ?", id).Delete(&model.SharesDBModel{})

	return s.UpdateConfigFile()
}

func (s *sharesStruct) UpdateConfigFile() error {
	shares := []model2.SharesDBModel{}
	s.db.Find(&shares)

	configStr := ""
	for _, share := range shares {
		configStr += sambaSection(share)
	}

	previous, previousErr := os.ReadFile(sambaShareConfigFile)

	file.WriteToPath([]byte(configStr), sambaConfigDir, sambaShareConfigName)

	if err := validateSambaConfig(); err != nil {
		// Put the working configuration back rather than leaving smbd pointed at
		// a file it will refuse on its next restart.
		if previousErr == nil {
			file.WriteToPath(previous, sambaConfigDir, sambaShareConfigName)
		}

		return err
	}

	// restart samba
	command.OnlyExec("source " + config.AppInfo.ShellPath + "/helper.sh ;RestartSMBD")

	return nil
}

func (s *sharesStruct) InitSambaConfig() {
	if file.Exists(sambaConfigFile) {
		str := file.ReadLine(1, sambaConfigFile)
		if strings.Contains(str, "# Copyright (c) 2021-2022 CasaOS Inc. All rights reserved.") {
			if err := migrateGuestMapping(); err != nil {
				logger.Error("failed to update the samba guest mapping", zap.Error(err))
			}

			return
		}
		file.MoveFile("/etc/samba/smb.conf", "/etc/samba/smb.conf.bak")
		smbConf := ""
		smbConf += `# Copyright (c) 2021-2022 CasaOS Inc. All rights reserved.
#
#
#                          ______     _______
#                        (  __  \   (  ___  )
#                        | (  \  )  | (   ) |
#                        | |   ) |  | |   | |
#                        | |   | |  | |   | |
#                        | |   ) |  | |   | |
#                        | (__/  )  | (___) |
#                        (______/   (_______)
#
#                   _          _______   _________
#                  ( (    /|  (  ___  )  \__   __/
#                  |  \  ( |  | (   ) |     ) (
#                  |   \ | |  | |   | |     | |
#                  | (\ \) |  | |   | |     | |
#                  | | \   |  | |   | |     | |
#                  | )  \  |  | (___) |     | |
#                  |/    )_)  (_______)     )_(
#
#   _______    _______    ______    _________   _______
#  (       )  (  ___  )  (  __  \   \__   __/  (  ____ \  |\     /|
#  | () () |  | (   ) |  | (  \  )     ) (     | (    \/  ( \   / )
#  | || || |  | |   | |  | |   ) |     | |     | (__       \ (_) /
#  | |(_)| |  | |   | |  | |   | |     | |     |  __)       \   /
#  | |   | |  | |   | |  | |   ) |     | |     | (           ) (
#  | )   ( |  | (___) |  | (__/  )  ___) (___  | )           | |
#  |/     \|  (_______)  (______/   \_______/  |/            \_/
#
#
# IMPORTANT: CasaOS will not provide technical support for any issues
#            caused by unauthorized modification to the configuration.

[global]
## fruit settings
   min protocol = SMB2
   ea support = yes
## vfs objects = fruit streams_xattr
   fruit:metadata = stream
   fruit:model = Macmini
   fruit:veto_appledouble = no
   fruit:posix_rename = yes
   fruit:zero_file_id = yes
   fruit:wipe_intentionally_left_blank_rfork = yes
   fruit:delete_empty_adfiles = yes
   multicast dns register = yes
   map to guest = never
   include=/etc/samba/smb.casa.conf`
		file.WriteToPath([]byte(smbConf), "/etc/samba", "smb.conf")
	}
}

func NewSharesService(db *gorm.DB) SharesService {
	return &sharesStruct{db: db}
}
