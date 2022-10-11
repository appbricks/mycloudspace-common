package mycsnode_test

import (
	"time"

	mycs_mocks "github.com/appbricks/mycloudspace-common/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MyCS Node API Client", func() {

	var (
		err error

		mockNodeService *mycs_mocks.MockNodeService
	)

	BeforeEach(func() {
		mockNodeService = mycs_mocks.StartMockNodeServices()
	})

	AfterEach(func() {		
		mockNodeService.Stop()
	})

	Context("Authentication", func() {

		It("Creates an API client and authenticates", func() {
			apiClient := mockNodeService.NewApiClient()
			handler := mockNodeService.NewServiceHandler()
			Expect(err).ToNot(HaveOccurred())
			Expect(apiClient.IsRunning()).To(BeTrue())

			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				WithCallbackTest(handler.SendAuthResponse)

			isAuthenticated, err := apiClient.Authenticate()
			Expect(err).ToNot(HaveOccurred())
			Expect(isAuthenticated).To(BeTrue())
			Expect(mockNodeService.TestServer.Done()).To(BeTrue())
			handler.ValidateEncryption(apiClient)

			Expect(apiClient.IsAuthenticated()).To(BeTrue())
			time.Sleep(1000 * time.Millisecond)
			Expect(apiClient.IsAuthenticated()).To(BeTrue())
			time.Sleep(1000 * time.Millisecond)
			Expect(apiClient.IsAuthenticated()).To(BeFalse())

			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				WithCallbackTest(handler.SendAuthResponse)

			_, err = apiClient.Authenticate()
			Expect(err).ToNot(HaveOccurred())
			Expect(apiClient.IsAuthenticated()).To(BeTrue())
			Expect(mockNodeService.TestServer.Done()).To(BeTrue())
		})

		It("Creates an API client and keeps it authenticated in the background", func() {
			apiClient := mockNodeService.NewApiClient()
			handler := mockNodeService.NewServiceHandler()
			Expect(err).ToNot(HaveOccurred())
			Expect(apiClient.IsRunning()).To(BeTrue())

			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				RespondWithError(authErrorResponse, 400)
			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				RespondWithError(authErrorResponse, 400)
			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				WithCallbackTest(handler.SendAuthResponse)
			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				WithCallbackTest(handler.SendAuthResponse)
			mockNodeService.TestServer.PushRequest().
				ExpectPath("/auth").
				ExpectMethod("POST").
				WithCallbackTest(handler.SendAuthResponse)

			err = apiClient.Start()
			Expect(err).NotTo(HaveOccurred())
			Expect(apiClient.IsAuthenticated()).To(BeFalse())
			time.Sleep(2000 * time.Millisecond)
			Expect(apiClient.IsAuthenticated()).To(BeFalse())
			time.Sleep(2000 * time.Millisecond)
			Expect(apiClient.IsAuthenticated()).To(BeTrue())
			time.Sleep(2000 * time.Millisecond)
			Expect(apiClient.IsAuthenticated()).To(BeTrue())
			time.Sleep(2000 * time.Millisecond)
			Expect(apiClient.IsAuthenticated()).To(BeTrue())
			Expect(mockNodeService.TestServer.Done()).To(BeTrue())
			apiClient.Stop()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

const authErrorResponse = `{"errorCode":1001,"errorMessage":"Request Error"}`
