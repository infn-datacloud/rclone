package s3

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/indigo-dc/liboidcagent-go"
	"github.com/rclone/rclone/fs"
)

// IAMProvider credential provider for oidc
type IAMProvider struct {
	stsEndpoint  string
	accountname  string
	useOidcAgent bool
	RoleName     string
	Audience     string
	httpClient   *http.Client
	creds        *AssumeRoleWithWebIdentityResponse
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
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`
	Audience        string          `xml:",omitempty"`
	// Ref: https://github.com/minio/minio/blob/master/internal/auth/credentials.go#L96
	Credentials                 Credentials `xml:",omitempty"`
	PackedPolicySize            int         `xml:",omitempty"`
	Provider                    string      `xml:",omitempty"`
	SubjectFromWebIdentityToken string      `xml:",omitempty"`
}

type MyXMLStruct struct {
	XMLName xml.Name `xml:"AssumeRoleWithWebIdentityResponse"`
	Attr    string   `xml:"xmlns,attr"`
	Result  struct {
		SubjectFromWebIdentityToken string `xml:"SubjectFromWebIdentityToken"`
		Audience                    string `xml:"Audience"`
		AssumedRoleUser             struct {
			Arn          string `xml:"Arn"`
			AssumeRoleID string `xml:"AssumeRoleId"`
		} `xml:"AssumedRoleUser"`
		Credentials struct {
			AccessKey    string `xml:"AccessKeyId"`
			Expiration   string `xml:"Expiration"`
			SecretAccess string `xml:"SecretAccessKey"`
			SessionToken string `xml:"SessionToken"`
		} `xml:"Credentials"`
		Provider         string `xml:"Provider"`
		PackedPolicySize int    `xml:"PackedPolicySize"`
	} `xml:"AssumeRoleWithWebIdentityResult"`
}

// Retrieve credentials
func (t *IAMProvider) Retrieve() (credentials.Value, error) {
	var err error
	var token string

	if t.useOidcAgent {
		token, err = liboidcagent.GetAccessToken(liboidcagent.TokenRequest{
			ShortName:      t.accountname,
			MinValidPeriod: 900,
			Audiences:      []string{t.Audience},
		})
		if err != nil {
			return credentials.Value{}, err
		}
	} else {
		dat, err := ioutil.ReadFile(".token")
		if err != nil {
			fs.Errorf(err, "IAM - token read error")
			return credentials.Value{}, err
		}
		token = string(dat)
	}

	fs.Debugf(token, "IAM - token")

	//contentType := ""
	body := url.Values{}
	body.Set("RoleArn", "arn:aws:iam:::role/"+t.RoleName)
	body.Set("RoleSessionName", t.RoleName)
	body.Set("Action", "AssumeRoleWithWebIdentity")
	body.Set("Version", "2011-06-15")
	body.Set("WebIdentityToken", token)
	body.Set("DurationSeconds", "900")

	// TODO: retrieve token with https POST with t.httpClient
	//r, err := t.httpClient.Post(t.stsEndpoint, contentType, strings.NewReader(body.Encode()))
	url, err := url.Parse(t.stsEndpoint + "?" + body.Encode())
	if err != nil {
		fs.Errorf(err, "IAM - encode URL")
		return credentials.Value{}, err
	}

	fs.Debugf(url, "IAM - url")
	req := http.Request{
		Method: "POST",
		URL:    url,
	}

	// Set a timeout of 30 seconds (adjust as needed)
	t.httpClient.Timeout = time.Duration(60) * time.Second

	// TODO: retrieve token with https POST with t.httpClient
	r, err := t.httpClient.Do(&req)
	if err != nil {
		fs.Errorf(err, "IAM - http request")
		return credentials.Value{}, err
	}

	var rbody bytes.Buffer

	bodyBytes, errRead := ioutil.ReadAll(r.Body)

	if errRead != nil {
		fs.Errorf(errRead, "IAM read body")
		return credentials.Value{}, errRead
	}

	ns := "https://sts.amazonaws.com/doc/2011-06-15/"

	data := string(bodyBytes)

	xmlStruct := MyXMLStruct{
		Attr: ns,
	}

	errUnmarshall := xml.Unmarshal([]byte(data), &xmlStruct)
	if errUnmarshall != nil {
		fs.Errorf(errUnmarshall, "IAM xml unmarshal")
		return credentials.Value{}, errUnmarshall
	}

	xmlBytes, errMarshalIndent := xml.MarshalIndent(xmlStruct, "", "  ")
	if errMarshalIndent != nil {
		fs.Errorf(errMarshalIndent, "IAM xml marshal indent")
		return credentials.Value{}, errMarshalIndent
	}

	rbody.Write(xmlBytes)

	t.creds = &AssumeRoleWithWebIdentityResponse{}

	if err != nil {
		fs.Errorf(err, "IAM - read body")
		return credentials.Value{}, err
	}

	err = xml.Unmarshal(rbody.Bytes(), t.creds)
	if err != nil {
		fs.Errorf(err, "IAM - unmarshal credentials")
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
