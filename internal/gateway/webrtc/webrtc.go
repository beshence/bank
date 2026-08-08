package webrtc

import (
	"bank/internal/gateway"
	"bank/internal/settings"
	"context"
	"crypto/mlkem"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign/mldsa/mldsa87"
	"github.com/cloudflare/circl/sign/slhdsa"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pion/webrtc/v4"
	"golang.org/x/crypto/hkdf"
	"gorm.io/gorm"
)

type WebSocketMessage struct {
	SessionID           string `json:"session_id,omitempty"`
	Type                string `json:"type"`
	EncapsulationKeyB64 string `json:"ek,omitempty"`
	CiphertextB64       string `json:"ct,omitempty"`
	SignatureB64        string `json:"sig,omitempty"`
}

type SignalingMessage struct {
	SessionID     string  `json:"session_id,omitempty"`
	Type          string  `json:"type"`
	SDP           string  `json:"sdp,omitempty"`
	Candidate     string  `json:"candidate,omitempty"`
	SDPMid        *string `json:"sdpmid,omitempty"`
	SDPMLineIndex *uint16 `json:"sdpmlineindex,omitempty"`
}

type Peer struct {
	SessionID string
	C2BKey    []byte
	B2CKey    []byte
	PC        *webrtc.PeerConnection
	DC        *webrtc.DataChannel
}

type Transport struct {
	WS *websocket.Conn

	Peers map[string]*Peer

	Router http.Handler

	Mu sync.RWMutex

	ctx context.Context
}

func Start(
	db *gorm.DB,
	router http.Handler,
) error {
	t := &Transport{
		Peers:  make(map[string]*Peer),
		Router: router,
		ctx:    context.Background(),
	}

	go t.reconnectLoop(db)

	return nil
}

func (t *Transport) reconnectLoop(db *gorm.DB) {
	delay := time.Second

	for {
		err := t.connect(db)

		if err == nil {
			// listen ended so WS disconnected
			log.Println("[gateway/webrtc] websocket disconnected")
			delay = time.Second
		} else {
			log.Println("[gateway/webrtc] connection error:", err)
		}

		// exponential backoff

		jitter := time.Duration(mathrand.Intn(500)) * time.Millisecond

		wait := min(delay+jitter, time.Minute)

		log.Println("[gateway/webrtc] reconnecting in", wait)

		time.Sleep(wait)

		delay *= 2
	}
}

func (t *Transport) connect(db *gorm.DB) error {
	ctx := context.Background()
	bankID := settings.GetBankID(db)

	wsURL := "wss://gateway.beshence.com/api/bank/" + bankID + "/ws?role=bank"

	token, err := gateway.GetGatewayToken(db)

	if err != nil {
		return err
	}

	ws, _, err := websocket.Dial(
		ctx,
		wsURL,
		&websocket.DialOptions{
			HTTPHeader: map[string][]string{
				"Authorization": {
					"Bearer " + token,
				},
			},
		},
	)

	if err != nil {
		return err
	}

	t.Mu.Lock()
	t.WS = ws
	t.Mu.Unlock()

	log.Println("[gateway/webrtc] connected")

	err = t.listen(db)

	_ = ws.Close(
		websocket.StatusNormalClosure,
		"reconnecting",
	)

	return err
}

func (t *Transport) listen(db *gorm.DB) error {
	ctx := context.Background()

	for {
		var wsMsg WebSocketMessage

		err := wsjson.Read(ctx, t.WS, &wsMsg)

		if err != nil {
			log.Println("[gateway/webrtc] websocket closed:", err)
			return err
		}

		switch wsMsg.Type {
		case "client_hello_v1":
			// 1. generate ML-DSA as this is more resource effective unlike SLH-DSA

			mlDsaPublicKey, mlDsaPrivateKey, err := mldsa87.GenerateKey(cryptorand.Reader)

			if err != nil {
				log.Fatal(err)
				return err
			}

			mlDsaPublicKeyBytes, err := mlDsaPublicKey.MarshalBinary()
			if err != nil {
				log.Fatal(err)
				return err
			}

			// 2. create its signature so we can tie ML-DSA to SLH-DSA

			domain := "BESHENCE-BANK-MLDSA-KEY-V1"

			message := make([]byte, 0, len(domain)+len(mlDsaPublicKeyBytes))

			message = append(
				message,
				[]byte(domain)...,
			)

			message = append(
				message,
				mlDsaPublicKeyBytes...,
			)

			bankPrivateKey, _ := settings.GetBankPrivateKey(db)

			if err != nil {
				log.Fatal(err)
				return err
			}

			mlDsaPKSignature, err := slhdsa.SignRandomized(
				&bankPrivateKey,
				cryptorand.Reader,
				slhdsa.NewMessage(message),
				nil,
			)

			if err != nil {
				log.Fatal(err)
				return err
			}

			print(mlDsaPrivateKey)
			print(mlDsaPKSignature)

			// 3. create ciphertext and shared secret from the client's encapsulation key

			encapsulationKeyBytes, err := base64.RawURLEncoding.DecodeString(wsMsg.EncapsulationKeyB64)

			if err != nil {
				log.Fatal(err)
				return err
			}

			encapsulationKey, err := mlkem.NewEncapsulationKey1024(encapsulationKeyBytes)

			if err != nil {
				log.Fatal(err)
				return err
			}

			sharedSecret, ciphertext := encapsulationKey.Encapsulate()

			if err != nil {
				log.Fatal(err)
				return err
			}

			// 4. sign encryption context

			domain = "BESHENCE-BANK-SIGNALING-SIGN-CONTEXT-V1"

			message = make([]byte, 0, len(domain)+len(encapsulationKeyBytes)+len(ciphertext))

			message = append(
				message,
				[]byte(domain)...,
			)

			message = append(
				message,
				encapsulationKeyBytes...,
			)

			message = append(
				message,
				ciphertext...,
			)

			signature := make([]byte, mldsa87.SignatureSize)

			err = mldsa87.SignTo(
				mlDsaPrivateKey,
				message,
				nil,
				true,
				signature,
			)

			if err != nil {
				log.Fatal(err)
				return err
			}

			// 5. derive keys and preserve them

			c2bKey, b2cKey, err := deriveKeys(sharedSecret)

			t.Mu.Lock()

			t.Peers[wsMsg.SessionID] = &Peer{
				SessionID: wsMsg.SessionID,
				C2BKey:    c2bKey,
				B2CKey:    b2cKey,
			}

			t.Mu.Unlock()

			// 6. send server hello

			wsjson.Write(
				context.Background(),
				t.WS,
				WebSocketMessage{
					SessionID:     wsMsg.SessionID,
					Type:          "server_hello_v1",
					CiphertextB64: base64.RawURLEncoding.EncodeToString(ciphertext),
					SignatureB64:  base64.RawURLEncoding.EncodeToString(signature),
					//Candidate:     candidate.ToJSON().Candidate,
					//SDPMid:        candidate.ToJSON().SDPMid,
					//SDPMLineIndex: candidate.ToJSON().SDPMLineIndex,
				},
			)

			break
		case "encrypted_v1":
			break
		}
		/*
			switch msg.Type {
			case "offer":
				t.handleOffer(msg)
			case "candidate":
				t.handleCandidate(msg)
			}
		*/
	}
}

func deriveKeys(sharedSecret []byte) (
	c2bKey []byte,
	b2cKey []byte,
	err error,
) {
	sessionKey := make([]byte, 32)

	h := hkdf.New(
		func() hash.Hash {
			return sha256.New()
		},
		sharedSecret,
		nil,
		[]byte("BESHENCE-BANK-SIGNALING-SESSION-KEY-V1"),
	)

	if _, err := io.ReadFull(h, sessionKey); err != nil {
		return nil, nil, err
	}

	c2bKey = make([]byte, 32)

	hClient := hkdf.New(
		func() hash.Hash {
			return sha256.New()
		},
		sessionKey,
		nil,
		[]byte("BESHENCE-BANK-SIGNALING-C2B-KEY-V1"),
	)

	if _, err := io.ReadFull(hClient, c2bKey); err != nil {
		return nil, nil, err
	}

	b2cKey = make([]byte, 32)

	hBank := hkdf.New(
		func() hash.Hash {
			return sha256.New()
		},
		sessionKey,
		nil,
		[]byte("BESHENCE-BANK-SIGNALING-B2C-KEY-V1"),
	)

	if _, err := io.ReadFull(hBank, b2cKey); err != nil {
		return nil, nil, err
	}

	return c2bKey, b2cKey, nil
}

/*func encryptJSON(
	key []byte,
	data any,
) ([]byte, error) {

	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())

	if _, err := cryptorand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(
		nil,
		nonce,
		plaintext,
		nil,
	)

	// отправляем nonce + ciphertext
	return append(
		nonce,
		ciphertext...,
	), nil
}

func decryptJSON(
	key []byte,
	packet []byte,
	out any,
) error {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return err
	}

	nonceSize := aead.NonceSize()

	nonce := packet[:nonceSize]
	ciphertext := packet[nonceSize:]

	plaintext, err := aead.Open(
		nil,
		nonce,
		ciphertext,
		nil,
	)

	if err != nil {
		return err
	}

	return json.Unmarshal(
		plaintext,
		out,
	)
}*/

func (t *Transport) handleOffer(msg SignalingMessage) {
	log.Println("[gateway/webrtc/" + msg.SessionID + "] got signaling offer")

	pc, err := webrtc.NewPeerConnection(
		webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		},
	)

	if err != nil {
		log.Println("create peer connection:", err)
		return
	}

	success := false

	defer func() {
		if !success {
			pc.Close()
		}
	}()

	pc.OnICECandidate(
		func(candidate *webrtc.ICECandidate) {
			if candidate == nil {
				return
			}

			wsjson.Write(
				context.Background(),
				t.WS,
				SignalingMessage{
					SessionID:     msg.SessionID,
					Type:          "candidate",
					Candidate:     candidate.ToJSON().Candidate,
					SDPMid:        candidate.ToJSON().SDPMid,
					SDPMLineIndex: candidate.ToJSON().SDPMLineIndex,
				},
			)
		},
	)

	pc.OnConnectionStateChange(
		func(state webrtc.PeerConnectionState) {
			log.Println(
				"[gateway/webrtc/"+msg.SessionID+"] client changed connection state to", state,
			)
		},
	)

	pc.OnDataChannel(
		func(dc *webrtc.DataChannel) {
			log.Println("[gateway/webrtc/"+msg.SessionID+"] opened datachannel:", dc.Label())

			t.Mu.Lock()

			if peer, ok := t.Peers[msg.SessionID]; ok {
				peer.DC = dc
			}

			t.Mu.Unlock()

			dc.OnMessage(
				func(message webrtc.DataChannelMessage) {
					var req Request

					err := json.Unmarshal(message.Data, &req)

					if err != nil {
						return
					}

					response := HandleEndpoints(t.Router, req)

					data, err := json.Marshal(response)

					if err != nil {
						return
					}

					err = dc.Send(data)

					if err != nil {
						fmt.Println("send error:", err)
					}

				},
			)
		},
	)

	t.Mu.Lock()

	t.Peers[msg.SessionID].PC = pc

	t.Mu.Unlock()

	err = pc.SetRemoteDescription(
		webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  msg.SDP,
		},
	)

	if err != nil {
		return
	}

	answer, err := pc.CreateAnswer(nil)

	if err != nil {
		return
	}

	err = pc.SetLocalDescription(answer)

	if err != nil {
		return
	}

	err = wsjson.Write(
		context.Background(),
		t.WS,
		SignalingMessage{
			SessionID: msg.SessionID,
			Type:      "answer",
			SDP:       answer.SDP,
		},
	)

	if err != nil {
		log.Println("send answer error:", err)
	}

	success = true

	log.Println("[gateway/webrtc/" + msg.SessionID + "] signaling answer sent")
}

func (t *Transport) handleCandidate(msg SignalingMessage) {
	t.Mu.RLock()

	peer, ok := t.Peers[msg.SessionID]

	t.Mu.RUnlock()

	if !ok {
		return
	}

	err := peer.PC.AddICECandidate(
		webrtc.ICECandidateInit{
			Candidate:     msg.Candidate,
			SDPMid:        msg.SDPMid,
			SDPMLineIndex: msg.SDPMLineIndex,
		},
	)

	if err != nil {
		log.Println("add ice candidate:", err)
		return
	}
}
