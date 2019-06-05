package apistrategy

import (
	"github.com/ibm-cloud-security/policy-enforcer-mixer-adapter/adapter/authserver/keyset"
	"github.com/ibm-cloud-security/policy-enforcer-mixer-adapter/adapter/policy"
	"github.com/ibm-cloud-security/policy-enforcer-mixer-adapter/config/template"
	"testing"

	"github.com/gogo/googleapis/google/rpc"
	"github.com/ibm-cloud-security/policy-enforcer-mixer-adapter/adapter/errors"
	"github.com/stretchr/testify/assert"
	"istio.io/api/policy/v1beta1"
)

func TestNew(t *testing.T) {
	strategy := New()
	assert.NotNil(t, strategy)
}

func TestHandleAuthorizationRequest(t *testing.T) {
	var tests = []struct {
		req           *authnz.HandleAuthnZRequest
		action        *policy.Action
		message       string
		code          int32
		validationErr *errors.OAuthError
	}{
		{
			generateAuthRequest(""),
			&policy.Action{},
			"authorization header not provided",
			int32(16),
			nil,
		},
		{
			generateAuthRequest("bearer"),
			&policy.Action{},
			"authorization header malformed - expected 'Bearer <access_token> <optional id_token>'",
			int32(16),
			nil,
		},
		{
			generateAuthRequest("Bearer invalid"),
			&policy.Action{},
			"invalid access token",
			int32(16),
			errors.UnauthorizedHTTPException("invalid access token", nil),
		},
		{
			generateAuthRequest("Bearer access"),
			&policy.Action{},
			"",
			int32(0),
			nil,
		},
		{
			generateAuthRequest("Bearer access id"),
			&policy.Action{},
			"",
			int32(0),
			nil,
		},
	}

	for _, test := range tests {
		t.Run("Validation Test", func(st *testing.T) {
			api := APIStrategy{
				tokenUtil: MockValidator{
					err: test.validationErr,
				},
			}
			checkresult, err := api.HandleAuthnZRequest(test.req, test.action)
			assert.Nil(st, err)
			assert.Equal(st, test.message, checkresult.Result.Status.Message)
			assert.Equal(st, test.code, checkresult.Result.Status.Code)
		})
	}
}

func TestParseRequest(t *testing.T) {
	var tests = []struct {
		r           *authnz.HandleAuthnZRequest
		expectErr   bool
		expectedMsg string
	}{
		{
			&authnz.HandleAuthnZRequest{
				Instance: &authnz.InstanceMsg{
					Request: &authnz.RequestMsg{
						Headers: &authnz.HeadersMsg{},
					},
				},
			},
			true,
			"authorization header not provided",
		},
		{
			generateAuthRequest(""),
			true,
			"authorization header not provided",
		},
		{
			generateAuthRequest("bearer"),
			true,
			"authorization header malformed - expected 'Bearer <access_token> <optional id_token>'",
		},
		{
			generateAuthRequest("b access id"),
			true,
			"unsupported authorization header format - expected 'Bearer <access_token> <optional id_token>'",
		},
		{
			generateAuthRequest("Bearer access id"),
			false,
			"",
		},
		{
			generateAuthRequest("bearer access id"),
			false,
			"",
		},
	}

	for _, e := range tests {
		t.Run("Parsing Test", func(st *testing.T) {
			tokens, err := getAuthTokensFromRequest(e.r)
			if !e.expectErr && tokens != nil {
				assert.Equal(st, "access", tokens.Access)
				assert.Equal(st, "id", tokens.ID)
			} else {
				assert.EqualError(st, err, e.expectedMsg)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	var tests = []struct {
		err          *errors.OAuthError
		Message      string
		ResponseCode v1beta1.HttpStatusCode
		Body         string
		HeadersMap   map[string]string
	}{
		{
			err: &errors.OAuthError{
				Msg:    "missing header",
				Code:   errors.InvalidToken,
				Scopes: []string{"scope"},
			},
			Message:      "missing header",
			ResponseCode: v1beta1.Unauthorized,
			Body:         errors.InvalidToken,
			HeadersMap:   make(map[string]string),
		},
	}
	for _, test := range tests {
		t.Run("Error Parse", func(st *testing.T) {
			checkresult := buildErrorResponse(test.err)
			assert.Equal(st, checkresult.Result.Status.Code, int32(rpc.UNAUTHENTICATED))
			assert.Equal(st, checkresult.Result.Status.Message, test.Message)
			assert.NotNil(st, checkresult.Result.Status.Details)
		})
	}
}

func generateAuthRequest(header string) *authnz.HandleAuthnZRequest {
	return &authnz.HandleAuthnZRequest{
		Instance: &authnz.InstanceMsg{
			Request: &authnz.RequestMsg{
				Headers: &authnz.HeadersMsg{
					Authorization: header,
				},
			},
		},
	}
}

type MockValidator struct {
	err *errors.OAuthError
}

func (v MockValidator) Validate(tkn string, ks keyset.KeySet, rules []policy.Rule) *errors.OAuthError {
	return v.err
}