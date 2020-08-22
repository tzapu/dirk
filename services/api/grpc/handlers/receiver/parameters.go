// Copyright © 2020 Attestant Limited.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package receiver

import (
	"errors"

	"github.com/attestantio/dirk/services/peers"
	"github.com/attestantio/dirk/services/process"
	"github.com/rs/zerolog"
)

type parameters struct {
	logLevel zerolog.Level
	process  process.Service
	peers    peers.Service
}

// Parameter is the interface for handler parameters.
type Parameter interface {
	apply(*parameters)
}

type parameterFunc func(*parameters)

func (f parameterFunc) apply(p *parameters) {
	f(p)
}

// WithLogLevel sets the log level for the handler.
func WithLogLevel(logLevel zerolog.Level) Parameter {
	return parameterFunc(func(p *parameters) {
		p.logLevel = logLevel
	})
}

// WithPeers sets the peers service for the handler.
func WithPeers(peers peers.Service) Parameter {
	return parameterFunc(func(p *parameters) {
		p.peers = peers
	})
}

// WithProcess sets the process service for the handler.
func WithProcess(process process.Service) Parameter {
	return parameterFunc(func(p *parameters) {
		p.process = process
	})
}

// parseAndCheckParameters parses and checks parameters to ensure that mandatory parameters are present and correct.
func parseAndCheckParameters(params ...Parameter) (*parameters, error) {
	parameters := parameters{
		logLevel: zerolog.GlobalLevel(),
	}
	for _, p := range params {
		if params != nil {
			p.apply(&parameters)
		}
	}

	if parameters.peers == nil {
		return nil, errors.New("no peers specified")
	}
	if parameters.process == nil {
		return nil, errors.New("no process specified")
	}

	return &parameters, nil
}