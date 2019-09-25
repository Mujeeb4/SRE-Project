// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package references

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"code.gitea.io/gitea/modules/markup/mdstripper"
	"code.gitea.io/gitea/modules/setting"
)

var (
	// validNamePattern performs only the most basic validation for user or repository names
	// Repository name should contain only alphanumeric, dash ('-'), underscore ('_') and dot ('.') characters.
	validNamePattern = regexp.MustCompile(`^[a-z0-9_.-]+$`)

	// mentionPattern matches all mentions in the form of "@user"
	mentionPattern = regexp.MustCompile(`(?:\s|^|\(|\[)(@[0-9a-zA-Z-_\.]+)(?:\s|$|\)|\])`)
	// issueNumericPattern matches string that references to a numeric issue, e.g. #1287
	issueNumericPattern = regexp.MustCompile(`(?:\s|^|\(|\[)(#[0-9]+)(?:\s|$|\)|\]|:|\.(\s|$))`)
	// issueAlphanumericPattern matches string that references to an alphanumeric issue, e.g. ABC-1234
	issueAlphanumericPattern = regexp.MustCompile(`(?:\s|^|\(|\[)([A-Z]{1,10}-[1-9][0-9]*)(?:\s|$|\)|\]|:|\.(\s|$))`)
	// crossReferenceIssueNumericPattern matches string that references a numeric issue in a different repository
	// e.g. gogits/gogs#12345
	crossReferenceIssueNumericPattern = regexp.MustCompile(`(?:\s|^|\(|\[)([0-9a-zA-Z-_\.]+/[0-9a-zA-Z-_\.]+#[0-9]+)(?:\s|$|\)|\]|\.(\s|$))`)

	// Same as GitHub. See
	// https://help.github.com/articles/closing-issues-via-commit-messages
	issueCloseKeywords  = []string{"close", "closes", "closed", "fix", "fixes", "fixed", "resolve", "resolves", "resolved"}
	issueReopenKeywords = []string{"reopen", "reopens", "reopened"}

	issueCloseKeywordsPat, issueReopenKeywordsPat *regexp.Regexp
)

// XRefAction represents the kind of effect a cross reference has once is resolved
type XRefAction int64

const (
	// XRefActionNone means the cross-reference is simply a comment
	XRefActionNone XRefAction = iota // 0
	// XRefActionCloses means the cross-reference should close an issue if it is resolved
	XRefActionCloses // 1
	// XRefActionReopens means the cross-reference should reopen an issue if it is resolved
	XRefActionReopens // 2
	// XRefActionNeutered means the cross-reference will no longer affect the source
	XRefActionNeutered // 3
)

// IssueReference contains an unverified cross-reference to a local issue/pull request
type IssueReference struct {
	Index  int64
	Owner  string
	Name   string
	Action XRefAction
}

// RenderizableReference contains an unverified cross-reference to with rendering information
type RenderizableReference interface {
	Issue() string
	Owner() string
	Name() string
	RefLocation() *ReferenceLocation
	Action() XRefAction
	ActionLocation() *ReferenceLocation
}

type rawReference struct {
	index          int64
	owner          string
	name           string
	action         XRefAction
	issue          string
	refLocation    *ReferenceLocation
	actionLocation *ReferenceLocation
}

// Index returns the number of the issue/pull request
func (r *rawReference) Index() int64 {
	return r.index
}

// Owner returns the owner of the repository for the issue/pull request
func (r *rawReference) Owner() string {
	return r.owner
}

// Owner returns the name of the repository for the issue/pull request
func (r *rawReference) Name() string {
	return r.name
}

// Issue returns the ID of the issue/pull request
func (r *rawReference) Issue() string {
	return r.issue
}

// RefLocation returns the location of the reference in the originating string
func (r *rawReference) RefLocation() *ReferenceLocation {
	return r.refLocation
}

// Action returns the action represented by the action keyword found preceeding the reference
func (r *rawReference) Action() XRefAction {
	return r.action
}

// RefLocation returns the location of the action keyword in the originating string
func (r *rawReference) ActionLocation() *ReferenceLocation {
	return r.actionLocation
}

func rawToIssueReferenceList(reflist []*rawReference) []IssueReference {
	refarr := make([]IssueReference, len(reflist))
	for i, r := range reflist {
		refarr[i] = IssueReference{
			Index:  r.index,
			Owner:  r.owner,
			Name:   r.name,
			Action: r.action,
		}
	}
	return refarr
}

// ReferenceLocation is the position where the reference was found within the parsed text
type ReferenceLocation struct {
	Start int
	End   int
}

func makeKeywordsPat(keywords []string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(?:\s|^|\(|\[)(` + strings.Join(keywords, `|`) + `):? $`)
}

func init() {
	issueCloseKeywordsPat = makeKeywordsPat(issueCloseKeywords)
	issueReopenKeywordsPat = makeKeywordsPat(issueReopenKeywords)
}

// FindAllMentionsMarkdown matches mention patterns in given content and
// returns a list of found unvalidated user names **not including** the @ prefix.
func FindAllMentionsMarkdown(content string) []string {
	bcontent, _ := mdstripper.StripMarkdownBytes([]byte(content))
	locations := FindAllMentionsBytes(bcontent)
	mentions := make([]string, len(locations))
	for i, val := range locations {
		mentions[i] = string(bcontent[val.Start+1 : val.End])
	}
	return mentions
}

// FindAllMentionsBytes matches mention patterns in given content
// and returns a list of found unvalidated user names including the @ prefix.
func FindAllMentionsBytes(content []byte) []ReferenceLocation {
	mentions := mentionPattern.FindAllSubmatchIndex(content, -1)
	ret := make([]ReferenceLocation, len(mentions))
	for i, val := range mentions {
		ret[i] = ReferenceLocation{Start: val[2], End: val[3]}
	}
	return ret
}

// FindFirstMentionBytes matches mention patterns in given content
// and returns a list of found unvalidated user names including the @ prefix.
func FindFirstMentionBytes(content []byte) (bool, ReferenceLocation) {
	mention := mentionPattern.FindSubmatchIndex(content)
	if mention == nil {
		return false, ReferenceLocation{}
	}
	return true, ReferenceLocation{Start: mention[2], End: mention[3]}
}

// FindAllIssueReferencesMarkdown strips content from markdown markup
// and returns a list of unvalidated references found in it.
func FindAllIssueReferencesMarkdown(content string) []IssueReference {
	return rawToIssueReferenceList(findAllIssueReferencesMarkdown(content))
}

func findAllIssueReferencesMarkdown(content string) []*rawReference {
	bcontent, links := mdstripper.StripMarkdownBytes([]byte(content))
	return findAllIssueReferencesBytes(bcontent, links)
}

// FindAllIssueReferences returns a list of unvalidated references found in a string.
func FindAllIssueReferences(content string) []IssueReference {
	return rawToIssueReferenceList(findAllIssueReferencesBytes([]byte(content), []string{}))
}

// FindRenderizableReferenceNumeric returns the first unvalidated references found in a byte slice
func FindRenderizableReferenceNumeric(content string) (bool, RenderizableReference) {
	match := issueNumericPattern.FindStringSubmatchIndex(content)
	if match == nil {
		if match = crossReferenceIssueNumericPattern.FindStringSubmatchIndex(content); match == nil {
			return false, nil
		}
	}

	return true, getCrossReference([]byte(content), match[2], match[3], false)
}

// FindRenderizableReferenceAlphanumeric returns the first alphanumeric unvalidated references found in a byte slice
func FindRenderizableReferenceAlphanumeric(content string) (bool, RenderizableReference) {
	match := issueAlphanumericPattern.FindStringSubmatchIndex(content)
	if match == nil {
		return false, nil
	}

	action, location := findActionKeywords([]byte(content), match[2])
	return true, &rawReference{
		issue:          string(content[match[2]:match[3]]),
		refLocation:    &ReferenceLocation{Start: match[2], End: match[3]},
		action:         action,
		actionLocation: location,
	}
}

// FindAllIssueReferencesBytes returns a list of unvalidated references found in a byte slice.
func findAllIssueReferencesBytes(content []byte, links []string) []*rawReference {

	ret := make([]*rawReference, 0, 10)

	matches := issueNumericPattern.FindAllSubmatchIndex(content, -1)
	for _, match := range matches {
		if ref := getCrossReference(content, match[2], match[3], false); ref != nil {
			ret = append(ret, ref)
		}
	}

	matches = crossReferenceIssueNumericPattern.FindAllSubmatchIndex(content, -1)
	for _, match := range matches {
		if ref := getCrossReference(content, match[2], match[3], false); ref != nil {
			ret = append(ret, ref)
		}
	}

	var giteahost string
	if uapp, err := url.Parse(setting.AppURL); err == nil {
		giteahost = strings.ToLower(uapp.Host)
	}

	for _, link := range links {
		if u, err := url.Parse(link); err == nil {
			// Note: we're not attempting to match the URL scheme (http/https)
			host := strings.ToLower(u.Host)
			if host != "" && host != giteahost {
				continue
			}
			parts := strings.Split(u.EscapedPath(), "/")
			// /user/repo/issues/3
			if len(parts) != 5 || parts[0] != "" {
				continue
			}
			if parts[3] != "issues" && parts[3] != "pulls" {
				continue
			}
			// Note: closing/reopening keywords not supported with URLs
			bytes := []byte(parts[1] + "/" + parts[2] + "#" + parts[4])
			if ref := getCrossReference(bytes, 0, len(bytes), true); ref != nil {
				ref.refLocation = nil
				ret = append(ret, ref)
			}
		}
	}

	return ret
}

func getCrossReference(content []byte, start, end int, fromLink bool) *rawReference {
	refid := string(content[start:end])
	parts := strings.Split(refid, "#")
	if len(parts) != 2 {
		return nil
	}
	repo, issue := parts[0], parts[1]
	index, err := strconv.ParseInt(issue, 10, 64)
	if err != nil {
		return nil
	}
	if repo == "" {
		if fromLink {
			// Markdown links must specify owner/repo
			return nil
		}
		action, location := findActionKeywords(content, start)
		return &rawReference{
			index:          index,
			action:         action,
			issue:          issue,
			refLocation:    &ReferenceLocation{Start: start, End: end},
			actionLocation: location,
		}
	}
	parts = strings.Split(strings.ToLower(repo), "/")
	if len(parts) != 2 {
		return nil
	}
	owner, name := parts[0], parts[1]
	if !validNamePattern.MatchString(owner) || !validNamePattern.MatchString(name) {
		return nil
	}
	action, location := findActionKeywords(content, start)
	return &rawReference{
		index:          index,
		owner:          owner,
		name:           name,
		action:         action,
		issue:          issue,
		refLocation:    &ReferenceLocation{Start: start, End: end},
		actionLocation: location,
	}
}

func findActionKeywords(content []byte, start int) (XRefAction, *ReferenceLocation) {
	m := issueCloseKeywordsPat.FindSubmatchIndex(content[:start])
	if m != nil {
		return XRefActionCloses, &ReferenceLocation{Start: m[2], End: m[3]}
	}
	m = issueReopenKeywordsPat.FindSubmatchIndex(content[:start])
	if m != nil {
		return XRefActionReopens, &ReferenceLocation{Start: m[2], End: m[3]}
	}
	return XRefActionNone, nil
}
