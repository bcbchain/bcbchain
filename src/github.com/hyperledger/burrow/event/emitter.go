// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package event

import (
	"common/log"
	"context"
	"github.com/tendermint/tmlibs/common"
	"math/rand"

	"github.com/hyperledger/burrow/event/pubsub"
	"github.com/hyperledger/burrow/event/query"

	"github.com/tmthrgd/go-hex"
)

const DefaultEventBufferCapacity = 2 << 10

// TODO: manage the creation, closing, and draining of channels behind subscribe rather than only closing.
// stop one subscriber from blocking everything!

// Emitter has methods for working with events
type Emitter struct {
	common.BaseService
	pubsubServer *pubsub.Server
	logger       log.Logger
}

// NewEmitter initializes an emitter struct with a pubsubServer
func NewEmitter() *Emitter {
	pubsubServer := pubsub.NewServer(pubsub.BufferCapacity(DefaultEventBufferCapacity))
	pubsubServer.BaseService = *common.NewBaseService(nil, "Emitter", pubsubServer)
	pubsubServer.Start()
	return &Emitter{
		pubsubServer: pubsubServer,
	}
}

// SetLogger attaches a log handler to this emitter
func (em *Emitter) SetLogger(logger log.Logger) {
	em.logger = logger.With("Component", "Events")
}

// Shutdown stops the pubsubServer
func (em *Emitter) Shutdown(ctx context.Context) error {
	return em.pubsubServer.Stop()
}

// Publish tells the emitter to publish with the given message and tags
func (em *Emitter) Publish(ctx context.Context, message interface{}, tags query.Tagged) error {
	if em == nil || em.pubsubServer == nil {
		return nil
	}
	return em.pubsubServer.PublishWithTags(ctx, message, tags)
}

// Subscribe tells the emitter to listen for messages on the given query
func (em *Emitter) Subscribe(ctx context.Context, subscriber string, queryable query.Queryable, bufferSize int) (<-chan interface{}, error) {
	qry, err := queryable.Query()
	if err != nil {
		return nil, err
	}
	return em.pubsubServer.Subscribe(ctx, subscriber, qry, bufferSize)
}

// Unsubscribe tells the emitter to stop listening for said messages
func (em *Emitter) Unsubscribe(ctx context.Context, subscriber string, queryable query.Queryable) error {
	pubsubQuery, err := queryable.Query()
	if err != nil {
		return nil
	}
	return em.pubsubServer.Unsubscribe(ctx, subscriber, pubsubQuery)
}

// UnsubscribeAll just stop listening for all messages
func (em *Emitter) UnsubscribeAll(ctx context.Context, subscriber string) error {
	return em.pubsubServer.UnsubscribeAll(ctx, subscriber)
}

// ***************
// Helper function

func GenSubID() string {
	bs := make([]byte, 32)
	rand.Read(bs)
	return hex.EncodeUpperToString(bs)
}
