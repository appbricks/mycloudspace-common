package mocks

import (
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-common/mycsnode"

	"github.com/mevansam/goutils/crypto"

	cb_mocks "github.com/appbricks/cloud-builder/test/mocks"
	utils_mocks "github.com/mevansam/goutils/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type MockNodeService struct {
	TestServer  *utils_mocks.MockHttpServer	
	TestTarget  *target.Target
	TestConfig  config.Config

	LoggedInUser *userspace.User
}

type MockServiceHandler struct {	
	tgt *target.Target

	ecdhKey       *crypto.ECDHKey
	encryptionKey []byte

	devicePublicKey *crypto.RSAPublicKey

	authIDKey string
}

var (
	testServerPort int32
)

func init() {
	testServerPort = 9000
}

func StartMockNodeServices() *MockNodeService {

	var (
		err error

		caRootPem string
	)

	svc := &MockNodeService{}

	// start different test server for each test
	testServerPort := int(atomic.AddInt32(&testServerPort, 1))
	svc.TestServer, caRootPem, err = utils_mocks.NewMockHttpsServer(testServerPort)
	Expect(err).ToNot(HaveOccurred())
	svc.TestServer.Start()

	cli := utils_mocks.NewFakeCLI(os.Stdout, os.Stderr)
	svc.TestTarget = cb_mocks.NewMockTarget(cli, "127.0.0.1", testServerPort, caRootPem)

	err = svc.TestTarget.LoadRemoteRefs()
	Expect(err).ToNot(HaveOccurred())

	deviceContext := config.NewDeviceContext()
	_, err = deviceContext.NewDevice()
	Expect(err).ToNot(HaveOccurred())
	svc.LoggedInUser, err = deviceContext.NewOwnerUser(loggedInUserID, "testuser")
	Expect(err).ToNot(HaveOccurred())
	deviceContext.SetDeviceID(deviceIDKey, deviceID, deviceName)
	deviceContext.SetLoggedInUser(loggedInUserID, "testuser");

	svc.TestConfig = cb_mocks.NewMockConfig(nil, deviceContext, nil)
	return svc
}

func (s *MockNodeService) Stop() {
	s.TestServer.Stop()
}

func (s *MockNodeService) NewApiClient() *mycsnode.ApiClient {	
	dc := s.TestConfig.DeviceContext()
	apiClient, err := mycsnode.NewApiClient(
		dc.GetDevice().Name,
		dc.GetLoggedInUserID(),
		dc.GetDeviceIDKey(),
		dc.GetDevice().RSAPrivateKey,
		s.TestTarget,
		"/auth",
	)	
	Expect(err).ToNot(HaveOccurred())
	return apiClient
}

func (s *MockNodeService) NewServiceHandler() *MockServiceHandler {

	var (
		err error
	)

	ecdhKey, err := crypto.NewECDHKey()
	Expect(err).ToNot(HaveOccurred())

	handler := &MockServiceHandler{
		tgt: s.TestTarget,
		ecdhKey: ecdhKey,
	}
	handler.devicePublicKey, err = crypto.NewPublicKeyFromPEM(s.TestConfig.DeviceContext().GetDevice().RSAPublicKey)
	Expect(err).ToNot(HaveOccurred())
	return handler
}

func (h *MockServiceHandler) SendAuthResponse(w http.ResponseWriter, r *http.Request, body string) *string {
	defer GinkgoRecover()

	var (
		err error
	)

	authRequest := &mycsnode.AuthRequest{}
	err = json.Unmarshal([]byte(body), &authRequest)
	Expect(err).ToNot(HaveOccurred())
	Expect(authRequest.AuthReqIDKey).To(Equal(deviceIDKey))

	// decrypt authReqKey payload
	key, err := crypto.NewRSAKeyFromPEM(h.tgt.RSAPrivateKey, nil)
	Expect(err).ToNot(HaveOccurred())

	authReqKeyJSON, err := key.DecryptBase64(authRequest.AuthReqKey)
	Expect(err).ToNot(HaveOccurred())

	authReqKey := &mycsnode.AuthReqKey{}
	err = json.Unmarshal(authReqKeyJSON, authReqKey)
	Expect(err).ToNot(HaveOccurred())

	Expect(authReqKey.RefID).To(Equal(loggedInUserID))
	Expect(authReqKey.Nonce).To(BeNumerically(">", 0))

	// create shared secret
	h.ecdhKey, err = crypto.NewECDHKey()
	Expect(err).ToNot(HaveOccurred())
	h.encryptionKey, err = h.ecdhKey.SharedSecret(authReqKey.ECDHKey)
	Expect(err).ToNot(HaveOccurred())

	ecdhPublicKey, err := h.ecdhKey.PublicKey()
	Expect(err).ToNot(HaveOccurred())

	// return shared secret and nonce
	authRespKey := &mycsnode.AuthRespKey{
		NodeECDHKey: ecdhPublicKey,
		Nonce: authReqKey.Nonce,
		// Nonce is in ms so need to convert it and add 2s
		TimeoutAt: int64(time.Duration(authReqKey.Nonce) * time.Millisecond + 2 * time.Second) / int64(time.Millisecond),
		RefName: deviceName,
	}
	authRespKeyJSON, err := json.Marshal(authRespKey)
	Expect(err).ToNot(HaveOccurred())
	encryptedAuthRespKey, err := h.devicePublicKey.EncryptBase64(authRespKeyJSON)
	Expect(err).ToNot(HaveOccurred())

	// auth id key
	h.authIDKey, err = key.PublicKey().EncryptBase64([]byte(authReqKey.RefID + "|" + deviceID))
	Expect(err).ToNot(HaveOccurred())

	authResponse := &mycsnode.AuthResponse{
		AuthRespKey: encryptedAuthRespKey,
		AuthRespIDKey: h.authIDKey,
	}
	authResponseJSON, err := json.Marshal(authResponse)
	Expect(err).ToNot(HaveOccurred())

	responseBody := string(authResponseJSON)
	return &responseBody
}

func (h *MockServiceHandler) ValidateEncryption(apiClient *mycsnode.ApiClient) {

	// validate encryption using shared key
	handlerCrypt, err := crypto.NewCrypt(h.encryptionKey)
	Expect(err).ToNot(HaveOccurred())
	cipher, err := handlerCrypt.EncryptB64("plain text test")
	Expect(err).ToNot(HaveOccurred())

	apiClientCrypt, _ := apiClient.Crypt()
	Expect(err).ToNot(HaveOccurred())
	plainText, err := apiClientCrypt.DecryptB64(cipher)
	Expect(err).ToNot(HaveOccurred())

	Expect(plainText).To(Equal("plain text test"))
}

const loggedInUserID = `7a4ae0c0-a25f-4376-9816-b45df8da5e88`
const deviceIDKey = `b1f187f2-1019-4848-ae7c-4db0cec1f256|F+IVHNUM85lwwLSfGdlZCR2gcDpzDs1wF6CcEjWOr2zL/Kr5Fw1Utu1BX2i+2p+b5v8sSfy9g1AdYZhHKLKI7qeXWg9n/E1r8YzCyunVeByiWpWpn51Afca+pg5wQMlnLD4Sy8SHRICZj9XDF/9MYna/iX8FKNtVEymOSceYVkgAuH/YypNLp48D6Wk9oOJGLb5OBiAnnpNqrLadQ3kbShoLvl41ynfkNX3pqOMj5Y2qWGOoFkiru+zch6xlit5XrKVIOpV/iWwjNJTOjCaNJ2bcuMNFcF6EA8DgnfQPjgR2CfJhoENoCSo7ieO9EAfQmZJS3fWPiIgo8tCGW7cneNWbWz5agKn5tjrmeGXkwkPDKnbRpTBLeZ6akNP2C6GncEHICXvbetP46DcoZjLBt5sPx8vQeQ3EYFehi4PDz6LuWvppAkMa2pmI4VTQIdRxUH4Rp23MgcKQ40vHRA7FDP4JSmyseRozfSksBXWjZIul0/QDV3yYvkKaeOqYWwQv+sZiV8ZFHVFQDYr8yBzvxR3WCyyJSP+jmWIfC32WHIwV1KTtxZXlYwGHs/JmScTcR4Gs9qTdemsdLIvro6wPmO6vsdMJqgp3NggzN3pkaIkvps+8tmGsqB7N7KxRmln9TFnKP3urp56CwnNzRKV8Z9tVBNxYJOnL1jxbVsMjniY=`
const deviceID = `676741a9-0608-4633-b293-05e49bea6504`
const deviceName = `Test Device`
