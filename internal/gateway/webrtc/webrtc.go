package webrtc

import (
	"bank/internal/gateway"
	"bank/internal/settings"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pion/webrtc/v4"
	"gorm.io/gorm"
)

type Message struct {
	SessionID     string  `json:"session_id,omitempty"`
	Type          string  `json:"type"`
	SDP           string  `json:"sdp,omitempty"`
	Candidate     string  `json:"candidate,omitempty"`
	SDPMid        *string `json:"sdpmid,omitempty"`
	SDPMLineIndex *uint16 `json:"sdpmlineindex,omitempty"`
}

type Peer struct {
	SessionID string
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

		jitter := time.Duration(rand.Intn(500)) * time.Millisecond

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

	err = t.listen()

	_ = ws.Close(
		websocket.StatusNormalClosure,
		"reconnecting",
	)

	return err
}

func (t *Transport) listen() error {
	ctx := context.Background()

	for {
		var msg Message

		err := wsjson.Read(ctx, t.WS, &msg)

		if err != nil {
			log.Println("[gateway/webrtc] websocket closed:", err)
			return err
		}

		switch msg.Type {
		case "offer":
			t.handleOffer(msg)
		case "candidate":
			t.handleCandidate(msg)
		}
	}
}

func (t *Transport) handleOffer(msg Message) {
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
				Message{
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

	t.Peers[msg.SessionID] = &Peer{
		SessionID: msg.SessionID,
		PC:        pc,
	}

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
		Message{
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

func (t *Transport) handleCandidate(msg Message) {
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
