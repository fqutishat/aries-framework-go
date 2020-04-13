/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/proof"
	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/signer"
	"github.com/hyperledger/aries-framework-go/pkg/doc/signature/verifier"
)

const (
	resolveIDParts = 2
)

// signatureSuite encapsulates signature suite methods required for signing documents
type signatureSuite interface {

	// GetCanonicalDocument will return normalized/canonical version of the document
	GetCanonicalDocument(doc map[string]interface{}) ([]byte, error)

	// GetDigest returns document digest
	GetDigest(doc []byte) []byte

	// Accept registers this signature suite with the given signature type
	Accept(signatureType string) bool

	// CompactProof indicates weather to compact the proof doc before canonization
	CompactProof() bool
}

// VerifierSignatureSuite encapsulates the methods required for verifying a documents.
type VerifierSignatureSuite interface {
	signatureSuite

	// Verify will verify signature against public key
	Verify(pubKey *verifier.PublicKey, doc []byte, signature []byte) error
}

// SignerSignatureSuite encapsulates the methods required for signing a documents.
type SignerSignatureSuite interface {
	signatureSuite

	// Sign will sign JSON LD document
	Sign(jsonLdDoc []byte) ([]byte, error)
}

type keyResolverAdapter struct {
	pubKeyFetcher PublicKeyFetcher
}

func (k *keyResolverAdapter) Resolve(id string) (*verifier.PublicKey, error) {
	// id will contain didID#keyID
	idSplit := strings.Split(id, "#")
	if len(idSplit) != resolveIDParts {
		return nil, fmt.Errorf("wrong id %s to resolve", idSplit)
	}
	// idSplit[0] is didID
	// idSplit[1] is keyID
	pubKey, err := k.pubKeyFetcher(idSplit[0], fmt.Sprintf("#%s", idSplit[1]))
	if err != nil {
		return nil, err
	}

	return pubKey, nil
}

// SignatureRepresentation is a signature value holder type (e.g. "proofValue" or "jws").
type SignatureRepresentation int

const (
	// SignatureProofValue uses "proofValue" field in a Proof to put/read a digital signature.
	SignatureProofValue SignatureRepresentation = iota

	// SignatureJWS uses "jws" field in a Proof as an element for representation of detached JSON Web Signatures.
	SignatureJWS
)

// LinkedDataProofContext holds options needed to build a Linked Data Proof.
type LinkedDataProofContext struct {
	SignatureType           string                  // required
	Suite                   SignerSignatureSuite    // required
	SignatureRepresentation SignatureRepresentation // required
	Created                 *time.Time              // optional
	VerificationMethod      string                  // optional
}

func checkLinkedDataProof(jsonldBytes []byte, suite VerifierSignatureSuite, pubKeyFetcher PublicKeyFetcher) error {
	documentVerifier := verifier.New(&keyResolverAdapter{pubKeyFetcher}, suite)

	err := documentVerifier.Verify(jsonldBytes)
	if err != nil {
		return fmt.Errorf("check linked data proof: %w", err)
	}

	return nil
}

type rawProof struct {
	Proof json.RawMessage `json:"proof,omitempty"`
}

// addLinkedDataProof adds a new proof to the JSON-LD document (VC or VP). It returns a slice
// of the proofs which were already present appended with a newly created proof.
func addLinkedDataProof(context *LinkedDataProofContext, jsonldBytes []byte) ([]Proof, error) {
	documentSigner := signer.New(context.Suite)

	vcWithNewProofBytes, err := documentSigner.Sign(mapContext(context), jsonldBytes)
	if err != nil {
		return nil, fmt.Errorf("add linked data proof: %w", err)
	}

	// Get a proof from json-ld document.
	var rProof rawProof

	err = json.Unmarshal(vcWithNewProofBytes, &rProof)
	if err != nil {
		return nil, err
	}

	proofs, err := decodeProof(rProof.Proof)
	if err != nil {
		return nil, err
	}

	return proofs, nil
}

func mapContext(context *LinkedDataProofContext) *signer.Context {
	return &signer.Context{
		SignatureType:           context.SignatureType,
		SignatureRepresentation: proof.SignatureRepresentation(context.SignatureRepresentation),
		Created:                 context.Created,
		VerificationMethod:      context.VerificationMethod,
	}
}
