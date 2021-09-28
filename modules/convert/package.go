// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package convert

import (
	"code.gitea.io/gitea/models/packages"
	api "code.gitea.io/gitea/modules/structs"
)

// ToPackage convert a packages.Package to api.Package
func ToPackage(p *packages.Package) *api.Package {
	if err := p.LoadCreator(); err != nil {
		return &api.Package{}
	}

	return &api.Package{
		ID:        p.ID,
		Creator:   ToUser(p.Creator, nil),
		Type:      p.Type.String(),
		Name:      p.Name,
		Version:   p.Version,
		CreatedAt: p.CreatedUnix.AsTime(),
		UpdatedAt: p.CreatedUnix.AsTime(),
	}
}

// ToPackageFile converts packages.PackageFile to api.PackageFile
func ToPackageFile(pf *packages.PackageFile) *api.PackageFile {
	return &api.PackageFile{
		ID:         pf.ID,
		Size:       pf.Size,
		Name:       pf.Name,
		HashMD5:    pf.HashMD5,
		HashSHA1:   pf.HashSHA1,
		HashSHA256: pf.HashSHA256,
		HashSHA512: pf.HashSHA512,
		CreatedAt:  pf.CreatedUnix.AsTime(),
		UpdatedAt:  pf.UpdatedUnix.AsTime(),
	}
}
