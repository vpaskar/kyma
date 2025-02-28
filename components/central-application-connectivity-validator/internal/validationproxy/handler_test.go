package validationproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma/common/logging/logger"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appconnv1alpha1 "github.com/kyma-project/kyma/components/application-operator/pkg/apis/applicationconnector/v1alpha1"
)

const (
	applicationName     = "test-application"
	applicationMetaName = "test-application-meta"
	applicationID       = "test-application-id"

	appNamePlaceholder             = "%%APP_NAME%%"
	eventingPathPrefixV1           = "/%%APP_NAME%%/v1/events"
	eventingPathPrefixV2           = "/%%APP_NAME%%/v2/events"
	eventingPathPrefixEvents       = "/%%APP_NAME%%/events"
	eventingDestinationPathPublish = "/publish"
)

type event struct {
	Title string `json:"title"`
}

var (
	applicationManagedByCompass = &appconnv1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "applicationconnector.kyma-project.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationMetaName,
		},
		Spec: appconnv1alpha1.ApplicationSpec{
			Description:     "Description",
			Services:        []appconnv1alpha1.Service{},
			CompassMetadata: &appconnv1alpha1.CompassMetadata{Authentication: appconnv1alpha1.Authentication{ClientIds: []string{applicationID}}},
		},
	}
	applicationNotManagedByCompass = &appconnv1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "applicationconnector.kyma-project.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: appconnv1alpha1.ApplicationSpec{
			Description: "Description",
			Services:    []appconnv1alpha1.Service{},
		},
	}
)

func TestProxyHandler_ProxyAppConnectorRequests(t *testing.T) {

	log, err := logger.New(logger.TEXT, logger.ERROR)
	require.NoError(t, err)
	testCases := []struct {
		caseDescription string
		tenant          string
		group           string
		certInfoHeader  string
		expectedStatus  int
		application     *appconnv1alpha1.Application
	}{
		{
			caseDescription: "Application without group and tenant",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=test-application-id,OU=OrgUnit,O=Organization,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusOK,
			application:    applicationManagedByCompass,
		},
		{
			caseDescription: "Application without group and tenant and with invalid Common Name",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=invalid-cn,OU=OrgUnit,O=Organization,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusForbidden,
			application:    applicationManagedByCompass,
		},
		{
			caseDescription: "Application with group and tenant",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=test-application-id,OU=group,O=tenant,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusOK,
			application:    applicationManagedByCompass,
		},
		{
			caseDescription: "Application with group, tenant and invalid Common Name",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=invalid-application,OU=group,O=tenant,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusForbidden,
			application:    applicationManagedByCompass,
		},
		{
			caseDescription: "X-Forwarded-Client-Cert header not specified",
			expectedStatus:  http.StatusInternalServerError,
			application:     applicationManagedByCompass,
		},
		{
			caseDescription: "Application not managed by Compass Runtime Agent without group and tenant",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=test-application,OU=OrgUnit,O=Organization,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusOK,
			application:    applicationNotManagedByCompass,
		},
		{
			caseDescription: "Application not managed by Compass Runtime Agent without group and tenant and with invalid Common Name",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=invalid-cn,OU=OrgUnit,O=Organization,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusForbidden,
			application:    applicationNotManagedByCompass,
		},
		{
			caseDescription: "Application not managed by Compass Runtime Agent with group and tenant",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=test-application,OU=group,O=tenant,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusOK,
			application:    applicationNotManagedByCompass,
		},
		{
			caseDescription: "Application not managed by Compass Runtime Agent with group, tenant and invalid Common Name",
			certInfoHeader: `Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=invalid-application,OU=group,O=tenant,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`,
			expectedStatus: http.StatusForbidden,
			application:    applicationNotManagedByCompass,
		},
	}

	t.Run("should proxy requests", func(t *testing.T) {
		const mockIncomingRequestHost = "fake.istio.gateway"
		const eventTitle = "my-event"
		eventPublisherProxyHandler := mux.NewRouter()
		eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
		eventPublisherProxyHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")

		// publish handler which are overwritten in the tests
		var publishHandler http.HandlerFunc
		eventPublisherProxyHandler.Path(eventingPathPrefixEvents).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			publishHandler.ServeHTTP(writer, request)
		})

		eventPublisherProxyHandler.PathPrefix(eventingDestinationPathPublish).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var receivedEvent event

			err := json.NewDecoder(r.Body).Decode(&receivedEvent)
			require.NoError(t, err)
			assert.Equal(t, eventTitle, receivedEvent.Title)

			assert.NotEqual(t, mockIncomingRequestHost, r.Host, "proxy should rewrite Host field")

			w.WriteHeader(http.StatusOK)
		})

		for _, testCase := range testCases {
			// given
			idCache := cache.New(time.Minute, time.Minute)
			if testCase.application.Spec.CompassMetadata != nil {
				idCache.Set(testCase.application.Name, []string{applicationID}, cache.NoExpiration)
			} else {
				idCache.Set(testCase.application.Name, []string{}, cache.NoExpiration)
			}

			proxyHandler := NewProxyHandler(
				appNamePlaceholder,
				eventingPathPrefixV1,
				eventingPathPrefixV2,
				eventingPathPrefixEvents,
				eventPublisherProxyHost,
				eventingDestinationPathPublish,
				idCache,
				log)

			t.Run("should proxy eventing V1 request when "+testCase.caseDescription, func(t *testing.T) {
				eventTitle := "my-event-1"

				eventPublisherProxyHandler.PathPrefix("/{application}/v1/events").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					appName := mux.Vars(r)["application"]
					assert.Equal(t, testCase.application.Name, appName, `Error reading "application" route variable from request context`)

					var receivedEvent event

					err := json.NewDecoder(r.Body).Decode(&receivedEvent)
					require.NoError(t, err)
					assert.Equal(t, eventTitle, receivedEvent.Title)

					w.WriteHeader(http.StatusOK)
				})

				body, err := json.Marshal(event{Title: eventTitle})
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/v1/events", testCase.application.Name), bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
				req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})

				recorder := httptest.NewRecorder()

				// when
				proxyHandler.ProxyAppConnectorRequests(recorder, req)

				// then
				assert.Equal(t, testCase.expectedStatus, recorder.Code)
			})

			t.Run("should proxy eventing V2 request when "+testCase.caseDescription, func(t *testing.T) {
				body, err := json.Marshal(event{Title: eventTitle})
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/v2/events", testCase.application.Name), bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
				req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})

				recorder := httptest.NewRecorder()

				// when
				proxyHandler.ProxyAppConnectorRequests(recorder, req)

				// then
				assert.Equal(t, testCase.expectedStatus, recorder.Code)
			})

			t.Run("should proxy eventing request when "+testCase.caseDescription, func(t *testing.T) {

				body, err := json.Marshal(event{Title: eventTitle})
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/events", testCase.application.Name), bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
				req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})

				// mock request Host to assert it gets rewritten by the proxy
				req.Host = mockIncomingRequestHost

				recorder := httptest.NewRecorder()

				// when
				proxyHandler.ProxyAppConnectorRequests(recorder, req)

				// then
				assert.Equal(t, testCase.expectedStatus, recorder.Code)
			})
		}
	})

	t.Run("should return 500 failed when cache doesn't contain the element", func(t *testing.T) {
		eventPublisherProxyHandler := mux.NewRouter()
		eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
		eventingPublisherHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")

		for _, testCase := range testCases {
			// given
			idCache := cache.New(time.Minute, time.Minute)

			proxyHandler := NewProxyHandler(
				appNamePlaceholder,
				eventingPathPrefixV1,
				eventingPathPrefixV2,
				eventingPathPrefixEvents,
				eventingPublisherHost,
				eventingDestinationPathPublish,
				idCache,
				log)

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%s/v1/metadata/services", testCase.application.Name), nil)
			require.NoError(t, err)
			req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
			req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})
			recorder := httptest.NewRecorder()

			// when
			proxyHandler.ProxyAppConnectorRequests(recorder, req)

			// then
			assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		}
	})

	t.Run("should return 400 when application not specified in path", func(t *testing.T) {
		eventPublisherProxyHandler := mux.NewRouter()
		eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
		eventingPublisherHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")

		// given
		certInfoHeader :=
			`Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=test-application,OU=OrgUnit,O=Organization,L=Waldorf,ST=Waldorf,C=DE";URI=`

		idCache := cache.New(time.Minute, time.Minute)
		idCache.Set(applicationName, []string{}, cache.NoExpiration)

		proxyHandler := NewProxyHandler(
			appNamePlaceholder,
			eventingPathPrefixV1,
			eventingPathPrefixV2,
			eventingPathPrefixEvents,
			eventingPublisherHost,
			eventingDestinationPathPublish,
			idCache,
			log)

		req, err := http.NewRequest(http.MethodGet, "/path", nil)
		require.NoError(t, err)
		req.Header.Set(CertificateInfoHeader, certInfoHeader)
		recorder := httptest.NewRecorder()

		// when
		proxyHandler.ProxyAppConnectorRequests(recorder, req)

		// then
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("should return 404 when path is invalid", func(t *testing.T) {
		eventPublisherProxyHandler := mux.NewRouter()
		eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
		eventingPublisherHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")

		// given
		certInfoHeader :=
			`Hash=f4cf22fb633d4df500e371daf703d4b4d14a0ea9d69cd631f95f9e6ba840f8ad;Subject="CN=test-application-id,OU=OrgUnit,O=Organization,L=Waldorf,ST=Waldorf,C=DE";` +
				`URI=,By=spiffe://cluster.local/ns/kyma-integration/sa/default;` +
				`Hash=6d1f9f3a6ac94ff925841aeb9c15bb3323014e3da2c224ea7697698acf413226;Subject="";` +
				`URI=spiffe://cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account`

		// mock cache sync controller that it fills cache
		idCache := cache.New(time.Minute, time.Minute)
		idCache.Set(applicationMetaName, []string{applicationID}, cache.NoExpiration)

		proxyHandler := NewProxyHandler(
			appNamePlaceholder,
			eventingPathPrefixV1,
			eventingPathPrefixV2,
			eventingPathPrefixEvents,
			eventingPublisherHost,
			eventingDestinationPathPublish,
			idCache,
			log)

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/%s/v1/bad/path", applicationMetaName), nil)
		require.NoError(t, err)
		req.Header.Set(CertificateInfoHeader, certInfoHeader)
		req = mux.SetURLVars(req, map[string]string{"application": applicationMetaName})
		recorder := httptest.NewRecorder()

		// when
		proxyHandler.ProxyAppConnectorRequests(recorder, req)

		// then
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("should proxy requests to Event Publisher Proxy(EPP) when BEB is enabled", func(t *testing.T) {

		eventPublisherProxyHandler := mux.NewRouter()
		eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
		eventingPublisherHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")

		for _, testCase := range testCases {
			// given
			idCache := cache.New(time.Minute, time.Minute)
			if testCase.application.Spec.CompassMetadata != nil {
				idCache.Set(testCase.application.Name, []string{applicationID}, cache.NoExpiration)
			} else {
				idCache.Set(testCase.application.Name, []string{}, cache.NoExpiration)
			}

			t.Run("should proxy requests in V1 to V1 endpoint of EPP when "+testCase.caseDescription, func(t *testing.T) {

				proxyHandlerBEB := NewProxyHandler(
					appNamePlaceholder,
					eventingPathPrefixV1,
					eventingPathPrefixV2,
					eventingPathPrefixEvents,
					eventingPublisherHost,
					eventingDestinationPathPublish,
					idCache,
					log)
				eventTitle := "my-event-1"

				eventPublisherProxyHandler.PathPrefix("/{application}/v1/events").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					appName := mux.Vars(r)["application"]
					assert.Equal(t, testCase.application.Name, appName, `Error reading "application" route variable from request context`)

					var receivedEvent event

					err := json.NewDecoder(r.Body).Decode(&receivedEvent)
					require.NoError(t, err)
					assert.Equal(t, eventTitle, receivedEvent.Title)

					w.WriteHeader(http.StatusOK)
				})

				body, err := json.Marshal(event{Title: eventTitle})
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/v1/events", testCase.application.Name), bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
				req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})

				recorder := httptest.NewRecorder()

				// when
				proxyHandlerBEB.ProxyAppConnectorRequests(recorder, req)

				// then
				assert.Equal(t, testCase.expectedStatus, recorder.Code)
			})

			t.Run("should proxy requests in V2 to /publish endpoint of EPP when "+testCase.caseDescription, func(t *testing.T) {
				eventTitle := "my-event-2"

				eventPublisherProxyHandler := mux.NewRouter()
				eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
				eventPublisherProxyHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")

				proxyHandlerBEB := NewProxyHandler(
					appNamePlaceholder,
					eventingPathPrefixV1,
					eventingPathPrefixV2,
					eventingPathPrefixEvents,
					eventPublisherProxyHost, // For a BEB enabled cluster requests to /v2 and /events should be forwarded to Event Publisher Proxy
					eventingDestinationPathPublish,
					idCache,
					log)

				eventPublisherProxyHandler.PathPrefix("/publish").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var receivedEvent event

					err := json.NewDecoder(r.Body).Decode(&receivedEvent)
					require.NoError(t, err)
					assert.Equal(t, eventTitle, receivedEvent.Title)

					w.WriteHeader(http.StatusOK)
				})

				body, err := json.Marshal(event{Title: eventTitle})
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/v2/events", testCase.application.Name), bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
				req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})

				recorder := httptest.NewRecorder()

				// when
				proxyHandlerBEB.ProxyAppConnectorRequests(recorder, req)

				// then
				assert.Equal(t, testCase.expectedStatus, recorder.Code)
			})

			t.Run("should proxy requests in /events to /publish endpoint of EPP when "+testCase.caseDescription, func(t *testing.T) {
				const eventTitle = "my-event"
				const mockIncomingRequestHost = "fake.istio.gateway"

				eventPublisherProxyHandler := mux.NewRouter()
				eventPublisherProxyServer := httptest.NewServer(eventPublisherProxyHandler)
				eventPublisherProxyHost := strings.TrimPrefix(eventPublisherProxyServer.URL, "http://")
				proxyHandlerBEB := NewProxyHandler(
					appNamePlaceholder,
					eventingPathPrefixV1,
					eventingPathPrefixV2,
					eventingPathPrefixEvents,
					eventPublisherProxyHost, // For a BEB enabled cluster requests to /v2 and /events should be forwarded to Event Publisher Proxy
					eventingDestinationPathPublish,
					idCache,
					log)

				eventPublisherProxyHandler.Path("/publish").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var receivedEvent event

					err := json.NewDecoder(r.Body).Decode(&receivedEvent)
					require.NoError(t, err)
					assert.Equal(t, eventTitle, receivedEvent.Title)

					assert.NotEqual(t, mockIncomingRequestHost, r.Host, "proxy should rewrite Host field")

					w.WriteHeader(http.StatusOK)
				})

				body, err := json.Marshal(event{Title: eventTitle})
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/events", testCase.application.Name), bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set(CertificateInfoHeader, testCase.certInfoHeader)
				req = mux.SetURLVars(req, map[string]string{"application": testCase.application.Name})

				// mock request Host to assert it gets rewritten by the proxy
				req.Host = mockIncomingRequestHost

				recorder := httptest.NewRecorder()

				// when
				proxyHandlerBEB.ProxyAppConnectorRequests(recorder, req)

				// then
				assert.Equal(t, testCase.expectedStatus, recorder.Code)
			})
		}
	})

}

func TestProxyHandler_ReplaceAppNamePlaceholder(t *testing.T) {
	log, err := logger.New(logger.TEXT, logger.ERROR)
	if err != nil {
		t.Error(err)
	}
	idCache := cache.New(time.Minute, time.Minute)

	ph := NewProxyHandler(
		appNamePlaceholder,
		eventingPathPrefixV1,
		eventingPathPrefixV2,
		eventingPathPrefixEvents,
		"N/A",
		"N/A",
		idCache,
		log)

	assert.Equal(t, "/commerce-mock/v1/events", ph.getApplicationPrefix(ph.eventingPathPrefixV1, "commerce-mock"))
	assert.Equal(t, "/commerce-mock/v2/events", ph.getApplicationPrefix(ph.eventingPathPrefixV2, "commerce-mock"))
	assert.Equal(t, "/commerce-mock/events", ph.getApplicationPrefix(ph.eventingPathPrefixEvents, "commerce-mock"))
}
