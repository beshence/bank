package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pion/webrtc/v4"
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
}

func Start(
	bankID string,
	token string,
	router http.Handler,
) error {
	ctx := context.Background()

	wsURL := "wss://gateway.beshence.com/api/bank/" + bankID + "/ws?role=bank"
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

	log.Println("[gateway/webrtc] connected to gateway using websocket, waiting for signaling messages")

	t := &Transport{
		WS:     ws,
		Peers:  make(map[string]*Peer),
		Router: router,
	}

	go t.listen()

	return nil
}

func (t *Transport) listen() {
	ctx := context.Background()

	for {
		var msg Message

		err := wsjson.Read(
			ctx,
			t.WS,
			&msg,
		)

		if err != nil {
			panic(err)
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
		panic(err)
	}

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

			log.Println(
				"[gateway/webrtc/"+msg.SessionID+"] opened datachannel:", dc.Label(),
			)

			t.Mu.Lock()

			if peer, ok := t.Peers[msg.SessionID]; ok {
				peer.DC = dc
			}

			t.Mu.Unlock()

			dc.OnMessage(
				func(message webrtc.DataChannelMessage) {

					/*log.Println(
						"[gateway/webrtc/"+msg.SessionID+"] received datachannel message:",
						string(message.Data),
					)*/

					var req Request

					err := json.Unmarshal(
						message.Data,
						&req,
					)

					if err != nil {
						return
					}

					response := HandleEndpoints(
						t.Router,
						req,
					)

					data, err := json.Marshal(
						response,
					)

					if err != nil {
						return
					}

					err = dc.Send(
						data,
					)

					if err != nil {
						fmt.Println(
							"send error:",
							err,
						)
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
		panic(err)
	}
}
