package private

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/modules/httplib"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

func newRequest(url, method string) *httplib.Request {
	return httplib.NewRequest(url, method).Header("Authorization",
		fmt.Sprintf("Bearer %s", setting.InternalToken))
}

type Response struct {
	Err string `json:"err"`
}

func decodeJSONError(resp *http.Response) *Response {
	var res Response
	err := json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		res.Err = err.Error()
	}
	return &res
}

// UpdatePublicKeyUpdated update publick key updates
func UpdatePublicKeyUpdated(keyID int64) error {
	// Ask for running deliver hook and test pull request tasks.
	reqURL := setting.LocalURL + fmt.Sprintf("internal/ssh/%d/update", keyID)
	log.GitLogger.Trace("UpdatePublicKeyUpdated: %s", reqURL)

	resp, err := newRequest(reqURL, "POST").SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	}).Response()
	if err != nil {
		return err
	}

	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Failed to update public key: %s", decodeJSONError(resp).Err)
	}
	return nil
}
