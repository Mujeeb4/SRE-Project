// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// Label a label to an issue or a pr
// swagger:model
type Label struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// example: false
	Exclusive bool `json:"exclusive"`
	// example: 00aabb
	Color       string `json:"color"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// CreateLabelOption options for creating a label
type CreateLabelOption struct {
	// required:true
	Name string `json:"name" binding:"Required"`
	// example: false
	Exclusive bool `json:"exclusive"`
	// required:true
	// example: #00aabb
	Color       string `json:"color" binding:"Required"`
	Description string `json:"description"`
}

// EditLabelOption options for editing a label
type EditLabelOption struct {
	Name *string `json:"name"`
	// example: false
	Exclusive *bool `json:"exclusive"`
	// example: #00aabb
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

// IssueLabelsOption a collection of labels
type IssueLabelsOption struct {
	// list of label IDs
	Labels []int64 `json:"labels"`
}

// LabelTemplate info of a Label template
type LabelTemplate struct {
	Name string `json:"name"`
	// example: false
	Exclusive bool `json:"exclusive"`
	// example: 00aabb
	Color       string `json:"color"`
	Description string `json:"description"`
}
