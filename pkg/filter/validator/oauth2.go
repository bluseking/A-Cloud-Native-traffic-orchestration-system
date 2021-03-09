package validator

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/megaease/easegateway/pkg/context"
)

type (
	// OAuth2TokenIntrospect defines the validator configuration for OAuth2 token introspection
	OAuth2TokenIntrospect struct {
		EndPoint     string `yaml:"endPoint" jsonschema:"required"`
		BasicAuth    string `yaml:"basicAuth" jsonschema:"omitempty"`
		ClientID     string `yaml:"clientId" jsonschema:"omitempty"`
		ClientSecret string `yaml:"clientSecret" jsonschema:"omitempty"`
		InsecureTLS  bool   `yaml:"insecureTls"`
	}

	// OAuth2JWT defines the validator configuration for OAuth2 self encoded access token
	OAuth2JWT struct {
		Algorithm string `yaml:"algorithm" jsonschema:"enum=HS256,enum=HS384,enum=HS512"`
		// Secret is in hex encoding
		Secret      string `yaml:"secret" jsonschema:"required,pattern=^[A-Fa-f0-9]+$"`
		secretBytes []byte
	}

	// OAuth2ValidatorSpec defines the configuration of OAuth2 validator
	OAuth2ValidatorSpec struct {
		TokenIntrospect *OAuth2TokenIntrospect `yaml:"tokenIntrospect" jsonschema:"omitempty"`
		JWT             *OAuth2JWT             `yaml:"jwt" jsonschema:"omitempty"`
	}

	// OAuth2Validator defines the OAuth2 validator
	OAuth2Validator struct {
		spec   *OAuth2ValidatorSpec
		client *http.Client
	}

	tokenInfo struct {
		Active    bool   `json:"active"`
		Scope     string `json:"scope"`
		ClientID  string `json:"client_id"`
		UserName  string `json:"username"`
		TokenType string `json:"token_type"`
		ExpiresAt int64  `json:"exp"`
		IssuedAt  int64  `json:"iat"`
		NotBefore int64  `json:"nbf"`
		Subject   string `json:"sub"`
		Audience  string `json:"aud"`
		Issuer    string `json:"iss"`
	}
)

// NewOAuth2Validator creates a new OAuth2 validator
func NewOAuth2Validator(spec *OAuth2ValidatorSpec) *OAuth2Validator {
	if spec.JWT != nil {
		spec.JWT.secretBytes, _ = hex.DecodeString(spec.JWT.Secret)
	}
	v := &OAuth2Validator{spec: spec}
	if spec.TokenIntrospect != nil {
		if spec.TokenIntrospect.InsecureTLS {
			cfg := tls.Config{InsecureSkipVerify: true}
			v.client = &http.Client{Transport: &http.Transport{TLSClientConfig: &cfg}}
		} else {
			v.client = http.DefaultClient
		}
	}
	return v
}

func (v *OAuth2Validator) introspectToken(tokenStr string) (*tokenInfo, error) {
	var body bytes.Buffer
	body.WriteString("token=")
	body.WriteString(tokenStr)
	if v.spec.TokenIntrospect.ClientID != "" {
		body.WriteString("&client_id=")
		body.WriteString(v.spec.TokenIntrospect.ClientID)
		body.WriteString("&client_secret=")
		body.WriteString(v.spec.TokenIntrospect.ClientSecret)
	}

	r, _ := http.NewRequest(http.MethodPost, v.spec.TokenIntrospect.EndPoint, &body)
	if v.spec.TokenIntrospect.ClientID != "" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if v.spec.TokenIntrospect.BasicAuth != "" {
		r.Header.Set("Authorization", "Basic "+v.spec.TokenIntrospect.BasicAuth)
	}

	resp, e := v.client.Do(r)
	if e != nil {
		return nil, e
	}

	var ti struct {
		tokenInfo
		Error     string `json:"error"`
		ErrorDesc string `json:"error_description"`
	}

	if e = json.NewDecoder(resp.Body).Decode(&ti); e != nil {
		return nil, e
	}
	if ti.Error != "" {
		return nil, fmt.Errorf("%s: %s", ti.Error, ti.ErrorDesc)
	}

	return &ti.tokenInfo, nil
}

// Validate validates the access token of a http request
func (v *OAuth2Validator) Validate(req context.HTTPRequest) error {
	const prefix = "Bearer "

	hdr := req.Header()
	tokenStr := hdr.Get("Authorization")
	if !strings.HasPrefix(tokenStr, prefix) {
		return fmt.Errorf("unexpected authorization header: %s", tokenStr)
	}
	tokenStr = tokenStr[len(prefix):]

	var subject, scope string
	if v.spec.TokenIntrospect != nil {
		ti, e := v.introspectToken(tokenStr)
		if e != nil {
			return e
		}
		if !ti.Active {
			return fmt.Errorf("oauth2 authorization failed, token is inactive")
		}
		subject = ti.Subject
		scope = ti.Scope
	} else {
		token, e := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if alg := token.Method.Alg(); alg != v.spec.JWT.Algorithm {
				return nil, fmt.Errorf("unexpected signing method: %v", alg)
			}
			return v.spec.JWT.secretBytes, nil
		})
		if e != nil {
			return e
		}

		claims := token.Claims.(jwt.MapClaims)
		subject, _ = claims["sub"].(string)
		scope, _ = claims["scope"].(string)
	}

	if subject != "" {
		hdr.Set("X-Authenticated-Userid", subject)
	}

	if scope != "" {
		hdr.Set("X-Authenticated-Scope", scope)
	}

	return nil
}
