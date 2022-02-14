// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package packages

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/builder"
)

var (
	// ErrDuplicatePackageVersion indicates a duplicated package version error
	ErrDuplicatePackageVersion = errors.New("Package version does exist already")
	// ErrPackageVersionNotExist indicates a package version not exist error
	ErrPackageVersionNotExist = errors.New("Package version does not exist")
)

// EmptyVersionKey is a named constant for an empty version key
const EmptyVersionKey = ""

func init() {
	db.RegisterModel(new(PackageVersion))
}

// PackageVersion represents a package version
type PackageVersion struct {
	ID            int64 `xorm:"pk autoincr"`
	PackageID     int64 `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CreatorID     int64
	Version       string
	LowerVersion  string             `xorm:"UNIQUE(s) INDEX NOT NULL"`
	CompositeKey  string             `xorm:"UNIQUE(s) INDEX"`
	CreatedUnix   timeutil.TimeStamp `xorm:"created INDEX NOT NULL"`
	MetadataJSON  string             `xorm:"TEXT metadata_json"`
	DownloadCount int64
}

// GetOrInsertVersion inserts a version. If the same version exist already ErrDuplicatePackageVersion is returned
func GetOrInsertVersion(ctx context.Context, pv *PackageVersion) (*PackageVersion, error) {
	e := db.GetEngine(ctx)

	key := &PackageVersion{
		PackageID:    pv.PackageID,
		LowerVersion: pv.LowerVersion,
		CompositeKey: pv.CompositeKey,
	}

	has, err := e.Get(key)
	if err != nil {
		return nil, err
	}
	if has {
		return key, ErrDuplicatePackageVersion
	}
	if _, err = e.Insert(pv); err != nil {
		return nil, err
	}
	return pv, nil
}

// UpdateVersion updates a version
func UpdateVersion(pv *PackageVersion) error {
	_, err := db.GetEngine(db.DefaultContext).ID(pv.ID).Update(pv)
	return err
}

// IncrementDownloadCounter increments the download counter of a version
func IncrementDownloadCounter(versionID int64) error {
	_, err := db.GetEngine(db.DefaultContext).Exec("UPDATE `package_version` SET `download_count` = `download_count` + 1 WHERE `id` = ?", versionID)
	return err
}

// GetVersionByID gets a version by id
func GetVersionByID(ctx context.Context, versionID int64) (*PackageVersion, error) {
	pv := &PackageVersion{}

	has, err := db.GetEngine(ctx).ID(versionID).Get(pv)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrPackageNotExist
	}
	return pv, nil
}

// GetVersionByNameAndVersion gets a version by name and version number
func GetVersionByNameAndVersion(ctx context.Context, ownerID int64, packageType Type, name, version, key string) (*PackageVersion, error) {
	var cond builder.Cond = builder.Eq{
		"package.owner_id":   ownerID,
		"package.type":       packageType,
		"package.lower_name": strings.ToLower(name),
	}
	pv := &PackageVersion{
		LowerVersion: strings.ToLower(version),
		CompositeKey: key,
	}
	has, err := db.GetEngine(ctx).
		Join("INNER", "package", "package.id = package_version.package_id").
		Where(cond).
		Get(pv)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrPackageNotExist
	}

	return pv, nil
}

// GetVersionsByPackageType gets all versions of a specific type
func GetVersionsByPackageType(ownerID int64, packageType Type) ([]*PackageVersion, error) {
	var cond builder.Cond = builder.Eq{
		"package.owner_id": ownerID,
		"package.type":     packageType,
	}

	pvs := make([]*PackageVersion, 0, 10)
	return pvs, db.GetEngine(db.DefaultContext).
		Where(cond).
		Join("INNER", "package", "package.id = package_version.package_id").
		Find(&pvs)
}

// GetVersionsByPackageName gets all versions of a specific package
func GetVersionsByPackageName(ownerID int64, packageType Type, name string) ([]*PackageVersion, error) {
	var cond builder.Cond = builder.Eq{
		"package.owner_id":   ownerID,
		"package.type":       packageType,
		"package.lower_name": strings.ToLower(name),
	}

	pvs := make([]*PackageVersion, 0, 10)
	return pvs, db.GetEngine(db.DefaultContext).
		Where(cond).
		Join("INNER", "package", "package.id = package_version.package_id").
		Find(&pvs)
}

// GetVersionsByFilename gets all versions which are linked to a filename
func GetVersionsByFilename(ownerID int64, packageType Type, filename string) ([]*PackageVersion, error) {
	var cond builder.Cond = builder.Eq{
		"package.owner_id":        ownerID,
		"package.type":            packageType,
		"package_file.lower_name": strings.ToLower(filename),
	}

	pvs := make([]*PackageVersion, 0, 10)
	return pvs, db.GetEngine(db.DefaultContext).
		Where(cond).
		Join("INNER", "package_file", "package_file.version_id = package_version.id").
		Join("INNER", "package", "package.id = package_version.package_id").
		Find(&pvs)
}

// DeleteVersionByID deletes a version by id
func DeleteVersionByID(ctx context.Context, versionID int64) error {
	_, err := db.GetEngine(ctx).ID(versionID).Delete(&PackageVersion{})
	return err
}

// HasVersionFileReferences checks if there are associated files
func HasVersionFileReferences(ctx context.Context, versionID int64) (bool, error) {
	return db.GetEngine(ctx).Get(&PackageFile{
		VersionID: versionID,
	})
}

// PackageSearchOptions are options for SearchXXX methods
type PackageSearchOptions struct {
	OwnerID    int64
	RepoID     int64
	Type       string
	Query      string
	Properties map[string]string
	Sort       string
	db.Paginator
}

func (opts *PackageSearchOptions) toConds() builder.Cond {
	cond := builder.NewCond()

	if opts.OwnerID != 0 {
		cond = cond.And(builder.Eq{"package.owner_id": opts.OwnerID})
	}
	if opts.RepoID != 0 {
		cond = cond.And(builder.Eq{"package.repo_id": opts.RepoID})
	}
	if opts.Type != "" && opts.Type != "all" {
		cond = cond.And(builder.Eq{"package.type": opts.Type})
	}
	if opts.Query != "" {
		cond = cond.And(builder.Like{"package.lower_name", strings.ToLower(opts.Query)})
	}

	if len(opts.Properties) != 0 {
		var propsCond builder.Cond = builder.Eq{
			"package_property.ref_type": PropertyTypeVersion,
		}
		propsCond = propsCond.And(builder.Expr("package_property.ref_id = package_version.id"))

		propsCondBlock := builder.NewCond()
		for name, value := range opts.Properties {
			propsCondBlock = propsCondBlock.Or(builder.Eq{
				"package_property.name":  name,
				"package_property.value": value,
			})
		}
		propsCond = propsCond.And(propsCondBlock)

		cond = cond.And(builder.Eq{
			strconv.Itoa(len(opts.Properties)): builder.Select("COUNT(*)").Where(propsCond).From("package_property"),
		})
	}

	return cond
}

func (opts *PackageSearchOptions) configureOrderBy(e db.Engine) {
	switch opts.Sort {
	case "alphabetically":
		e.Asc("package.name")
	case "reversealphabetically":
		e.Desc("package.name")
	case "highestversion":
		e.Desc("package_version.version")
	case "lowestversion":
		e.Asc("package_version.version")
	case "oldest":
		e.Asc("package_version.created_unix")
	default:
		e.Desc("package_version.created_unix")
	}
}

// SearchVersions gets all versions of packages matching the search options
func SearchVersions(opts *PackageSearchOptions) ([]*PackageVersion, int64, error) {
	sess := db.GetEngine(db.DefaultContext).
		Where(opts.toConds()).
		Table("package_version").
		Join("INNER", "package", "package.id = package_version.package_id")

	opts.configureOrderBy(sess)

	if opts.Paginator != nil {
		sess = db.SetSessionPagination(sess, opts)
	}

	pvs := make([]*PackageVersion, 0, 10)
	count, err := sess.FindAndCount(&pvs)
	return pvs, count, err
}

// SearchLatestVersions gets the latest version of every package matching the search options
func SearchLatestVersions(opts *PackageSearchOptions) ([]*PackageVersion, int64, error) {
	cond := opts.toConds().
		And(builder.Expr("pv2.id IS NULL"))

	sess := db.GetEngine(db.DefaultContext).
		Table("package_version").
		Join("LEFT", "package_version pv2", "package_version.package_id = pv2.package_id AND package_version.created_unix < pv2.created_unix").
		Join("INNER", "package", "package.id = package_version.package_id").
		Where(cond)

	opts.configureOrderBy(sess)

	if opts.Paginator != nil {
		sess = db.SetSessionPagination(sess, opts)
	}

	pvs := make([]*PackageVersion, 0, 10)
	count, err := sess.FindAndCount(&pvs)
	return pvs, count, err
}

// FindVersionsByPropertyNameAndValue gets all package versions which are associated with a specific property + value
func FindVersionsByPropertyNameAndValue(ctx context.Context, packageID int64, name, value string) ([]*PackageVersion, error) {
	var cond builder.Cond = builder.Eq{
		"package_property.ref_type":  PropertyTypeVersion,
		"package_property.name":      name,
		"package_property.value":     value,
		"package_version.package_id": packageID,
	}

	pvs := make([]*PackageVersion, 0, 5)
	return pvs, db.GetEngine(ctx).
		Where(cond).
		Join("INNER", "package_property", "package_property.ref_id = package_version.id").
		Find(&pvs)
}
