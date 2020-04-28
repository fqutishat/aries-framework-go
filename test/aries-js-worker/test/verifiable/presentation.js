/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

import {newAries, newAriesREST,healthCheck} from "../common.js"
import {environment} from "../environment.js";

const agentControllerApiUrl = `${environment.HTTP_SCHEME}://${environment.USER_HOST}:${environment.USER_API_PORT}`

// verifiable credential
const didName = "alice-did-presentation"
const vc = `
{ 
   "@context":[ 
      "https://www.w3.org/2018/credentials/v1"
   ],
   "id":"http://faber.edu/credentials/1989",
   "type":"VerifiableCredential",
   "credentialSubject":{ 
      "id":"did:example:iuajk1f712ebc6f1c276e12ec21"
   },
   "issuer":{ 
      "id":"did:example:09s12ec712ebc6f1c671ebfeb1f",
      "name":"Faber University"
   },
   "issuanceDate":"2020-01-01T10:54:01Z",
   "credentialStatus":{ 
      "id":"https://example.gov/status/65",
      "type":"CredentialStatusList2017"
   }
}`

const restMode = 'rest'
const wasmMode = 'wasm'


describe("Verifiable Presentation Test", async function () {
    await presentation(newAriesREST(agentControllerApiUrl), restMode)
    await presentation(newAries('demo','demo-agent', [`${environment.HTTP_LOCAL_DID_RESOLVER}`]))
})

async function presentation(newAries, mode = wasmMode) {
    let aries
    let did
    let retries = 10;
    let modePrefix = '[' + mode + '] '

    before(async () => {
        await newAries
            .then(a => {
                aries = a
            })
            .catch(err => new Error(err.message));
    })

    after(() => {
        aries.destroy()
    })

    it(modePrefix + "Alice creates a DID through VDRI", function (done) {
        aries.vdri.createPublicDID({
            method: "sidetree",
            header: '{"alg":"","kid":"","operation":"create"}'
        }).then(
            resp => {
                did = resp.did
                done()
            },
            err => done(err)
        )
    })

    it(modePrefix + "Alice makes sure that the DID is resolvable", async function () {
        let success = false
        for (var i = 0; i < 10; i++) {
            await axios.get(`${environment.HTTP_LOCAL_RESOLVER_URL}/` + did.id)
                .then(function(response) {
                    success = true
                })
                .catch(function(error) {
                    console.log('will try to resolve again : attempt=',i+1);
                });

            if (success) {
                break
            }

            await new Promise(r => setTimeout(r, 2000));
        }
    })

    it(modePrefix + "Alice stores the did generated by her", function (done) {
        aries.vdri.saveDID({
            name: didName,
            did: did
        }).then(
            resp => {
                done()
            },
            err => done(err)
        )
    })

    it(modePrefix + "Alice generates the signed  verifiable presentation to pass it to the employer", async function () {
        const keyset= await aries.kms.createKeySet({keyType: "ED25519"})

        await aries.verifiable.generatePresentation({
            "verifiableCredential": [JSON.parse(vc)],
            "did": did.id,
            "verifiableMethod":did.id+"#"+keyset.keyID,
            "signatureType":"Ed25519Signature2018"
        }).then(
            resp => {
                try {
                    assert.isTrue(resp.verifiablePresentation.type.includes("VerifiablePresentation"))
                    assert.equal(resp.verifiablePresentation.proof.type, "Ed25519Signature2018")
                } catch (err) {
                    assert.fail(err)
                }
            },
            err => assert.fail(err)
    )
    });

    it(modePrefix + "Alice generates the signed  verifiable presentation to pass it to the employer using P-256 key", function (done) {
        aries.verifiable.generatePresentation({
            "verifiableCredential": [JSON.parse(vc)],
            "did" : did.id,
            "didKeyID": did.id + did.publicKey[0].id,
            "privateKey" :"WejGrq3SkHF1YpsdXSCg46FK8vuTDxroA9wh2q1398MUqrpKrFts54j8rLqGfT5Tu8cmG6PVUXUoFWManr4uVEpVFd8ZywoHPV8nBRQTxQXjucdd22nji7ijKG18kuptpArQBrAAo2GLmv8yFtSagkvFrYQ4A8Ti4aafw",
            "keyType" : "P256",
            "signatureType":"JsonWebSignature2020"
        }).then(
            resp => {
                try {
                    assert.isTrue(resp.verifiablePresentation.type.includes("VerifiablePresentation"))
                    assert.equal(resp.verifiablePresentation.proof.type, "JsonWebSignature2020")
                    done()
                } catch (err) {
                    console.log(err);
                    done(err)
                }
            },
            err => done(err)
        )
    });
}