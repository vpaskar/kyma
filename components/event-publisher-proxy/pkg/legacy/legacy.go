package legacy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	cev2event "github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/kyma-project/kyma/components/event-publisher-proxy/internal"
	"github.com/kyma-project/kyma/components/event-publisher-proxy/pkg/application"
	apiv1 "github.com/kyma-project/kyma/components/event-publisher-proxy/pkg/legacy/api"
	"github.com/kyma-project/kyma/components/event-publisher-proxy/pkg/tracing"
)

var (
	isValidEventTypeVersion = regexp.MustCompile(AllowedEventTypeVersionChars).MatchString
	isValidEventID          = regexp.MustCompile(AllowedEventIDChars).MatchString
)

const (
	requestBodyTooLargeErrorMessage = "http: request body too large"
	eventTypeVersionExtensionKey    = "eventtypeversion"
)

type RequestToCETransformer interface {
	TransformLegacyRequestsToCE(http.ResponseWriter, *http.Request) (*cev2event.Event, string)
	TransformsCEResponseToLegacyResponse(http.ResponseWriter, int, *cev2event.Event, string)
}

type Transformer struct {
	bebNamespace      string
	eventTypePrefix   string
	applicationLister *application.Lister
}

func NewTransformer(bebNamespace string, eventTypePrefix string, applicationLister *application.Lister) *Transformer {
	return &Transformer{
		bebNamespace:      bebNamespace,
		eventTypePrefix:   eventTypePrefix,
		applicationLister: applicationLister,
	}
}

// CheckParameters validates the parameters in the request and sends error responses if found invalid
func (t *Transformer) checkParameters(parameters *apiv1.PublishEventParametersV1) (response *apiv1.PublishEventResponses) {
	if parameters == nil {
		return ErrorResponseBadRequest(ErrorMessageBadPayload)
	}
	if len(parameters.PublishrequestV1.EventType) == 0 {
		return ErrorResponseMissingFieldEventType()
	}
	if len(parameters.PublishrequestV1.EventTypeVersion) == 0 {
		return ErrorResponseMissingFieldEventTypeVersion()
	}
	if !isValidEventTypeVersion(parameters.PublishrequestV1.EventTypeVersion) {
		return ErrorResponseWrongEventTypeVersion()
	}
	if len(parameters.PublishrequestV1.EventTime) == 0 {

		return ErrorResponseMissingFieldEventTime()
	}
	if _, err := time.Parse(time.RFC3339, parameters.PublishrequestV1.EventTime); err != nil {
		return ErrorResponseWrongEventTime()
	}
	if len(parameters.PublishrequestV1.EventID) > 0 && !isValidEventID(parameters.PublishrequestV1.EventID) {
		return ErrorResponseWrongEventID()
	}
	if parameters.PublishrequestV1.Data == nil {
		return ErrorResponseMissingFieldData()
	}
	if d, ok := (parameters.PublishrequestV1.Data).(string); ok && len(d) == 0 {
		return ErrorResponseMissingFieldData()
	}
	// OK
	return &apiv1.PublishEventResponses{}
}

// TransformLegacyRequestsToCE transforms the legacy event to cloudevent from the given request.
// It also returns the original event-type without cleanup as the second return type.
func (t *Transformer) TransformLegacyRequestsToCE(writer http.ResponseWriter, request *http.Request) (*cev2event.Event, string) {
	// parse request body to PublishRequestV1
	if request.Body == nil || request.ContentLength == 0 {
		resp := ErrorResponseBadRequest(ErrorMessageBadPayload)
		writeJSONResponse(writer, resp)
		return nil, ""
	}

	parameters := &apiv1.PublishEventParametersV1{}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&parameters.PublishrequestV1); err != nil {
		var resp *apiv1.PublishEventResponses
		if err.Error() == requestBodyTooLargeErrorMessage {
			resp = ErrorResponseRequestBodyTooLarge(err.Error())
		} else {
			resp = ErrorResponseBadRequest(err.Error())
		}
		writeJSONResponse(writer, resp)
		return nil, ""
	}

	// validate the PublishRequestV1 for missing / incoherent values
	checkResp := t.checkParameters(parameters)
	if checkResp.Error != nil {
		writeJSONResponse(writer, checkResp)
		return nil, ""
	}

	// clean the application name form non-alphanumeric characters
	appNameOriginal := ParseApplicationNameFromPath(request.URL.Path)
	appName := appNameOriginal
	if appObj, err := t.applicationLister.Get(appName); err == nil {
		// handle existing applications
		appName = application.GetCleanTypeOrName(appObj)
	} else {
		// handle non-existing applications
		appName = application.GetCleanName(appName)
	}

	event, err := t.convertPublishRequestToCloudEvent(appName, parameters)
	if err != nil {
		response := ErrorResponse(http.StatusInternalServerError, err)
		writeJSONResponse(writer, response)
		return nil, ""
	}

	// Add tracing context to cloud events
	tracing.AddTracingContextToCEExtensions(request.Header, event)

	// prepare the original event-type without cleanup
	eventType := formatEventType(t.eventTypePrefix, appNameOriginal, parameters.PublishrequestV1.EventType, parameters.PublishrequestV1.EventTypeVersion)

	return event, eventType
}

func (t *Transformer) TransformsCEResponseToLegacyResponse(writer http.ResponseWriter, statusCode int, event *cev2event.Event, msg string) {
	response := &apiv1.PublishEventResponses{}
	// Fail
	if !is2XXStatusCode(statusCode) {
		response.Error = &apiv1.Error{
			Status:  statusCode,
			Message: msg,
		}
		writeJSONResponse(writer, response)
		return
	}

	// Success
	response.Ok = &apiv1.PublishResponse{EventID: event.ID()}
	writeJSONResponse(writer, response)
}

// convertPublishRequestToCloudEvent converts the given publish request to a CloudEvent.
func (t *Transformer) convertPublishRequestToCloudEvent(appName string, publishRequest *apiv1.PublishEventParametersV1) (*cev2event.Event, error) {
	if !application.IsCleanName(appName) {
		return nil, errors.New("application name should be cleaned from none-alphanumeric characters")
	}

	event := cev2event.New(cev2event.CloudEventsVersionV1)

	evTime, err := time.Parse(time.RFC3339, publishRequest.PublishrequestV1.EventTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse time from the external publish request")
	}
	event.SetTime(evTime)

	if err := event.SetData(internal.ContentTypeApplicationJSON, publishRequest.PublishrequestV1.Data); err != nil {
		return nil, errors.Wrap(err, "failed to set data to CloudEvent data field")
	}

	// set the event id from the request if it is available
	// otherwise generate a new one
	if len(publishRequest.PublishrequestV1.EventID) > 0 {
		event.SetID(publishRequest.PublishrequestV1.EventID)
	} else {
		event.SetID(uuid.New().String())
	}

	eventName := combineEventNameSegments(removeNonAlphanumeric(publishRequest.PublishrequestV1.EventType))
	prefix := removeNonAlphanumeric(t.eventTypePrefix)
	eventType := formatEventType(prefix, appName, eventName, publishRequest.PublishrequestV1.EventTypeVersion)
	event.SetType(eventType)
	event.SetSource(t.bebNamespace)
	event.SetExtension(eventTypeVersionExtensionKey, publishRequest.PublishrequestV1.EventTypeVersion)
	event.SetDataContentType(internal.ContentTypeApplicationJSON)
	return &event, nil
}

// combineEventNameSegments returns an eventName with exactly two segments separated by "." if the given event-type
// has more than two segments separated by "." (e.g. "Account.Order.Created" becomes "AccountOrder.Created")
func combineEventNameSegments(eventName string) string {
	parts := strings.Split(eventName, ".")
	if len(parts) > 2 {
		businessObject := strings.Join(parts[0:len(parts)-1], "")
		operation := parts[len(parts)-1]
		eventName = fmt.Sprintf("%s.%s", businessObject, operation)
	}
	return eventName
}

// removeNonAlphanumeric returns an eventName without any non-alphanumerical character besides dot (".")
func removeNonAlphanumeric(eventType string) string {
	return regexp.MustCompile("[^a-zA-Z0-9.]+").ReplaceAllString(eventType, "")
}
