package s3

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/indigo-dc/liboidcagent-go/liboidcagent"
	"github.com/minio/minio/pkg/auth"
)

// IAMProvider credential provider for oidc
type IAMProvider struct {
	stsEndpoint string
	accountname string
	httpClient  *http.Client
	creds       *AssumeRoleWithWebIdentityResponse
}

// AssumeRoleWithWebIdentityResponse the struct of the STS WebIdentity call response
type AssumeRoleWithWebIdentityResponse struct {
	XMLName          xml.Name          `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithWebIdentityResponse" json:"-"`
	Result           WebIdentityResult `xml:"AssumeRoleWithWebIdentityResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

// AssumedRoleUser - The identifiers for the temporary security credentials that
// the operation returns. Please also see https://docs.aws.amazon.com/goto/WebAPI/sts-2011-06-15/AssumedRoleUser
type AssumedRoleUser struct {
	Arn           string
	AssumedRoleID string `xml:"AssumeRoleId"`
	// contains filtered or unexported fields
}

// WebIdentityResult - Contains the response to a successful AssumeRoleWithWebIdentity
// request, including temporary credentials that can be used to make MinIO API requests.
type WebIdentityResult struct {
	AssumedRoleUser             AssumedRoleUser  `xml:",omitempty"`
	Audience                    string           `xml:",omitempty"`
	Credentials                 auth.Credentials `xml:",omitempty"`
	PackedPolicySize            int              `xml:",omitempty"`
	Provider                    string           `xml:",omitempty"`
	SubjectFromWebIdentityToken string           `xml:",omitempty"`
}

// Retrieve credentials
func (t *IAMProvider) Retrieve() (credentials.Value, error) {

	token, err := liboidcagent.GetAccessToken2(t.accountname, 60, "", "", "")
	if err != nil {
		return credentials.Value{}, err
	}

	fmt.Printf("Access token is: %s\n", token)

	//contentType := ""
	body := url.Values{}
	body.Set("Action", "AssumeRoleWithWebIdentity")
	body.Set("Version", "2011-06-15")
	body.Set("WebIdentityToken", token)
	body.Set("DurationSeconds", "900")

	// TODO: retrieve token with https POST with t.httpClient
	//r, err := t.httpClient.Post(t.stsEndpoint, contentType, strings.NewReader(body.Encode()))
	url, err := url.Parse(t.stsEndpoint + "?" + body.Encode())
	if err != nil {
		// fmt.Println(err)
		return credentials.Value{}, err
	}

	// fmt.Println(url)
	req := http.Request{
		Method: "POST",
		URL:    url,
	}

	// TODO: retrieve token with https POST with t.httpClient
	r, err := t.httpClient.Do(&req)
	if err != nil {
		// fmt.Println(err)
		return credentials.Value{}, err
	}

	t.creds = &AssumeRoleWithWebIdentityResponse{}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// fmt.Printf("error: %v", err)
		return credentials.Value{}, err
	}

	err = xml.Unmarshal(rbody, t.creds)
	if err != nil {
		// fmt.Printf("error: %v", err)
		return credentials.Value{}, err
	}

	return credentials.Value{
		AccessKeyID:     t.creds.Result.Credentials.AccessKey,
		SecretAccessKey: t.creds.Result.Credentials.SecretKey,
		SessionToken:    t.creds.Result.Credentials.SessionToken,
	}, nil

}

// IsExpired test
func (t *IAMProvider) IsExpired() bool {
	return t.creds.Result.Credentials.IsExpired()
}
