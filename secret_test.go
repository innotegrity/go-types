package types_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.innotegrity.dev/types"
)

// randomString just generates a random string of characters for test file names, secrets, etc.
func randomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	sb := strings.Builder{}
	sb.Grow(length)
	for range length {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// UserPasswordSecret is used for marshaling test data to JSON because [types.UsernamePasswordSecret] obscures
// the information with *** characters.
type UsernamePasswordSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TestUsernamePasswordSecret tests the [types.UsernamePasswordSecret] object formats.
//
// This function tests the following formats:
//   - awssecret://AWS_SECRET_NAME - name/path of the secret stored in AWS Secrets Manager containing the
//     username and password stored as JSON using the keys "username" and "password"
//   - base64://DATA - where DATA is a string formatted as username:password and then base64-encoded
//   - env://USER_VAR:PASS_VAR - where USER_VAR is the environment variable containing the username and PASS_VAR is
//     the environment variable containing the password
//   - env+base64://USER_VAR:PASS_VAR - where USER_VAR is the environment variable containing the username encoded in
//     base64 and PASS_VAR is the environment variable containing the password encoded in base64
//   - file://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the credentials stored as JSON using the
//     keys "username" and "password"
//   - file+base64://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the credentials stored as JSON using
//     the keys "username" and "password", each of which are base64-encoded
//   - raw://DATA - where DATA is a string formatted as username:password
func TestUsernamePasswordSecret(t *testing.T) {
	ctx := context.Background()
	username := "admin"
	password := "password"
	creds := UsernamePasswordSecret{
		Username: username,
		Password: password,
	}
	base64Creds := UsernamePasswordSecret{
		Username: base64.StdEncoding.EncodeToString([]byte(username)),
		Password: base64.StdEncoding.EncodeToString([]byte(password)),
	}
	baseName := randomString(15) + "_" + time.Now().Format("20060102_150405")
	expectedOutput := types.UsernamePasswordSecret{
		Username: username,
		Password: password,
	}

	// create each secret
	rawData := fmt.Sprintf("%s:%s", username, password)
	rawURL := fmt.Sprintf("raw://%s", rawData)

	base64Data := base64.StdEncoding.EncodeToString([]byte(rawData))
	base64URL := fmt.Sprintf("base64://%s", base64Data)

	os.Setenv("username", username)
	os.Setenv("password", password)
	envURL := "env://username:password"

	os.Setenv("username_b64", base64.StdEncoding.EncodeToString([]byte(username)))
	os.Setenv("password_b64", base64.StdEncoding.EncodeToString([]byte(password)))
	envBase64URL := "env+base64://username_b64:password_b64"

	data, _ := json.Marshal(creds)
	fileName := fmt.Sprintf("./%s", baseName)
	if err := os.WriteFile(fileName, data, 0640); err != nil {
		t.Fatalf("[file] failed to write secret file '%s': %s", fileName, err.Error())
	}
	fileURL := fmt.Sprintf("file://%s", fileName)
	defer os.Remove(fileName)

	data, _ = json.Marshal(base64Creds)
	fileName = fmt.Sprintf("./%s_b64", baseName)
	if err := os.WriteFile(fileName, data, 0640); err != nil {
		t.Fatalf("[file+base64] failed to write secret file '%s': %s", fileName, err.Error())
	}
	fileBase64URL := fmt.Sprintf("file+base64://%s", fileName)
	defer os.Remove(fileName)

	data, _ = json.Marshal(creds)
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("[awssecret] failed to load AWS config: %s", err.Error())
	}
	client := secretsmanager.NewFromConfig(awsConfig)
	secretName := fmt.Sprintf("types_test_%s", baseName)
	value := string(data)
	input := &secretsmanager.CreateSecretInput{
		Name:         &secretName,
		SecretString: &value,
	}
	output, err := client.CreateSecret(ctx, input)
	if err != nil {
		t.Fatalf("[awssecret] failed to store secret '%s': %s", secretName, err.Error())
	}
	defer func() {
		recoveryDays := int64(7)
		client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:             output.ARN,
			RecoveryWindowInDays: &recoveryDays,
		})
	}()
	awsSecretURL := fmt.Sprintf("awssecret://%s", secretName)

	// now let's test parsing each secret
	s, xerr := types.ParseUsernamePasswordSecret(ctx, rawURL)
	if xerr != nil {
		t.Errorf("[raw] failed to parse URL '%s': %s", rawURL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[raw] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[raw] Passed")
	}

	s, xerr = types.ParseUsernamePasswordSecret(ctx, base64URL)
	if xerr != nil {
		t.Errorf("[base64] failed to parse URL '%s': %s", base64URL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[base64] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[base64] Passed")
	}

	s, xerr = types.ParseUsernamePasswordSecret(ctx, envURL)
	if xerr != nil {
		t.Errorf("[env] failed to parse URL '%s': %s", envURL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[env] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[env] Passed")
	}

	s, xerr = types.ParseUsernamePasswordSecret(ctx, envBase64URL)
	if xerr != nil {
		t.Errorf("[env+base64] failed to parse URL '%s': %s", envBase64URL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[env+base64] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[env+base64] Passed")
	}

	s, xerr = types.ParseUsernamePasswordSecret(ctx, fileURL)
	if xerr != nil {
		t.Errorf("[file] failed to parse URL '%s': %s", fileURL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[file] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[file] Passed")
	}

	s, xerr = types.ParseUsernamePasswordSecret(ctx, fileBase64URL)
	if xerr != nil {
		t.Errorf("[file+base64] failed to parse URL '%s': %s", fileBase64URL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[file+base64] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[file+base64] Passed")
	}

	s, xerr = types.ParseUsernamePasswordSecret(ctx, awsSecretURL)
	if xerr != nil {
		t.Errorf("[awssecret] failed to parse raw URL '%s': %s", awsSecretURL, xerr.Error())
	} else if s != expectedOutput {
		t.Errorf("[awssecret] got --> %s / %s ; expected --> %s / %s", s.Username, s.Password,
			expectedOutput.Username, expectedOutput.Password)
	} else {
		t.Logf("[awssecret] Passed")
	}
}

// TestGenericSecret tests the [types.GenericSecret] object formats.
//
// This function tests the following formats:
//   - awssecret://AWS_SECRET_NAME - name/path of the secret stored in AWS Secrets Manager containing the
//     raw secret as a string
//   - awssecret+binary://AWS_SECRET_NAME - name/path of the secret stored in AWS Secrets Manager containing the
//     raw secret as binary data
//   - base64://DATA - where DATA is a string that's simply base64-encoded
//   - env://VAR_NAME - where VAR_NAME is the environment variable containing the secret
//   - env+base64://VAR_NAME - where VAR_NAME is the environment variable containing the base64-encoded secret
//   - file://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the secret data
//   - file+base64://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the base64-encoded secret data
//   - raw://DATA - where DATA is the raw secret
func TestGenericSecret(t *testing.T) {
	ctx := context.Background()
	secretData := "some super secret data"
	baseName := randomString(15) + "_" + time.Now().Format("20060102_150405")
	expectedOutput := types.GenericSecret{
		Data: []byte(secretData),
	}

	// create each secret
	rawData := secretData
	rawURL := fmt.Sprintf("raw://%s", rawData)

	base64Data := base64.StdEncoding.EncodeToString([]byte(rawData))
	base64URL := fmt.Sprintf("base64://%s", base64Data)

	os.Setenv("secret", secretData)
	envURL := "env://secret"

	os.Setenv("secret_b64", base64.StdEncoding.EncodeToString([]byte(secretData)))
	envBase64URL := "env+base64://secret_b64"

	fileName := fmt.Sprintf("./%s", baseName)
	if err := os.WriteFile(fileName, []byte(secretData), 0640); err != nil {
		t.Fatalf("[file] failed to write secret file '%s': %s", fileName, err.Error())
	}
	fileURL := fmt.Sprintf("file://%s", fileName)
	defer os.Remove(fileName)

	fileName = fmt.Sprintf("./%s_b64", baseName)
	if err := os.WriteFile(fileName, []byte(base64.StdEncoding.EncodeToString([]byte(secretData))),
		0640); err != nil {
		t.Fatalf("[file+base64] failed to write secret file '%s': %s", fileName, err.Error())
	}
	fileBase64URL := fmt.Sprintf("file+base64://%s", fileName)
	defer os.Remove(fileName)

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("[awssecret/awssecret+binary] failed to load AWS config: %s", err.Error())
	}
	client := secretsmanager.NewFromConfig(awsConfig)
	secretName := fmt.Sprintf("types_test_%s", baseName)
	input := &secretsmanager.CreateSecretInput{
		Name:         &secretName,
		SecretString: &secretData,
	}
	output, err := client.CreateSecret(ctx, input)
	if err != nil {
		t.Fatalf("[awssecret] failed to store secret '%s': %s", secretName, err.Error())
	}
	defer func() {
		recoveryDays := int64(7)
		client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:             output.ARN,
			RecoveryWindowInDays: &recoveryDays,
		})
	}()
	awsSecretURL := fmt.Sprintf("awssecret://%s", secretName)

	secretName = fmt.Sprintf("types_test_bin_%s", baseName)
	input = &secretsmanager.CreateSecretInput{
		Name:         &secretName,
		SecretBinary: []byte(secretData),
	}
	binOutput, err := client.CreateSecret(ctx, input)
	if err != nil {
		t.Fatalf("[awssecret+binary] failed to store secret '%s': %s", secretName, err.Error())
	}
	defer func() {
		recoveryDays := int64(7)
		client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:             binOutput.ARN,
			RecoveryWindowInDays: &recoveryDays,
		})
	}()
	awsBinarySecretURL := fmt.Sprintf("awssecret+binary://%s", secretName)

	// now let's test parsing each secret
	s, xerr := types.ParseGenericSecret(ctx, rawURL)
	if xerr != nil {
		t.Errorf("[raw] failed to parse URL '%s': %s", rawURL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[raw] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[raw] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, base64URL)
	if xerr != nil {
		t.Errorf("[base64] failed to parse URL '%s': %s", base64URL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[base64] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[base64] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, envURL)
	if xerr != nil {
		t.Errorf("[env] failed to parse URL '%s': %s", envURL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[env] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[env] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, envBase64URL)
	if xerr != nil {
		t.Errorf("[env+base64] failed to parse URL '%s': %s", envBase64URL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[env+base64] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[env+base64] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, fileURL)
	if xerr != nil {
		t.Errorf("[file] failed to parse URL '%s': %s", fileURL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[file] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[file] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, fileBase64URL)
	if xerr != nil {
		t.Errorf("[file+base64] failed to parse URL '%s': %s", fileBase64URL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[file+base64] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[file+base64] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, awsSecretURL)
	if xerr != nil {
		t.Errorf("[awssecret] failed to parse URL '%s': %s", awsSecretURL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[awssecret] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[awssecret] Passed")
	}

	s, xerr = types.ParseGenericSecret(ctx, awsBinarySecretURL)
	if xerr != nil {
		t.Errorf("[awssecret+binary] failed to parse URL '%s': %s", awsBinarySecretURL, xerr.Error())
	} else if !bytes.Equal(s.Data, expectedOutput.Data) {
		t.Errorf("[awssecret+binary] got --> %s ; expected --> %s", s.Data, expectedOutput.Data)
	} else {
		t.Logf("[awssecret+binary] Passed")
	}
}
