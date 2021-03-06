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

package standard

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/attestantio/dirk/rules"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	e2types "github.com/wealdtech/go-eth2-types/v2"
)

type signBeaconAttestationState struct {
	SourceEpoch int64
	TargetEpoch int64
}

func (s *Service) OnSignBeaconAttestation(ctx context.Context, metadata *rules.ReqMetadata, req *rules.SignBeaconAttestationData) rules.Result {
	span, _ := opentracing.StartSpanFromContext(ctx, "rules.OnSignBeaconAttestation")
	defer span.Finish()
	log := log.With().Str("client", metadata.Client).Str("account", metadata.Account).Str("rule", "sign beacon attestation").Logger()

	// The request must have the appropriate domain.
	if !bytes.Equal(req.Domain[0:4], e2types.DomainBeaconAttester[:]) {
		log.Warn().Msg("Not approving non-beacon attestation due to incorrect domain")
		return rules.DENIED
	}

	// Fetch state from previous signings.
	state, err := s.fetchSignBeaconAttestationState(ctx, metadata.PubKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch state for beacon attestation")
		return rules.FAILED
	}
	sourceEpoch := req.Source.Epoch
	targetEpoch := req.Target.Epoch

	// The request target epoch must be greater than the request source epoch (or both 0).
	if (sourceEpoch != 0 || targetEpoch != 0) && (targetEpoch <= sourceEpoch) {
		log.Warn().
			Uint64("sourceEpoch", sourceEpoch).
			Uint64("targetEpoch", targetEpoch).
			Msg("Request target epoch equal to or lower than request source epoch")
		return rules.DENIED
	}

	if state.TargetEpoch != -1 {
		// The request target epoch must be greater than the previous request target epoch.
		if int64(targetEpoch) <= state.TargetEpoch {
			log.Warn().
				Int64("previousTargetEpoch", state.TargetEpoch).
				Uint64("targetEpoch", targetEpoch).
				Msg("Request target epoch equal to or lower than previous signed target epoch")
			return rules.DENIED
		}
	}

	if state.SourceEpoch != -1 {
		// The request source epoch must be greater than or equal to the previous request source epoch.
		if int64(sourceEpoch) < state.SourceEpoch {
			log.Warn().
				Int64("previousSourceEpoch", state.SourceEpoch).
				Uint64("sourceEpoch", sourceEpoch).
				Msg("Request source epoch lower than previous signed source epoch")
			return rules.DENIED
		}
	}

	state.SourceEpoch = int64(sourceEpoch)
	state.TargetEpoch = int64(targetEpoch)
	if err = s.storeSignBeaconAttestationState(ctx, metadata.PubKey, state); err != nil {
		log.Error().Err(err).Msg("Failed to store state for beacon attestation")
		return rules.FAILED
	}

	return rules.APPROVED
}

func (s *Service) fetchSignBeaconAttestationState(ctx context.Context, pubKey []byte) (*signBeaconAttestationState, error) {
	state := &signBeaconAttestationState{
		SourceEpoch: -1,
		TargetEpoch: -1,
	}
	key := make([]byte, len(pubKey)+len(actionSignBeaconAttestation))
	copy(key, pubKey)
	copy(key[len(pubKey):], actionSignBeaconAttestation)
	data, err := s.store.Fetch(ctx, key)
	if err == nil {
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		err = dec.Decode(&state)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode state")
		}
	} else if err.Error() != "not found" {
		return nil, err
	}
	return state, nil
}

func (s *Service) storeSignBeaconAttestationState(ctx context.Context, pubKey []byte, state *signBeaconAttestationState) error {
	key := make([]byte, len(pubKey)+len(actionSignBeaconAttestation))
	copy(key, pubKey)
	copy(key[len(pubKey):], actionSignBeaconAttestation)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(state); err != nil {
		return err
	}
	value := buf.Bytes()
	return s.store.Store(ctx, key, value)
}
