// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"database/sql"

	"gorm.io/datatypes"
)

type Status string

const (
	Uninstalled Status = "uninstalled"
	Installed   Status = "installed"
	ToUpgrade   Status = "to upgrade"
	ToInstall   Status = "to install"
)

type Module struct {
	BaseModel     `gorm:"embedded"`
	ApplicationId sql.NullString `gorm:"type:char(20)"`
	Application   *Application   `gorm:"foreignKey:ApplicationId;"`
	Models        []*Model       `gorm:"foreignKey:ModuleId;constraint:OnDelete:CASCADE;" json:"models"`
	Components    []*Component   `gorm:"foreignKey:ModuleId;constraint:OnDelete:CASCADE;" json:"components"`
	UiResources   []*UiResource  `gorm:"foreignKey:ModuleId;constraint:OnDelete:CASCADE;" json:"uiResources"`
	Category      string         `gorm:"type:varchar(255);index" json:"category"`

	Dependencies []*Module `gorm:"many2many:meta_module_dependencies;joinForeignKey:ModuleId;joinReferences:DependModuleId;constraint:OnDelete:CASCADE;"`
	Dependents   []*Module `gorm:"many2many:meta_module_dependencies;joinForeignKey:DependModuleId;joinReferences:ModuleId;constraint:OnDelete:CASCADE;"`

	// module fields
	Name                 string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"module_name"`
	ShortDesc            string         `gorm:"type:text;" json:"name"`
	Version              string         `gorm:"type:varchar(255);" json:"version"`
	Tarball              string         `gorm:"type:varchar(255);" json:"tarball"`
	Integrity            string         `gorm:"-" json:"integrity,omitempty"`
	Summary              string         `gorm:"type:text;" json:"summary"`
	Description          string         `gorm:"type:text;" json:"description"`
	ApplicationStr       string         `gorm:"type:varchar(255);" json:"application"`
	EntryPoints          datatypes.JSON `json:"entryPoints"`
	WebEntryPoint        string         `gorm:"type:varchar(512);"`
	ServiceEntryPoint    string         `gorm:"type:varchar(512);"`
	DependsStr           datatypes.JSON `json:"depends"`
	DataStr              datatypes.JSON `json:"data"`
	DemoStr              datatypes.JSON `json:"demo"`
	ExternalDependencies datatypes.JSON `json:"externalDependencies"`
	Author               string         `gorm:"type:varchar(255);" json:"author"`
	License              string         `gorm:"type:text;" json:"license"`
	Homepage             string         `gorm:"type:text;" json:"homepage"`
	Repository           string         `gorm:"type:text;" json:"repository"`
	Path                 string         `gorm:"type:varchar(512);" json:"path"`
	Status               Status         `json:"-"`
}

func (mod *Module) TableName() string {
	return "meta_module"
}
