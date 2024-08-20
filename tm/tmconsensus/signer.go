package tmconsensus

import (
	"context"
	"fmt"

	"github.com/rollchains/gordian/gcrypto"
)

// Signer is the tm-aware signer.
// While [gcrypto.Signer] offers a low level interface to sign raw bytes,
// this Signer is aware of tmconsensus types,
// in case the underlying signer needs any additional context
// on what exactly is being signed.
type Signer interface {
	// Prevote and Precommit return the byte slices containing
	// the signing content and signature for a prevote or precommit
	// for a block or nil, as specified by the VoteTarget.
	//
	// The signing content is necessary as part of the return signature,
	// in order to reduce duplicative work elsewhere internal to the consensus engine.
	Prevote(ctx context.Context, vt VoteTarget) (signContent, signature []byte, err error)
	Precommit(ctx context.Context, vt VoteTarget) (signContent, signature []byte, err error)

	// SignProposedBlock sets the Signature field on the proposed block.
	// All other fields on pb must already be populated.
	SignProposedBlock(ctx context.Context, pb *ProposedBlock) error

	// PubKey returns the public key of the signer.
	PubKey() gcrypto.PubKey
}

var _ Signer = PassthroughSigner{}

// PassthroughSigner is a [Signer] that directly generates signatures
// with the given signer and scheme.
type PassthroughSigner struct {
	Signer          gcrypto.Signer
	SignatureScheme SignatureScheme
}

func (s PassthroughSigner) Prevote(ctx context.Context, vt VoteTarget) (
	signContent, signature []byte, err error,
) {
	signContent, err = PrevoteSignBytes(vt, s.SignatureScheme)
	if err != nil {
		return nil, nil, fmt.Errorf("PassthroughSigner.Prevote failed to generate sign bytes: %w", err)
	}

	signature, err = s.Signer.Sign(ctx, signContent)
	if err != nil {
		return nil, nil, fmt.Errorf("PassthroughSigner.Prevote failed to sign prevote bytes: %w", err)
	}

	return signContent, signature, nil
}

func (s PassthroughSigner) Precommit(ctx context.Context, vt VoteTarget) (
	signContent, signature []byte, err error,
) {
	signContent, err = PrecommitSignBytes(vt, s.SignatureScheme)
	if err != nil {
		return nil, nil, fmt.Errorf("PassthroughSigner.Precommit failed to generate sign bytes: %w", err)
	}

	signature, err = s.Signer.Sign(ctx, signContent)
	if err != nil {
		return nil, nil, fmt.Errorf("PassthroughSigner.Precommit failed to sign precommit bytes: %w", err)
	}

	return signContent, signature, nil
}

func (s PassthroughSigner) SignProposedBlock(ctx context.Context, pb *ProposedBlock) error {
	signContent, err := ProposalSignBytes(pb.Block, pb.Round, pb.Annotations, s.SignatureScheme)
	if err != nil {
		return fmt.Errorf("PassthroughSigner.SignProposedBlock failed to generate sign bytes: %w", err)
	}
	sig, err := s.Signer.Sign(ctx, signContent)
	if err != nil {
		return fmt.Errorf("PassthroughSigner.SignProposedBlock failed to sign proposal: %w", err)
	}

	pb.Signature = sig
	return nil
}

func (s PassthroughSigner) PubKey() gcrypto.PubKey {
	return s.Signer.PubKey()
}
