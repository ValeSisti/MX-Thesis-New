package common

import "errors"

// ErrInvalidAPIType signals that an invalid api type has been provided
var ErrInvalidAPIType = errors.New("invalid api type provided")

// ErrInvalidConnectorType signals that an invalid observer connector type has been provided
var ErrInvalidConnectorType = errors.New("invalid observer connector type provided")

// ErrInvalidPubKeyConverterType signals that an invalid pubkey converter type has been provided
var ErrInvalidPubKeyConverterType = errors.New("invalid pubkey converter type provided")

// ErrInvalidDispatchType signals that an invalid dispatch type has been provided
var ErrInvalidDispatchType = errors.New("invalid dispatch type")

// ErrInvalidRedisConnType signals that an invalid redis connection type has been provided
var ErrInvalidRedisConnType = errors.New("invalid redis connection type")

// ErrReceivedEmptyEvents signals that empty events have been received
var ErrReceivedEmptyEvents = errors.New("received empty events")

// ErrNilMarshaller signals that a nil marshaller has been provided
var ErrNilMarshaller = errors.New("nil marshaller provided")

// ErrNilInternalMarshaller signals that a nil internal marshaller has been provided
var ErrNilInternalMarshaller = errors.New("nil external marshaller provided")

// ErrNilFacadeHandler signals that a nil facade handler has been provided
var ErrNilFacadeHandler = errors.New("nil facade handler")

// ErrNilStatusMetricsHandler signals that a nil status metrics handler has been provided
var ErrNilStatusMetricsHandler = errors.New("nil status metrics handler")

// ErrWrongTypeAssertion signals a wrong type assertion
var ErrWrongTypeAssertion = errors.New("wrong type assertion")

// ErrLoopAlreadyStarted signals that a loop has already been started
var ErrLoopAlreadyStarted = errors.New("loop already started")
