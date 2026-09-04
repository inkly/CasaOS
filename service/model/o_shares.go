/*
 * @Author: LinkLeong link@icewhale.org
 * @Date: 2022-07-26 11:17:17
 * @LastEditors: LinkLeong
 * @LastEditTime: 2022-07-27 15:25:07
 * @FilePath: /CasaOS/service/model/o_shares.go
 * @Description:
 * @Website: https://www.casaos.io
 * Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package model

type SharesDBModel struct {
	ID        uint `gorm:"column:id;primary_key" json:"id"`
	Anonymous bool `json:"anonymous"`
	// Username is the Samba account allowed to mount the share. Empty means the
	// share is open to guests, which is what every row created before
	// authenticated shares existed carries, so an upgrade leaves them untouched.
	Username string `json:"username"`
	// TimeMachine adds Apple's SMB extensions to this share's section so macOS
	// offers it as a Time Machine destination. AutoMigrate adds the column, and
	// rows predating it read back as false, i.e. the section they already had.
	TimeMachine bool   `json:"time_machine"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Updated     int64  `gorm:"autoUpdateTime"`
	Created     int64  `gorm:"autoCreateTime"`
}

func (p *SharesDBModel) TableName() string {
	return "o_shares"
}
