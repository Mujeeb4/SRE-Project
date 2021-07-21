---
date: "2021-07-20T00:00:00+00:00"
title: "Generic Packages Repository"
slug: "generic"
draft: false
toc: false
menu:
  sidebar:
    parent: "packages"
    name: "Generic"
    weight: 10
    identifier: "generic"
---

# Generic Packages Repository

Publish generic files, like release binaries or other output, in your project’s Package Registry.

**Table of Contents**

{{< toc >}}

## Authenticate to the package registry

To authenticate to the Package Registry, you need to provide [custom HTTP headers or use HTTP Basic authentication]({{< relref "doc/developers/api-usage.en-us.md#authentication" >}}).

## Publish a package

To publish a generic package perform a HTTP PUT operation with the package content in the request body.
You cannot publish a package if a package of the same name and version already exists. You must delete the existing package first.

```
PUT https://gitea.example.com/api/v1/repos/{owner}/{repository}/packages/generic/{package_name}/{package_version}/{file_name}
```

| Parameter         | Description |
| ----------------- | ----------- |
| `owner`           | The owner of the repository. |
| `repository`      | The name of the repository. |
| `package_name`    | The package name. It can contain only lowercase letters (`a-z`), uppercase letter (`A-Z`), numbers (`0-9`), dots (`.`), hyphens (`-`), or underscores (`_`). |
| `package_version` | The package version as described in the [SemVer](https://semver.org/) spec. |
| `file_name`       | The filename. It can contain only lowercase letters (`a-z`), uppercase letter (`A-Z`), numbers (`0-9`), dots (`.`), hyphens (`-`), or underscores (`_`). |

Example request using HTTP Basic authentication:

```shell
curl --user your_username:your_password_or_token \
     --upload-file path/to/file.bin \
     "https://gitea.example.com/api/v1/repos/testuser/test-repository/packages/generic/test_package/1.0.0/file.bin"
```

The server reponds with the following HTTP Status codes.

| HTTP Status Code  | Meaning |
| ----------------- | ------- |
| `201 Created`     | The package has been published. |
| `400 Bad Request` | The package name and/or version are invalid or a package with the same name and version already exist. |

## Download a package

To download a generic package perform a HTTP GET operation.

```
GET https://gitea.example.com/api/v1/repos/{owner}/{repository}/packages/generic/{package_name}/{package_version}/{file_name}
```

| Parameter         | Description |
| ----------------- | ----------- |
| `owner`           | The owner of the repository. |
| `repository`      | The name of the repository. |
| `package_name`    | The package name. |
| `package_version` | The package version. |
| `file_name`       | The filename. |

The file content is served in the response body. The response content type is `application/octet-stream`.

Example request using HTTP Basic authentication:

```shell
curl --user your_username:your_token_or_password \
     "https://gitea.example.com/api/v1/repos/testuser/test-repository/packages/generic/test_package/1.0.0/file.bin"
```
