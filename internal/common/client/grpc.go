package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	// Below are necessary imports for the cosmos SDK types
	// Otherwise we would have to manually import the relevant SDK types
	_ "cosmossdk.io/api/cosmos/crypto/ed25519"
	_ "cosmossdk.io/api/cosmos/crypto/secp256k1"
)

type GrpcClient struct {
	client   *grpc.ClientConn
	endpoint string
	logger   *logrus.Logger
	timeout  time.Duration
	headers  map[string]string
}

func (gc *GrpcClient) SetLogger(logger *logrus.Logger) Client {
	gc.logger = logger
	return gc
}

// Basically returns a new GrpcClient instance as it does not
// support changing the endpoint
func (gc *GrpcClient) SetEndpoint(endpoint string) Client {

	var dialOptions []grpc.DialOption

	tlsConf := &tls.Config{
		NextProtos: []string{"h2"}, // only allow HTTP/2
		MinVersion: tls.VersionTLS12,
	}
	gCreds := credentials.NewTLS(tlsConf)

	// Simple handshake check to determine if the server supports TLS
	dialer := &tls.Dialer{
		Config: tlsConf,
	}
	ctx, ctxCancel := context.WithTimeout(context.Background(), gc.timeout)
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	// conn, err := Dialer.DialContext("tcp", endpoint, tlsConf)
	if err != nil {
		var recordHeaderError tls.RecordHeaderError
		if errors.As(err, &recordHeaderError) {
			gc.logger.Warnf("TLS handshake failed for endpoint: %s", endpoint)
			dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

		} else {
			gc.logger.Errorf("error setting up gRPC connection: %s", err.Error())
			ctxCancel()
			return nil
		}
	} else {
		conn.Close()
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(gCreds))
	}
	ctxCancel()
	client, err := grpc.NewClient(endpoint, dialOptions...)
	if err != nil {
		gc.logger.Errorf("error setting up grpc client: %s", err.Error())
		return nil
	}
	gc.client = client
	gc.endpoint = endpoint
	return gc
}

func (gc *GrpcClient) SetHeaders(headers map[string]string) {
	gc.headers = headers
	gc.logger.Warnf("setting headers not implemented in grpc client yet!")
}

func (gc *GrpcClient) GetEndpoint() (string, error) {
	if gc.endpoint == "" {
		gc.logger.Error("grpc endpoint is not set")
		return "", errors.New("endpoint is not set")
	}
	return gc.endpoint, nil
}

// As Get is similar to Post with an empty body, it is implemented as a simple wrapper around Post
func (gc *GrpcClient) Get(ctx context.Context, uri string) ([]byte, error) {
	respJSON, err := gc.Post(ctx, uri, []byte(``))
	if err != nil {
		gc.logger.Errorf("grpc api failed to get: %s", err.Error())
		return nil, err
	}
	return respJSON, nil
}

func (gc *GrpcClient) Post(ctx context.Context, uri string, body ...[]byte) ([]byte, error) {

	var protoResolver CosmosAnyMessageResolver
	// var reflectionHeader metadata.MD
	var respHeader metadata.MD

	if body == nil {
		gc.logger.Errorf("grpc api requires a body for POST request. Use Get method instead.")
		return nil, errors.New("grpc api requires a body for POST request")
	}

	// wakes up the client if it's not ready
	if gc.client.GetState() != connectivity.Ready {
		gc.logger.Info("grpc client is not ready yet. Waking up...")
		gc.client.Connect()
		if !gc.client.WaitForStateChange(ctx, connectivity.Ready) {
			gc.logger.Error("grpc client still not ready after waiting for connection")
			return nil, errors.New("grpc connection not ready")
		}
	}

	refCtx := metadata.NewOutgoingContext(ctx, make(metadata.MD))

	reflectionClient := grpcreflect.NewClientAuto(
		refCtx,
		gc.client,
	)

	descriptor, err := resolveMessage(uri, reflectionClient)
	if err != nil {
		gc.logger.Errorf("grpc api failed to resolve proto message: %s", err.Error())
		return nil, err
	}

	msg, err := createMessage(descriptor, string(body[0]))
	if err != nil {
		gc.logger.Errorf("grpc api failed to create proto message: %s", err.Error())
		return nil, err
	}

	// This only exists to support the current variable syntax
	// e.g. "cosmos.staking.v1beta1.Query.Validators" to actually use the method it
	// must be re-formatted to "/cosmos.staking.v1beta1.Query/Validators"
	response := dynamicpb.NewMessage(descriptor.Output())
	idx := strings.LastIndex(uri, ".")
	fullMethodName := "/" + uri[:idx] + "/" + uri[idx+1:]

	peer := &peer.Peer{}
	gc.logger.Debugf("Invoking method: %s", fullMethodName)
	err = gc.client.Invoke(ctx, fullMethodName, msg, response, grpc.Header(&respHeader), grpc.Peer(peer))
	gc.logger.Debugf(
		"GRPC connection info: Protocol: %s Target: %s Local Address: %s Authentication Type: %s",
		peer.Addr.Network(),
		peer.Addr.String(),
		peer.LocalAddr.String(),
		peer.AuthInfo.AuthType(),
	)
	gc.logger.Debugf("GRPC header returned: %+v", respHeader)
	if err != nil {
		gc.logger.Errorf("failed to invoke grpc request: %s", err.Error())
		return nil, err
	}
	marshaller := protojson.MarshalOptions{
		AllowPartial:    true,
		Indent:          "  ",
		UseProtoNames:   true,
		EmitUnpopulated: false,
		Resolver:        protoResolver,
	}

	respJSON, err := marshaller.Marshal(response)
	if err != nil {
		gc.logger.Errorf("grpc api failed to marshal string with grpc data: %s", err.Error())
		return nil, err
	}
	return respJSON, nil
}

func NewGrpcClient(timeout time.Duration) *GrpcClient {
	gc := &GrpcClient{
		timeout: timeout,
	}
	return gc
}

// Internal non-public helper methods originally part of the
// grpchelper package

// grpc reflection helper methods
func resolveMessage(fullMethodName string, rcli *grpcreflect.Client) (protoreflect.MethodDescriptor, error) {
	// assume that fully-qualified method name cosists of
	// FULL_SERVER_NAME + "." + METHOD_NAME
	// so split the last dot to get service name
	n := strings.LastIndex(fullMethodName, ".")
	if n < 0 {
		return nil, fmt.Errorf("invalid method name: %v", fullMethodName)
	}
	serviceName := fullMethodName[0:n]
	methodName := fullMethodName[n+1:]

	sdesc, err := rcli.ResolveService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service couldn't be resolved: %v: %v", err, serviceName)
	}
	mdesc := sdesc.UnwrapService().Methods().ByName(protoreflect.Name(methodName))

	if mdesc == nil {
		return nil, fmt.Errorf("method couldn't be found")
	}

	return mdesc, nil
}

func createMessage(mdesc protoreflect.MethodDescriptor, inputJsonString string) (*dynamicpb.Message, error) {

	msg := dynamicpb.NewMessage(mdesc.Input())
	if err := protojson.Unmarshal([]byte(inputJsonString), msg); err != nil {
		return nil, fmt.Errorf("unmarshal %v", err)
	}
	return msg, nil
}

// grpc resolver to decode protobuf messages
type CosmosAnyMessageResolver struct {
	protoregistry.MessageTypeResolver
	protoregistry.ExtensionTypeResolver
}

// Every resolver MUST implement the FindMessageByURL method. This message will be called by the
// Marshaller to resolve the message.
func (r CosmosAnyMessageResolver) FindMessageByURL(typeURL string) (protoreflect.MessageType, error) {
	// Only the part of typeUrl after the last slash is relevant.
	mname := typeURL
	if slash := strings.LastIndex(mname, "/"); slash >= 0 {
		mname = mname[slash+1:]
	}

	a, err := protoregistry.GlobalTypes.FindMessageByURL(mname)
	if err != nil {
		return nil, err
	}

	return a, nil
}
