// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"time"
)

func ListSundaysBetween(startStr, endStr string) ([]int64, error) {
	layout := "2006-01-02"
	startDate, err := time.Parse(layout, startStr)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse(layout, endStr)
	if err != nil {
		return nil, err
	}

	// Ensure the start date is a Sunday
	for startDate.Weekday() != time.Sunday {
		startDate = startDate.AddDate(0, 0, 1)
	}

	var sundays []int64

	// Iterate from start date to end date and find all Sundays
	for currentDate := startDate; currentDate.Before(endDate); currentDate = currentDate.AddDate(0, 0, 7) {
		sundays = append(sundays, currentDate.UnixMilli())
	}

	return sundays, nil
}

func FindLastSundayBeforeDate(dateStr string) (string, error) {
	layout := "2006-01-02"

	date, err := time.Parse(layout, dateStr)
	if err != nil {
		return "", err
	}

	weekday := date.Weekday()
	daysToSubtract := int(weekday) - int(time.Sunday)
	if daysToSubtract < 0 {
		daysToSubtract += 7
	}

	last_sunday := date.AddDate(0, 0, -daysToSubtract)
	return last_sunday.Format(layout), nil
}

func FindFirstSundayAfterDate(dateStr string) (string, error) {
	layout := "2006-01-02"

	date, err := time.Parse(layout, dateStr)
	if err != nil {
		return "", err
	}

	weekday := date.Weekday()
	daysToAdd := int(time.Sunday) - int(weekday)
	if daysToAdd <= 0 {
		daysToAdd += 7
	}

	first_sunday := date.AddDate(0, 0, daysToAdd)
	return first_sunday.Format(layout), nil
}
