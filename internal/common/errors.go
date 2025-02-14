package common

import (
	"errors"
	"fmt"
)

const ErrorPrefix = "cvms common errors"

var ErrCanSkip = fmt.Errorf("skip")

var (
	ErrFailedToBuildPackager      = errors.New("failed to build the packger").Error()
	ErrUnDefinedSomeConfiguration = errors.New("undefinded port or something in your prometheus config file")
	ErrUnSupportedMessageType     = errors.New("unsupported message type")
	ErrUnSupportedEventType       = errors.New("unsupported event type")
	ErrUnDefinedApp               = fmt.Errorf("%s: undefinded app name", ErrorPrefix)
	ErrUnSupportedPackage         = fmt.Errorf("%s: this is unsupported monitoring package", ErrorPrefix)
	ErrUnSupportedMethod          = fmt.Errorf("%s: this is unsupported method", ErrorPrefix)
	ErrUnsetHttpSchema            = fmt.Errorf("%s: failed to unset http schema for grpc connecting", ErrorPrefix)
	ErrFailedHttpRequest          = fmt.Errorf("%s: failed to request from node", ErrorPrefix)
	ErrGotStrangeStatusCode       = fmt.Errorf("%s: got strange status code in request", ErrorPrefix)
	ErrUnExpectedMethodCall       = fmt.Errorf("%s: got unexpected call method", ErrorPrefix)
	ErrFailedJsonUnmarshal        = fmt.Errorf("%s: failed to unmarshing json data", ErrorPrefix)
	ErrFailedConvertTypes         = fmt.Errorf("%s: failed to converting number types", ErrorPrefix)
	ErrOutOfSwitchCases           = fmt.Errorf("%s: out of switch case in router", ErrorPrefix)
	ErrFailedGatheringMiddleData  = fmt.Errorf("%s: failed to gather middle data like orchestrator address or somethings", ErrorPrefix)
	ErrFailedCreateGrpcConnection = fmt.Errorf("%s: failed to create grpc connection", ErrorPrefix)
	ErrFailedGrpcRequest          = fmt.Errorf("%s: failed to grpc request from node", ErrorPrefix)
	ErrFailedBuildingLogger       = fmt.Errorf("%s: failed to build logger by not found workspace string", ErrorPrefix)
)

// TODO: aggregate common errors from api level
