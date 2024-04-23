package tmintegration

import (
	"context"

	"github.com/rollchains/gordian/gcrypto"
	"github.com/rollchains/gordian/tm/tmconsensus"
	"github.com/rollchains/gordian/tm/tmconsensus/tmconsensustest"
	"github.com/rollchains/gordian/tm/tmstore"
	"github.com/rollchains/gordian/tm/tmstore/tmmemstore"
)

// InmemStoreFactory is meant to be embedded in another [tmintegration.Factory]
// to provide in-memory implementations of stores.
type InmemStoreFactory struct{}

func (f InmemStoreFactory) NewActionStore(ctx context.Context, idx int) (tmstore.ActionStore, error) {
	return tmmemstore.NewActionStore(), nil
}

func (f InmemStoreFactory) NewBlockStore(ctx context.Context, idx int) (tmstore.BlockStore, error) {
	return tmmemstore.NewBlockStore(), nil
}

func (f InmemStoreFactory) NewFinalizationStore(ctx context.Context, idx int) (tmstore.FinalizationStore, error) {
	return tmmemstore.NewFinalizationStore(), nil
}

func (f InmemStoreFactory) NewMirrorStore(ctx context.Context, idx int) (tmstore.MirrorStore, error) {
	return tmmemstore.NewMirrorStore(), nil
}

func (f InmemStoreFactory) NewRoundStore(ctx context.Context, idx int) (tmstore.RoundStore, error) {
	return tmmemstore.NewRoundStore(), nil
}

func (f InmemStoreFactory) NewValidatorStore(ctx context.Context, idx int, hs tmconsensus.HashScheme) (tmstore.ValidatorStore, error) {
	return tmmemstore.NewValidatorStore(hs), nil
}

type InmemSchemeFactory struct{}

func (f InmemSchemeFactory) HashScheme(ctx context.Context, idx int) (tmconsensus.HashScheme, error) {
	return tmconsensustest.SimpleHashScheme{}, nil
}

func (f InmemSchemeFactory) SignatureScheme(ctx context.Context, idx int) (tmconsensus.SignatureScheme, error) {
	return tmconsensustest.SimpleSignatureScheme{}, nil
}

func (f InmemSchemeFactory) CommonMessageSignatureProofScheme(ctx context.Context, idx int) (gcrypto.CommonMessageSignatureProofScheme, error) {
	return gcrypto.SimpleCommonMessageSignatureProofScheme, nil
}