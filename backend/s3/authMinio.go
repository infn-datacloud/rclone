package s3

// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"crypto"
	"crypto/hmac"
	"sync"

	jwtgo "github.com/golang-jwt/jwt"

	jsoniter "github.com/json-iterator/go"
)

const (
	// Minimum length for MinIO access key.
	accessKeyMinLen = 3

	// Maximum length for MinIO access key.
	// There is no max length enforcement for access keys
	accessKeyMaxLen = 20

	// Minimum length for MinIO secret key for both server and gateway mode.
	secretKeyMinLen = 8

	// Maximum secret key length for MinIO, this
	// is used when autogenerating new credentials.
	// There is no max length enforcement for secret keys
	secretKeyMaxLen = 40

	// Alpha numeric table used for generating access keys.
	alphaNumericTable = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Total length of the alpha numeric table.
	alphaNumericTableLen = byte(len(alphaNumericTable))
)

// Common errors generated for access and secret key validation.
var (
	ErrInvalidAccessKeyLength = fmt.Errorf("access key length should be between %d and %d", accessKeyMinLen, accessKeyMaxLen)
	ErrInvalidSecretKeyLength = fmt.Errorf("secret key length should be between %d and %d", secretKeyMinLen, secretKeyMaxLen)
)

// IsAccessKeyValid - validate access key for right length.
func IsAccessKeyValid(accessKey string) bool {
	return len(accessKey) >= accessKeyMinLen
}

// IsSecretKeyValid - validate secret key for right length.
func IsSecretKeyValid(secretKey string) bool {
	return len(secretKey) >= secretKeyMinLen
}

// Default access and secret keys.
const (
	DefaultAccessKey = "minioadmin"
	DefaultSecretKey = "minioadmin"
)

// Default access credentials
var (
	DefaultCredentials = Credentials{
		AccessKey: DefaultAccessKey,
		SecretKey: DefaultSecretKey,
	}
)

const (
	// AccountOn indicates that credentials are enabled
	AccountOn = "on"
	// AccountOff indicates that credentials are disabled
	AccountOff = "off"
)

// Credentials holds access and secret keys.
type Credentials struct {
	AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
	SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
	Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
	SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
	Status       string    `xml:"-" json:"status,omitempty"`
	ParentUser   string    `xml:"-" json:"parentUser,omitempty"`
	Groups       []string  `xml:"-" json:"groups,omitempty"`
}

func (cred Credentials) String() string {
	var s strings.Builder
	s.WriteString(cred.AccessKey)
	s.WriteString(":")
	s.WriteString(cred.SecretKey)
	if cred.SessionToken != "" {
		s.WriteString("\n")
		s.WriteString(cred.SessionToken)
	}
	if !cred.Expiration.IsZero() && !cred.Expiration.Equal(timeSentinel) {
		s.WriteString("\n")
		s.WriteString(cred.Expiration.String())
	}
	return s.String()
}

// IsExpired - returns whether Credential is expired or not.
func (cred Credentials) IsExpired() bool {
	if cred.Expiration.IsZero() || cred.Expiration.Equal(timeSentinel) {
		return false
	}

	return cred.Expiration.Before(time.Now().UTC())
}

// IsTemp - returns whether credential is temporary or not.
func (cred Credentials) IsTemp() bool {
	return cred.SessionToken != "" && !cred.Expiration.IsZero() && !cred.Expiration.Equal(timeSentinel)
}

// IsServiceAccount - returns whether credential is a service account or not
func (cred Credentials) IsServiceAccount() bool {
	return cred.ParentUser != "" && (cred.Expiration.IsZero() || cred.Expiration.Equal(timeSentinel))
}

// IsValid - returns whether credential is valid or not.
func (cred Credentials) IsValid() bool {
	// Verify credentials if its enabled or not set.
	if cred.Status == AccountOff {
		return false
	}
	return IsAccessKeyValid(cred.AccessKey) && IsSecretKeyValid(cred.SecretKey) && !cred.IsExpired()
}

// Equal - returns whether two credentials are equal or not.
func (cred Credentials) Equal(ccred Credentials) bool {
	if !ccred.IsValid() {
		return false
	}
	return (cred.AccessKey == ccred.AccessKey && subtle.ConstantTimeCompare([]byte(cred.SecretKey), []byte(ccred.SecretKey)) == 1 &&
		subtle.ConstantTimeCompare([]byte(cred.SessionToken), []byte(ccred.SessionToken)) == 1)
}

var timeSentinel = time.Unix(0, 0).UTC()

// ErrInvalidDuration invalid token expiry
var ErrInvalidDuration = errors.New("invalid token expiry")

// ExpToInt64 - convert input interface value to int64.
func ExpToInt64(expI interface{}) (expAt int64, err error) {
	switch exp := expI.(type) {
	case string:
		expAt, err = strconv.ParseInt(exp, 10, 64)
	case float64:
		expAt, err = int64(exp), nil
	case int64:
		expAt, err = exp, nil
	case int:
		expAt, err = int64(exp), nil
	case uint64:
		expAt, err = int64(exp), nil
	case uint:
		expAt, err = int64(exp), nil
	case json.Number:
		expAt, err = exp.Int64()
	case time.Duration:
		expAt, err = time.Now().UTC().Add(exp).Unix(), nil
	case nil:
		expAt, err = 0, nil
	default:
		expAt, err = 0, ErrInvalidDuration
	}
	if expAt < 0 {
		return 0, ErrInvalidDuration
	}
	return expAt, err
}

// GetNewCredentialsWithMetadata generates and returns new credential with expiry.
func GetNewCredentialsWithMetadata(m map[string]interface{}, tokenSecret string) (cred Credentials, err error) {
	readBytes := func(size int) (data []byte, err error) {
		data = make([]byte, size)
		var n int
		if n, err = rand.Read(data); err != nil {
			return nil, err
		} else if n != size {
			return nil, fmt.Errorf("Not enough data. Expected to read: %v bytes, got: %v bytes", size, n)
		}
		return data, nil
	}

	// Generate access key.
	keyBytes, err := readBytes(accessKeyMaxLen)
	if err != nil {
		return cred, err
	}
	for i := 0; i < accessKeyMaxLen; i++ {
		keyBytes[i] = alphaNumericTable[keyBytes[i]%alphaNumericTableLen]
	}
	accessKey := string(keyBytes)

	// Generate secret key.
	keyBytes, err = readBytes(secretKeyMaxLen)
	if err != nil {
		return cred, err
	}

	secretKey := strings.Replace(string([]byte(base64.StdEncoding.EncodeToString(keyBytes))[:secretKeyMaxLen]),
		"/", "+", -1)

	return CreateNewCredentialsWithMetadata(accessKey, secretKey, m, tokenSecret)
}

// CreateNewCredentialsWithMetadata - creates new credentials using the specified access & secret keys
// and generate a session token if a secret token is provided.
func CreateNewCredentialsWithMetadata(accessKey, secretKey string, m map[string]interface{}, tokenSecret string) (cred Credentials, err error) {
	if len(accessKey) < accessKeyMinLen || len(accessKey) > accessKeyMaxLen {
		return Credentials{}, ErrInvalidAccessKeyLength
	}

	if len(secretKey) < secretKeyMinLen || len(secretKey) > secretKeyMaxLen {
		return Credentials{}, ErrInvalidSecretKeyLength
	}

	cred.AccessKey = accessKey
	cred.SecretKey = secretKey
	cred.Status = AccountOn

	if tokenSecret == "" {
		cred.Expiration = timeSentinel
		return cred, nil
	}

	expiry, err := ExpToInt64(m["exp"])
	if err != nil {
		return cred, err
	}
	cred.Expiration = time.Unix(expiry, 0).UTC()

	cred.SessionToken, err = JWTSignWithAccessKey(cred.AccessKey, m, tokenSecret)
	if err != nil {
		return cred, err
	}

	return cred, nil
}

// JWTSignWithAccessKey - generates a session token.
func JWTSignWithAccessKey(accessKey string, m map[string]interface{}, tokenSecret string) (string, error) {
	m["accessKey"] = accessKey
	jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.MapClaims(m))
	return jwt.SignedString([]byte(tokenSecret))
}

// ExtractClaims extracts JWT claims from a security token using a secret key
func ExtractClaims(token, secretKey string) (*MapClaims, error) {
	if token == "" || secretKey == "" {
		return nil, errors.New("invalid argument")
	}

	claims := NewMapClaims()
	stsTokenCallback := func(claims *MapClaims) ([]byte, error) {
		return []byte(secretKey), nil
	}

	if err := ParseWithClaims(token, claims, stsTokenCallback); err != nil {
		return nil, err
	}

	return claims, nil
}

// GetNewCredentials generates and returns new credential.
func GetNewCredentials() (cred Credentials, err error) {
	return GetNewCredentialsWithMetadata(map[string]interface{}{}, "")
}

// CreateCredentials returns new credential with the given access key and secret key.
// Error is returned if given access key or secret key are invalid length.
func CreateCredentials(accessKey, secretKey string) (cred Credentials, err error) {
	if !IsAccessKeyValid(accessKey) {
		return cred, ErrInvalidAccessKeyLength
	}
	if !IsSecretKeyValid(secretKey) {
		return cred, ErrInvalidSecretKeyLength
	}
	cred.AccessKey = accessKey
	cred.SecretKey = secretKey
	cred.Expiration = timeSentinel
	cred.Status = AccountOn
	return cred, nil
}

// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// This file is a re-implementation of the original code here with some
// additional allocation tweaks reproduced using GODEBUG=allocfreetrace=1
// original file https://github.com/golang-jwt/jwt/blob/main/parser.go
// borrowed under MIT License https://github.com/golang-jwt/jwt/blob/main/LICENSE

// SigningMethodHMAC - Implements the HMAC-SHA family of signing methods signing methods
// Expects key type of []byte for both signing and validation
type SigningMethodHMAC struct {
	Name string
	Hash crypto.Hash
}

// Specific instances for HS256, HS384, HS512
var (
	SigningMethodHS256 *SigningMethodHMAC
	SigningMethodHS384 *SigningMethodHMAC
	SigningMethodHS512 *SigningMethodHMAC
)

var (
	base64BufPool sync.Pool
	hmacSigners   []*SigningMethodHMAC
)

func init() {
	base64BufPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 8192)
			return &buf
		},
	}

	hmacSigners = []*SigningMethodHMAC{
		{"HS256", crypto.SHA256},
		{"HS384", crypto.SHA384},
		{"HS512", crypto.SHA512},
	}
}

// StandardClaims are basically standard claims with "accessKey"
type StandardClaims struct {
	AccessKey string `json:"accessKey,omitempty"`
	jwtgo.StandardClaims
}

// MapClaims - implements custom unmarshaller
type MapClaims struct {
	AccessKey string `json:"accessKey,omitempty"`
	jwtgo.MapClaims
}

// GetAccessKey will return the access key.
// If nil an empty string will be returned.
func (c *MapClaims) GetAccessKey() string {
	if c == nil {
		return ""
	}
	return c.AccessKey
}

// NewStandardClaims - initializes standard claims
func NewStandardClaims() *StandardClaims {
	return &StandardClaims{}
}

// SetIssuer sets issuer for these claims
func (c *StandardClaims) SetIssuer(issuer string) {
	c.Issuer = issuer
}

// SetAudience sets audience for these claims
func (c *StandardClaims) SetAudience(aud string) {
	c.Audience = aud
}

// SetExpiry sets expiry in unix epoch secs
func (c *StandardClaims) SetExpiry(t time.Time) {
	c.ExpiresAt = t.Unix()
}

// SetAccessKey sets access key as jwt subject and custom
// "accessKey" field.
func (c *StandardClaims) SetAccessKey(accessKey string) {
	c.Subject = accessKey
	c.AccessKey = accessKey
}

// Valid - implements https://godoc.org/github.com/golang-jwt/jwt#Claims compatible
// claims interface, additionally validates "accessKey" fields.
func (c *StandardClaims) Valid() error {
	if err := c.StandardClaims.Valid(); err != nil {
		return err
	}

	if c.AccessKey == "" && c.Subject == "" {
		return jwtgo.NewValidationError("accessKey/sub missing",
			jwtgo.ValidationErrorClaimsInvalid)
	}

	return nil
}

// NewMapClaims - Initializes a new map claims
func NewMapClaims() *MapClaims {
	return &MapClaims{MapClaims: jwtgo.MapClaims{}}
}

// Lookup returns the value and if the key is found.
func (c *MapClaims) Lookup(key string) (value string, ok bool) {
	if c == nil {
		return "", false
	}
	var vinterface interface{}
	vinterface, ok = c.MapClaims[key]
	if ok {
		value, ok = vinterface.(string)
	}
	return
}

// SetExpiry sets expiry in unix epoch secs
func (c *MapClaims) SetExpiry(t time.Time) {
	c.MapClaims["exp"] = t.Unix()
}

// SetAccessKey sets access key as jwt subject and custom
// "accessKey" field.
func (c *MapClaims) SetAccessKey(accessKey string) {
	c.MapClaims["sub"] = accessKey
	c.MapClaims["accessKey"] = accessKey
}

// Valid - implements https://godoc.org/github.com/golang-jwt/jwt#Claims compatible
// claims interface, additionally validates "accessKey" fields.
func (c *MapClaims) Valid() error {
	if err := c.MapClaims.Valid(); err != nil {
		return err
	}

	if c.AccessKey == "" {
		return jwtgo.NewValidationError("accessKey/sub missing",
			jwtgo.ValidationErrorClaimsInvalid)
	}

	return nil
}

// Map returns underlying low-level map claims.
func (c *MapClaims) Map() map[string]interface{} {
	if c == nil {
		return nil
	}
	return c.MapClaims
}

// MarshalJSON marshals the MapClaims struct
func (c *MapClaims) MarshalJSON() ([]byte, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(c.MapClaims)
}

// ParseWithStandardClaims - parse the token string, valid methods.
func ParseWithStandardClaims(tokenStr string, claims *StandardClaims, key []byte) error {
	// Key is not provided.
	if key == nil {
		// keyFunc was not provided, return error.
		return jwtgo.NewValidationError("no key was provided.", jwtgo.ValidationErrorUnverifiable)
	}

	bufp := base64BufPool.Get().(*[]byte)
	defer base64BufPool.Put(bufp)

	signer, err := ParseUnverifiedStandardClaims(tokenStr, claims, *bufp)
	if err != nil {
		return err
	}

	i := strings.LastIndex(tokenStr, ".")
	if i < 0 {
		return jwtgo.ErrSignatureInvalid
	}

	n, err := base64Decode(tokenStr[i+1:], *bufp)
	if err != nil {
		return err
	}

	hasher := hmac.New(signer.Hash.New, key)
	hasher.Write([]byte(tokenStr[:i]))
	if !hmac.Equal((*bufp)[:n], hasher.Sum(nil)) {
		return jwtgo.ErrSignatureInvalid
	}

	if claims.AccessKey == "" && claims.Subject == "" {
		return jwtgo.NewValidationError("accessKey/sub missing",
			jwtgo.ValidationErrorClaimsInvalid)
	}

	// Signature is valid, lets validate the claims for
	// other fields such as expiry etc.
	return claims.Valid()
}

// https://tools.ietf.org/html/rfc7519#page-11
type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// ParseUnverifiedStandardClaims - WARNING: Don't use this method unless you know what you're doing
//
// This method parses the token but doesn't validate the signature. It's only
// ever useful in cases where you know the signature is valid (because it has
// been checked previously in the stack) and you want to extract values from
// it.
func ParseUnverifiedStandardClaims(tokenString string, claims *StandardClaims, buf []byte) (*SigningMethodHMAC, error) {
	if strings.Count(tokenString, ".") != 2 {
		return nil, jwtgo.ErrSignatureInvalid
	}

	i := strings.Index(tokenString, ".")
	j := strings.LastIndex(tokenString, ".")

	n, err := base64Decode(tokenString[:i], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	var header = jwtHeader{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(buf[:n], &header); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	n, err = base64Decode(tokenString[i+1:j], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	if err = json.Unmarshal(buf[:n], claims); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	for _, signer := range hmacSigners {
		if header.Algorithm == signer.Name {
			return signer, nil
		}
	}

	return nil, jwtgo.NewValidationError(fmt.Sprintf("signing method (%s) is unavailable.", header.Algorithm),
		jwtgo.ValidationErrorUnverifiable)
}

// ParseWithClaims - parse the token string, valid methods.
func ParseWithClaims(tokenStr string, claims *MapClaims, fn func(*MapClaims) ([]byte, error)) error {
	// Key lookup function has to be provided.
	if fn == nil {
		// keyFunc was not provided, return error.
		return jwtgo.NewValidationError("no Keyfunc was provided.", jwtgo.ValidationErrorUnverifiable)
	}

	bufp := base64BufPool.Get().(*[]byte)
	defer base64BufPool.Put(bufp)

	signer, err := ParseUnverifiedMapClaims(tokenStr, claims, *bufp)
	if err != nil {
		return err
	}

	i := strings.LastIndex(tokenStr, ".")
	if i < 0 {
		return jwtgo.ErrSignatureInvalid
	}

	n, err := base64Decode(tokenStr[i+1:], *bufp)
	if err != nil {
		return err
	}

	var ok bool
	claims.AccessKey, ok = claims.Lookup("accessKey")
	if !ok {
		claims.AccessKey, ok = claims.Lookup("sub")
		if !ok {
			return jwtgo.NewValidationError("accessKey/sub missing",
				jwtgo.ValidationErrorClaimsInvalid)
		}
	}

	// Lookup key from claims, claims may not be valid and may return
	// invalid key which is okay as the signature verification will fail.
	key, err := fn(claims)
	if err != nil {
		return err
	}

	hasher := hmac.New(signer.Hash.New, key)
	hasher.Write([]byte(tokenStr[:i]))
	if !hmac.Equal((*bufp)[:n], hasher.Sum(nil)) {
		return jwtgo.ErrSignatureInvalid
	}

	// Signature is valid, lets validate the claims for
	// other fields such as expiry etc.
	return claims.Valid()
}

// base64Decode returns the bytes represented by the base64 string s.
func base64Decode(s string, buf []byte) (int, error) {
	return base64.RawURLEncoding.Decode(buf, []byte(s))
}

// ParseUnverifiedMapClaims - WARNING: Don't use this method unless you know what you're doing
//
// This method parses the token but doesn't validate the signature. It's only
// ever useful in cases where you know the signature is valid (because it has
// been checked previously in the stack) and you want to extract values from
// it.
func ParseUnverifiedMapClaims(tokenString string, claims *MapClaims, buf []byte) (*SigningMethodHMAC, error) {
	if strings.Count(tokenString, ".") != 2 {
		return nil, jwtgo.ErrSignatureInvalid
	}

	i := strings.Index(tokenString, ".")
	j := strings.LastIndex(tokenString, ".")

	n, err := base64Decode(tokenString[:i], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	var header = jwtHeader{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(buf[:n], &header); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	n, err = base64Decode(tokenString[i+1:j], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	if err = json.Unmarshal(buf[:n], &claims.MapClaims); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	for _, signer := range hmacSigners {
		if header.Algorithm == signer.Name {
			return signer, nil
		}
	}

	return nil, jwtgo.NewValidationError(fmt.Sprintf("signing method (%s) is unavailable.", header.Algorithm),
		jwtgo.ValidationErrorUnverifiable)
}
