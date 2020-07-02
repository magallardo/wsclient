package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
	"github.com/sacOO7/gowebsocket"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &Output{})

func init() {
	trigger.Register(&Trigger{}, &Factory{})
}

// Factory for creating triggers
type Factory struct {
}

// Metadata implements trigger.Factory.Metadata
func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// Trigger trigger struct
type Trigger struct {
	runner   action.Runner
	socket   gowebsocket.Socket
	settings *Settings
	logger   log.Logger
	config   *trigger.Config
}

type sub struct {
	Subscribe string `json:"subscribe"`
}

// New implements trigger.Factory.New
func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	s := &Settings{}
	err := metadata.MapToStruct(config.Settings, s, true)
	if err != nil {
		return nil, err
	}

	return &Trigger{settings: s, config: config}, nil
}

// Initialize initializes the trigger
func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	t.logger = ctx.Logger()
	urlSetting := t.config.Settings["url"]
	if urlSetting == nil || urlSetting.(string) == "" {
		return fmt.Errorf("server url not provided")
	}

	subPathSetting := t.config.Settings["subPath"]
	if subPathSetting == nil || subPathSetting.(string) == "" {
		return fmt.Errorf("subscription path not provided")
	}

	tokenSetting := t.config.Settings["token"]
	if tokenSetting == nil || tokenSetting.(string) == "" {
		return fmt.Errorf("authorization token not provided")
	}

	url := urlSetting.(string)
	subPath := subPathSetting.(string)
	token := "Token " + tokenSetting.(string)
	t.logger.Infof("dialing websocket endpoint[%s]...", url)
	t.logger.Infof("subscribing to path: [%s]", subPath)
	t.logger.Infof("authorization token: [%s]", token)

	socket := gowebsocket.New(url)
	socket.RequestHeader.Set("Authorization", token)

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		t.logger.Errorf("Received websocket connect error")
	}

	socket.OnConnected = func(socket gowebsocket.Socket) {
		// Subscribe to locations for clients on a map
		subReq := &sub{
			Subscribe: subPath}

		jsonSubReq, _ := json.Marshal(subReq)

		socket.SendText(string(jsonSubReq))
	}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {

		t.logger.Infof("Received message: [%s]", message)

		for _, handler := range ctx.GetHandlers() {
			out := &Output{}
			out.Content = message
			_, err := handler.Handle(context.Background(), out)
			if err != nil {
				t.logger.Errorf("Run action  failed [%s] ", err)
			}
		}
	}

	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		t.logger.Infof("Disconnected from server: [%s]", "error")
		return
	}

	socket.Connect()
	t.socket = socket

	return nil
}

// Start starts the trigger
func (t *Trigger) Start() error {
	return nil
}

// Stop stops the trigger
func (t *Trigger) Stop() error {
	t.socket.Close()
	return nil
}
