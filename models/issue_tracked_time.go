// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"github.com/go-xorm/xorm"
	"time"
)

// TrackedTime represents a time that was spent for a specific issue.
type TrackedTime struct {
	ID          int64     `xorm:"pk autoincr" json:"id"`
	IssueID     int64     `xorm:"INDEX" json:"issue_id"`
	UserID      int64     `xorm:"INDEX" json:"user_id"`
	Created     time.Time `xorm:"-" json:"created"`
	CreatedUnix int64     `json:"-"`
	Time        int64     `json:"time"`
}

// AfterSet is invoked from XORM after setting the value of a field of this object.
func (t *TrackedTime) AfterSet(colName string, _ xorm.Cell) {
	switch colName {
	case "created_unix":
		t.Created = time.Unix(t.CreatedUnix, 0).Local()
	}
}

// GetTrackedTimesByIssue will return all tracked times that are part of the issue
func GetTrackedTimesByIssue(issueID int64) (trackedTimes []*TrackedTime, err error) {
	err = x.Where("issue_id = ?", issueID).Find(&trackedTimes)
	return
}

// GetTrackedTimesByUser will return all tracked times which are created by the user
func GetTrackedTimesByUser(userID int64) (trackedTimes []*TrackedTime, err error) {
	err = x.Where("user_id = ?", userID).Find(&trackedTimes)
	return
}

// BeforeInsert will be invoked by XORM before inserting a record
// representing this object.
func (t *TrackedTime) BeforeInsert() {
	t.CreatedUnix = time.Now().Unix()
}

// AddTime will add the given time (in seconds) to the issue
func AddTime(userID int64, issueID int64, time int64) error {
	tt := &TrackedTime{
		IssueID: issueID,
		UserID:  userID,
		Time:    time,
	}
	if _, err := x.Insert(tt); err != nil {
		return err
	}
	comment := &Comment{
		IssueID:  issueID,
		PosterID: userID,
		Type:     CommentTypeAddTimeManual,
		Content:  secToTime(time),
	}
	if _, err := x.Insert(comment); err != nil {
		return err
	}
	return nil
}

// TotalTimes returns the spent time for each user by an issue
func TotalTimes(issueID int64) (map[*User]string, error) {
	var trackedTimes []TrackedTime
	if err := x.
		Where("issue_id = ?", issueID).
		Find(&trackedTimes); err != nil {
		return nil, err
	}
	//Adding total time per user ID
	totalTimesByUser := make(map[int64]int64)
	for _, t := range trackedTimes {
		if total, ok := totalTimesByUser[t.UserID]; !ok {
			totalTimesByUser[t.UserID] = t.Time
		} else {
			totalTimesByUser[t.UserID] = total + t.Time
		}
	}

	totalTimes := make(map[*User]string)
	//Fetching User and making time human readable
	for userID, total := range totalTimesByUser {
		user, err := GetUserByID(userID)
		if err != nil || user == nil {
			continue
		}
		totalTimes[user] = secToTime(total)
	}
	return totalTimes, nil
}
