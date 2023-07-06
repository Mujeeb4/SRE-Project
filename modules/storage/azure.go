// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

var _ ObjectStorage = &AzureBlobStorage{}

type azureBlobObject struct {
	BlobClient *blob.Client
	Context    *context.Context
	Name       string
	Size       int64
	ModTime    *time.Time

	Offset int64
}

func (a *azureBlobObject) downloadStream(p []byte) (int, error) {
	if a.Offset > a.Size {
		return 0, io.EOF
	}
	count := a.Size - a.Offset
	pl := int64(len(p))
	if pl > count {
		p = p[0:count]
	} else {
		count = pl
	}
	res, err := a.BlobClient.DownloadStream(*a.Context, &blob.DownloadStreamOptions{
		Range: blob.HTTPRange{
			Offset: a.Offset,
			Count:  count,
		},
	})
	if err != nil {
		return 0, convertAzureBlobErr(err)
	}
	a.Offset += count

	buf := bytes.NewBuffer(p)
	c, err := io.Copy(buf, res.Body)
	return int(c), err
}

func (a *azureBlobObject) Close() error {
	return nil
}

func (a *azureBlobObject) Read(p []byte) (int, error) {
	c, err := a.downloadStream(p)
	return c, err
}

func (a *azureBlobObject) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errors.New("Seek: invalid whence")
	case io.SeekStart:
		offset += 0
	case io.SeekCurrent:
		offset += a.Offset
	case io.SeekEnd:
		offset += a.Size
	}
	if offset < 0 {
		return 0, errors.New("Seek: invalid offset")
	}
	a.Offset = offset
	return offset, nil
}

func (a *azureBlobObject) Stat() (os.FileInfo, error) {
	return &azureBlobFileInfo{
		a.Name,
		a.Size,
		*a.ModTime,
	}, nil
}

// AzureStorage returns a azure blob storage
type AzureBlobStorage struct {
	cfg       *setting.AzureBlobStorageConfig
	ctx       context.Context
	client    *azblob.Client
	container string
	basePath  string
}

func convertAzureBlobErr(err error) error {
	if err == nil {
		return nil
	}

	if bloberror.HasCode(err, bloberror.BlobNotFound) {
		return os.ErrNotExist
	}
	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return nil
	}
	return fmt.Errorf(respErr.ErrorCode)
}

// NewAzureBlobStorage returns a azure blob storage
func NewAzureBlobStorage(ctx context.Context, cfg *setting.Storage) (ObjectStorage, error) {
	config := cfg.AzureBlobConfig

	log.Info("Creating Azure Blob storage at %s:%s with base path %s", config.Endpoint, config.Container, config.BasePath)

	cred, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, convertAzureBlobErr(err)
	}
	client, err := azblob.NewClientWithSharedKeyCredential(config.Endpoint, cred, &azblob.ClientOptions{})
	if err != nil {
		return nil, convertAzureBlobErr(err)
	}

	_, err = client.CreateContainer(ctx, config.Container, &container.CreateOptions{})
	if err != nil {
		// Check to see if we already own this container (which happens if you run this twice)
		if !bloberror.HasCode(err, bloberror.ContainerAlreadyExists) {
			return nil, convertMinioErr(err)
		}
	}

	return &AzureBlobStorage{
		cfg:       &config,
		ctx:       ctx,
		client:    client,
		container: config.Container,
		basePath:  config.BasePath,
	}, nil
}

func (a *AzureBlobStorage) buildAzureBlobPath(p string) string {
	p = util.PathJoinRelX(a.basePath, p)
	if p == "." || p == "/" {
		p = "" // azure uses prefix, so path should be empty as relative path
	}
	return p
}

func (a *AzureBlobStorage) getObjectNameFromPath(path string) string {
	s := strings.Split(path, "/")
	return s[len(s)-1]
}

// Open opens a file
func (a *AzureBlobStorage) Open(path string) (Object, error) {
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(a.buildAzureBlobPath(path))
	res, err := blobClient.GetProperties(a.ctx, &blob.GetPropertiesOptions{})
	if err != nil {
		return nil, convertAzureBlobErr(err)
	}
	return &azureBlobObject{
		Context:    &a.ctx,
		BlobClient: blobClient,
		Name:       a.getObjectNameFromPath(path),
		Size:       *res.ContentLength,
		ModTime:    res.LastModified,
	}, nil
}

// Save saves a file to azure blob storage
func (a *AzureBlobStorage) Save(path string, r io.Reader, size int64) (int64, error) {
	_, err := a.client.UploadStream(
		a.ctx,
		a.container,
		a.buildAzureBlobPath(path),
		r,
		&blockblob.UploadStreamOptions{},
	)
	if err != nil {
		return 0, convertAzureBlobErr(err)
	}
	return size, nil
}

type azureBlobFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (a azureBlobFileInfo) Name() string {
	return path.Base(a.name)
}

func (a azureBlobFileInfo) Size() int64 {
	return a.size
}

func (a azureBlobFileInfo) ModTime() time.Time {
	return a.modTime
}

func (a azureBlobFileInfo) IsDir() bool {
	return strings.HasSuffix(a.name, "/")
}

func (a azureBlobFileInfo) Mode() os.FileMode {
	return os.ModePerm
}

func (a azureBlobFileInfo) Sys() interface{} {
	return nil
}

// Stat returns the stat information of the object
func (a *AzureBlobStorage) Stat(path string) (os.FileInfo, error) {
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(a.buildAzureBlobPath(path))
	res, err := blobClient.GetProperties(a.ctx, &blob.GetPropertiesOptions{})
	if err != nil {
		return nil, convertAzureBlobErr(err)
	}
	s := strings.Split(path, "/")
	return &azureBlobFileInfo{
		s[len(s)-1],
		*res.ContentLength,
		*res.LastModified,
	}, nil
}

// Delete delete a file
func (a *AzureBlobStorage) Delete(path string) error {
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(a.buildAzureBlobPath(path))
	_, err := blobClient.Delete(a.ctx, &blob.DeleteOptions{})
	return convertAzureBlobErr(err)
}

// URL gets the redirect URL to a file. The presigned link is valid for 5 minutes.
func (a *AzureBlobStorage) URL(path, name string) (*url.URL, error) {
	return nil, ErrURLNotSupported
}

// IterateObjects iterates across the objects in the azureblobstorage
func (a *AzureBlobStorage) IterateObjects(dirName string, fn func(path string, obj Object) error) error {
	dirName = a.buildAzureBlobPath(dirName)
	if dirName != "" {
		dirName += "/"
	}
	pager := a.client.NewListBlobsFlatPager(a.container, &container.ListBlobsFlatOptions{
		Prefix: &dirName,
	})
	for pager.More() {
		resp, err := pager.NextPage(a.ctx)
		if err != nil {
			return convertAzureBlobErr(err)
		}
		for _, object := range resp.Segment.BlobItems {
			fullPath := a.buildAzureBlobPath(*object.Name)
			blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(fullPath)
			object := &azureBlobObject{
				Context:    &a.ctx,
				BlobClient: blobClient,
				Name:       fullPath,
				Size:       *object.Properties.ContentLength,
				ModTime:    object.Properties.LastModified,
			}
			if err := func(object *azureBlobObject, fn func(path string, obj Object) error) error {
				defer object.Close()
				return fn(strings.TrimPrefix(object.Name, a.basePath), object)
			}(object, fn); err != nil {
				return convertAzureBlobErr(err)
			}
		}
	}
	return nil
}

func init() {
	RegisterStorageType(setting.AzureBlobStorageType, NewAzureBlobStorage)
}
