package lfs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"encoding/base64"
	"github.com/dgrijalva/jwt-go"
	"github.com/gogits/gogs/models"
	"github.com/gogits/gogs/modules/context"
	"github.com/gogits/gogs/modules/log"
	"github.com/gogits/gogs/modules/setting"
	"gopkg.in/macaron.v1"
)

const (
	contentMediaType = "application/vnd.git-lfs"
	metaMediaType    = contentMediaType + "+json"
)

// RequestVars contain variables from the HTTP request. Variables from routing, json body decoding, and
// some headers are stored.
type RequestVars struct {
	Oid           string
	Size          int64
	User          string
	Password      string
	Repo          string
	Authorization string
}

type BatchVars struct {
	Transfers []string       `json:"transfers,omitempty"`
	Operation string         `json:"operation"`
	Objects   []*RequestVars `json:"objects"`
}

type BatchResponse struct {
	Transfer string            `json:"transfer,omitempty"`
	Objects  []*Representation `json:"objects"`
}

// Representation is object medata as seen by clients of the lfs server.
type Representation struct {
	Oid     string           `json:"oid"`
	Size    int64            `json:"size"`
	Actions map[string]*link `json:"actions"`
	Error   *ObjectError     `json:"error,omitempty"`
}

type ObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ObjectLink builds a URL linking to the object.
func (v *RequestVars) ObjectLink() string {
	path := ""

	if len(v.User) > 0 {
		path += fmt.Sprintf("/%s", v.User)
	}

	if len(v.Repo) > 0 {
		path += fmt.Sprintf("/%s", v.Repo)
	}

	path += fmt.Sprintf("/objects/%s", v.Oid)

	return fmt.Sprintf("%slfs%s", setting.AppUrl, path)
}

// link provides a structure used to build a hypermedia representation of an HTTP link.
type link struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresAt time.Time         `json:"expires_at,omitempty"`
}

type LFSHandler struct {
	contentStore *ContentStore
}

func NewLFSHandler() *LFSHandler {
	contentStore, err := NewContentStore(setting.LFS.ContentPath)

	if err != nil {
		log.Fatal(4, "Error initializing LFS content store: %s", err)
	}

	app := &LFSHandler{contentStore: contentStore}
	return app
}

func (a *LFSHandler) ObjectOidHandler(ctx *context.Context) {

	if ctx.Req.Method == "GET" || ctx.Req.Method == "HEAD" {
		if MetaMatcher(ctx.Req) {
			a.GetMetaHandler(ctx)
			return
		}
		if ContentMatcher(ctx.Req) {
			a.GetContentHandler(ctx)
			return
		}
	} else if ctx.Req.Method == "PUT" && ContentMatcher(ctx.Req) {
		a.PutHandler(ctx)
		return
	}

}

// GetContentHandler gets the content from the content store
func (a *LFSHandler) GetContentHandler(ctx *context.Context) {

	rv := unpack(ctx)
	if !authenticate(rv.Authorization, false) {
		requireAuth(ctx)
		return
	}

	meta, err := models.GetLFSMetaObjectByOid(rv.Oid)
	if err != nil {
		writeStatus(ctx, 404)
		return
	}

	// Support resume download using Range header
	var fromByte int64
	statusCode := 200
	if rangeHdr := ctx.Req.Header.Get("Range"); rangeHdr != "" {
		regex := regexp.MustCompile(`bytes=(\d+)\-.*`)
		match := regex.FindStringSubmatch(rangeHdr)
		if match != nil && len(match) > 1 {
			statusCode = 206
			fromByte, _ = strconv.ParseInt(match[1], 10, 32)
			ctx.Resp.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", fromByte, meta.Size-1, int64(meta.Size)-fromByte))
		}
	}

	content, err := a.contentStore.Get(meta, fromByte)
	if err != nil {
		writeStatus(ctx, 404)
		return
	}

	ctx.Resp.WriteHeader(statusCode)
	io.Copy(ctx.Resp, content)
	logRequest(ctx.Req, statusCode)
}

// GetMetaHandler retrieves metadata about the object
func (a *LFSHandler) GetMetaHandler(ctx *context.Context) {

	rv := unpack(ctx)
	if !authenticate(rv.Authorization, false) {
		requireAuth(ctx)
		return
	}

	meta, err := models.GetLFSMetaObjectByOid(rv.Oid)
	if err != nil {
		writeStatus(ctx, 404)
		return
	}

	ctx.Resp.Header().Set("Content-Type", metaMediaType)

	if ctx.Req.Method == "GET" {
		enc := json.NewEncoder(ctx.Resp)
		enc.Encode(a.Represent(rv, meta, true, false))
	}

	logRequest(ctx.Req, 200)
}

// PostHandler instructs the client how to upload data
func (a *LFSHandler) PostHandler(ctx *context.Context) {

	rv := unpack(ctx)

	if !authenticate(rv.Authorization, true) {
		requireAuth(ctx)
	}

	meta, err := models.NewLFSMetaObject(&models.LFSMetaObject{Oid: rv.Oid, Size: rv.Size})

	if err != nil {
		writeStatus(ctx, 404)
		return
	}

	ctx.Resp.Header().Set("Content-Type", metaMediaType)

	sentStatus := 202
	if meta.Existing && a.contentStore.Exists(meta) {
		sentStatus = 200
	}
	ctx.Resp.WriteHeader(sentStatus)

	enc := json.NewEncoder(ctx.Resp)
	enc.Encode(a.Represent(rv, meta, meta.Existing, true))
	logRequest(ctx.Req, sentStatus)
}

// BatchHandler provides the batch api
func (a *LFSHandler) BatchHandler(ctx *context.Context) {
	bv := unpackbatch(ctx)

	var responseObjects []*Representation

	// Create a response object
	for _, object := range bv.Objects {

		if !authenticate(object.Authorization, true) {
			requireAuth(ctx)
			return
		}

		meta, err := models.GetLFSMetaObjectByOid(object.Oid)

		if err == nil && a.contentStore.Exists(meta) { // Object is found and exists
			responseObjects = append(responseObjects, a.Represent(object, meta, true, false))
			continue
		}

		// Object is not found
		meta, err = models.NewLFSMetaObject(&models.LFSMetaObject{Oid: object.Oid, Size: object.Size})

		if err == nil {
			responseObjects = append(responseObjects, a.Represent(object, meta, meta.Existing, true))
		}
	}

	ctx.Resp.Header().Set("Content-Type", metaMediaType)

	respobj := &BatchResponse{Objects: responseObjects}

	enc := json.NewEncoder(ctx.Resp)
	enc.Encode(respobj)
	logRequest(ctx.Req, 200)
}

// PutHandler receives data from the client and puts it into the content store
func (a *LFSHandler) PutHandler(ctx *context.Context) {
	rv := unpack(ctx)

	if !authenticate(rv.Authorization, true) {
		requireAuth(ctx)
		return
	}

	meta, err := models.GetLFSMetaObjectByOid(rv.Oid)

	if err != nil {
		writeStatus(ctx, 404)
		return
	}

	if err := a.contentStore.Put(meta, ctx.Req.Body().ReadCloser()); err != nil {
		models.RemoveLFSMetaObjectByOid(rv.Oid)
		ctx.Resp.WriteHeader(500)
		fmt.Fprintf(ctx.Resp, `{"message":"%s"}`, err)
		return
	}

	logRequest(ctx.Req, 200)
}

// Represent takes a RequestVars and Meta and turns it into a Representation suitable
// for json encoding
func (a *LFSHandler) Represent(rv *RequestVars, meta *models.LFSMetaObject, download, upload bool) *Representation {
	rep := &Representation{
		Oid:     meta.Oid,
		Size:    meta.Size,
		Actions: make(map[string]*link),
	}

	header := make(map[string]string)
	header["Accept"] = contentMediaType
	header["Authorization"] = rv.Authorization

	if download {
		rep.Actions["download"] = &link{Href: rv.ObjectLink(), Header: header}
	}

	if upload {
		rep.Actions["upload"] = &link{Href: rv.ObjectLink(), Header: header}
	}

	return rep
}

// ContentMatcher provides a mux.MatcherFunc that only allows requests that contain
// an Accept header with the contentMediaType
func ContentMatcher(r macaron.Request) bool {
	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	return mt == contentMediaType
}

// MetaMatcher provides a mux.MatcherFunc that only allows requests that contain
// an Accept header with the metaMediaType
func MetaMatcher(r macaron.Request) bool {
	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	return mt == metaMediaType
}

func unpack(ctx *context.Context) *RequestVars {
	r := ctx.Req
	rv := &RequestVars{
		User:          ctx.Params("user"),
		Repo:          ctx.Params("repo"),
		Oid:           ctx.Params("oid"),
		Authorization: r.Header.Get("Authorization"),
	}

	if r.Method == "POST" { // Maybe also check if +json
		var p RequestVars
		dec := json.NewDecoder(r.Body().ReadCloser())
		err := dec.Decode(&p)
		if err != nil {
			return rv
		}

		rv.Oid = p.Oid
		rv.Size = p.Size
	}

	return rv
}

// TODO cheap hack, unify with unpack
func unpackbatch(ctx *context.Context) *BatchVars {

	r := ctx.Req
	var bv BatchVars

	dec := json.NewDecoder(r.Body().ReadCloser())
	err := dec.Decode(&bv)
	if err != nil {
		return &bv
	}

	for i := 0; i < len(bv.Objects); i++ {
		bv.Objects[i].User = ctx.Params("user")
		bv.Objects[i].Repo = ctx.Params("repo")
		bv.Objects[i].Authorization = r.Header.Get("Authorization")
	}

	return &bv
}

func writeStatus(ctx *context.Context, status int) {
	message := http.StatusText(status)

	mediaParts := strings.Split(ctx.Req.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	if strings.HasSuffix(mt, "+json") {
		message = `{"message":"` + message + `"}`
	}

	ctx.Resp.WriteHeader(status)
	fmt.Fprint(ctx.Resp, message)
	logRequest(ctx.Req, status)
}

func logRequest(r macaron.Request, status int) {
	log.Debug("LFS request - Method: %s, URL: %s, Status %s", r.Method, r.URL, status)
}

// authenticate uses the authorization string to determine whether
// or not to proceed. This server assumes an HTTP Basic auth format.
func authenticate(authorization string, requireWrite bool) bool {

	if authorization == "" {
		return false
	}

	if authenticateToken(authorization, requireWrite) {
		return true
	}

	if !strings.HasPrefix(authorization, "Basic ") {
		return false
	}

	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authorization, "Basic "))
	if err != nil {
		return false
	}
	cs := string(c)
	i := strings.IndexByte(cs, ':')
	if i < 0 {
		return false
	}
	user, password := cs[:i], cs[i+1:]
	_ = user
	_ = password
	// TODO check Basic Authentication

	return false
}

func authenticateToken(authorization string, requireWrite bool) bool {
	if !strings.HasPrefix(authorization, "Bearer ") {
		return false
	}

	token, err := jwt.Parse(authorization[7:], func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return setting.LFS.JWTSecretBytes, nil
	})
	if err != nil {
		return false
	}
	claims, claimsOk := token.Claims.(jwt.MapClaims)
	if !token.Valid || !claimsOk {
		return false
	}

	opstr, ok := claims["op"].(string)
	if !ok {
		return false
	}
	op := strings.ToLower(strings.TrimSpace(opstr))
	status := op == "upload" || (op == "download" && !requireWrite)
	return status
}

type authError struct {
	error
}

func (e authError) AuthError() bool {
	return true
}

func requireAuth(ctx *context.Context) {
	ctx.Resp.Header().Set("WWW-Authenticate", "Basic realm=gogs-lfs")
	writeStatus(ctx, 401)
}
